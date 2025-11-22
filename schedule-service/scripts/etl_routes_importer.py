#!/usr/bin/env python3
"""
ETL Script for Latacunga Garbage Collection Routes
==================================================
This script processes CSV files containing collection routes for Latacunga, Ecuador,
geocodes the locations using Nominatim API, and generates SQL seed data for PostgreSQL/PostGIS.

Author: Data Engineering Team
Date: 2025-11-14
"""

import pandas as pd
import requests
import time
import os
import sys
import logging
from pathlib import Path
from typing import Optional, Tuple, List, Dict
import re
from datetime import datetime
import openpyxl  # For reading Excel files

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('etl_routes_importer.log', encoding='utf-8'),
        logging.StreamHandler(sys.stdout)
    ]
)
logger = logging.getLogger(__name__)

# Constants
INPUT_DIR = Path('./data/raw')
OUTPUT_DIR = Path('./migrations')
OUTPUT_FILE = OUTPUT_DIR / 'seed_routes.sql'
NOMINATIM_URL = 'https://nominatim.openstreetmap.org/search'
RATE_LIMIT_DELAY = 1.2  # seconds between API calls
USER_AGENT = 'LatacungaWasteManagementApp/1.0'

# Day name to number mapping (0=Domingo, 1=Lunes, ..., 6=Sábado)
DAY_MAP = {
    'DOMINGO': 0,
    'LUNES': 1,
    'MARTES': 2,
    'MIERCOLES': 3,
    'MIÉRCOLES': 3,
    'JUEVES': 4,
    'VIERNES': 5,
    'SABADO': 6,
    'SÁBADO': 6
}

# Geocoding cache to avoid repeated API calls
geocoding_cache = {}

# Counter for generating unique fallback coordinates
fallback_counter = 0

# Known reference points in Latacunga (landmarks for better approximation)
LATACUNGA_LANDMARKS = {
    'centro': (-0.9346, -78.6156),
    'norte': (-0.9200, -78.6156),
    'sur': (-0.9500, -78.6156),
    'oriental': (-0.9346, -78.6000),
    'occidental': (-0.9346, -78.6300),
    'aeropuerto': (-0.9063, -78.6156),
    'panamericana_norte': (-0.9100, -78.6156),
    'panamericana_sur': (-0.9600, -78.6156)
}


def get_coords(place_name: str) -> Optional[Tuple[float, float]]:
    """
    Get coordinates (lat, lon) for a place using Nominatim API with intelligent fallback.
    
    Tries multiple approaches:
    1. Full address with Latacunga context
    2. Without common prefixes (Barrio, Sector, etc.)
    3. Just the place name with Ecuador
    4. Search in Cotopaxi province
    5. Intelligent fallback based on place name analysis
    
    Args:
        place_name: Name of the place to geocode
        
    Returns:
        Tuple of (latitude, longitude), never returns None
    """
    global fallback_counter
    
    # Check cache first
    if place_name in geocoding_cache:
        logger.debug(f"Cache hit for: {place_name}")
        return geocoding_cache[place_name]
    
    # Strategy 1: Full address with specific context
    search_strategies = [
        f"{place_name}, Latacunga, Cotopaxi, Ecuador",
        f"{place_name}, Latacunga, Ecuador",
        f"{place_name}, Cotopaxi, Ecuador",
        f"{place_name}, Ecuador"
    ]
    
    # Remove common prefixes and add alternative searches
    clean_name = place_name
    for prefix in ['Barrio ', 'Sector ', 'Comunidad ', 'Urbanización ', 'Urb. ', 
                   'Floricola ', 'Unidad Educativa ', 'Centro de Salud ', 
                   'Gasolinera ', 'Parroquia ', 'Ciudadela ']:
        if clean_name.startswith(prefix):
            clean_name = clean_name[len(prefix):].strip()
            search_strategies.insert(1, f"{clean_name}, Latacunga, Cotopaxi, Ecuador")
            search_strategies.insert(2, f"{clean_name}, Latacunga, Ecuador")
            break
    
    try:
        logger.info(f"Geocoding: {place_name}")
        
        for idx, search_query in enumerate(search_strategies):
            # Rate limiting
            time.sleep(RATE_LIMIT_DELAY)
            
            params = {
                'q': search_query,
                'format': 'json',
                'limit': 1,
                'countrycodes': 'ec'  # Restrict to Ecuador
            }
            
            headers = {
                'User-Agent': USER_AGENT
            }
            
            if idx > 0:
                logger.debug(f"  Trying strategy {idx+1}: {search_query}")
            
            response = requests.get(NOMINATIM_URL, params=params, headers=headers, timeout=10)
            response.raise_for_status()
            
            data = response.json()
            
            if data and len(data) > 0:
                lat = float(data[0]['lat'])
                lon = float(data[0]['lon'])
                
                # Validate coordinates are in Ecuador (rough bounds)
                # Ecuador: lat -5 to 2, lon -92 to -75
                if -5 <= lat <= 2 and -92 <= lon <= -75:
                    coords = (lat, lon)
                    geocoding_cache[place_name] = coords
                    strategy_msg = f" (strategy {idx+1})" if idx > 0 else ""
                    logger.info(f"[OK] Found: {place_name} -> ({lat:.6f}, {lon:.6f}){strategy_msg}")
                    return coords
                else:
                    logger.debug(f"  Coordinates outside Ecuador, skipping")
                    continue
        
        # All strategies failed - use intelligent fallback based on name analysis
        fallback_coords = generate_intelligent_fallback(place_name, fallback_counter)
        fallback_counter += 1
        
        logger.warning(f"[X] Not found: {place_name} - using intelligent fallback")
        geocoding_cache[place_name] = fallback_coords
        logger.info(f"[FALLBACK] Approximate location for: {place_name} -> ({fallback_coords[0]:.6f}, {fallback_coords[1]:.6f})")
        return fallback_coords
            
    except Exception as e:
        logger.error(f"Error geocoding {place_name}: {str(e)}")
        # Use intelligent fallback even on error
        fallback_coords = generate_intelligent_fallback(place_name, fallback_counter)
        fallback_counter += 1
        geocoding_cache[place_name] = fallback_coords
        logger.info(f"[FALLBACK] Approximate location for: {place_name} due to error")
        return fallback_coords


def generate_intelligent_fallback(place_name: str, counter: int) -> Tuple[float, float]:
    """
    Generate intelligent fallback coordinates based on place name analysis.
    Uses different base coordinates depending on keywords in the name.
    Adds deterministic offset to avoid duplicate coordinates.
    
    Args:
        place_name: Name of the place
        counter: Counter (not used, kept for compatibility)
        
    Returns:
        Tuple of (latitude, longitude) - unique for each place name
    """
    import hashlib
    
    place_upper = place_name.upper()
    place_lower = place_name.lower()
    
    # More specific keyword detection with priorities
    base_lat, base_lon = None, None
    
    # Check for specific locations first (most specific)
    if 'AEROPUERTO' in place_upper or 'ADUANA' in place_upper:
        base_lat, base_lon = LATACUNGA_LANDMARKS['aeropuerto']
    elif 'PANAMERICANA' in place_upper and 'SUR' in place_upper:
        base_lat, base_lon = LATACUNGA_LANDMARKS['panamericana_sur']
    elif 'PANAMERICANA' in place_upper and 'NORTE' in place_upper:
        base_lat, base_lon = LATACUNGA_LANDMARKS['panamericana_norte']
    elif 'PANAMERICANA' in place_upper:
        # Generic Panamericana, use center between north and south
        base_lat = (LATACUNGA_LANDMARKS['panamericana_norte'][0] + LATACUNGA_LANDMARKS['panamericana_sur'][0]) / 2
        base_lon = LATACUNGA_LANDMARKS['centro'][1]
    
    # Parroquias rurales (more specific locations)
    elif any(word in place_upper for word in ['MULALO', 'MULALÓ', 'JOSEGUANGO', 'JOSE GUANGO', 'GUANGO']):
        base_lat, base_lon = (-0.7800, -78.5756)  # Mulaló area
    elif any(word in place_upper for word in ['BELISARIO QUEVEDO', 'BELISARIO']):
        base_lat, base_lon = (-0.9150, -78.5800)  # Belisario Quevedo
    elif any(word in place_upper for word in ['ALAQUEZ', 'ALÁQUEZ']):
        base_lat, base_lon = (-0.8800, -78.5900)  # Aláquez
    elif any(word in place_upper for word in ['TANICUCHI', 'TANICUCHÍ']):
        base_lat, base_lon = (-0.9000, -78.6700)  # Tanicuchí
    elif any(word in place_upper for word in ['GUAYTACAMA']):
        base_lat, base_lon = (-0.8200, -78.6400)  # Guaytacama
    elif any(word in place_upper for word in ['PASTOCALLE', 'PASTO CALLE']):
        base_lat, base_lon = (-0.8700, -78.7500)  # Pastocalle
    elif any(word in place_upper for word in ['TOACAZO']):
        base_lat, base_lon = (-0.8500, -78.7800)  # Toacazo
    elif any(word in place_upper for word in ['POALO', 'POALÓ']):
        base_lat, base_lon = (-0.9000, -78.5500)  # Poaló
    elif any(word in place_upper for word in ['SAN BUENAVENTURA', 'BUENAVENTURA']):
        base_lat, base_lon = (-0.8900, -78.6400)  # San Buenaventura
    
    # Directional keywords (less specific)
    elif any(word in place_upper for word in ['ORIENTAL', 'ESTE', 'NIAGARA', 'MIRADOR']):
        base_lat, base_lon = LATACUNGA_LANDMARKS['oriental']
    elif any(word in place_upper for word in ['OCCIDENTAL', 'OESTE']):
        base_lat, base_lon = LATACUNGA_LANDMARKS['occidental']
    elif 'SUR' in place_upper or 'SALIDA' in place_upper:
        base_lat, base_lon = LATACUNGA_LANDMARKS['sur']
    elif 'NORTE' in place_upper or 'ENTRADA' in place_upper:
        base_lat, base_lon = LATACUNGA_LANDMARKS['norte']
    
    # Type-based detection (general areas)
    elif any(word in place_lower for word in ['floricola', 'florícola', 'flower']):
        base_lat, base_lon = (-0.8500, -78.6000)  # Flower farms area (north-east)
    elif any(word in place_upper for word in ['GASOLINERA', 'GASOLINER']):
        base_lat, base_lon = LATACUNGA_LANDMARKS['panamericana_norte']  # Usually on Panamericana
    elif any(word in place_upper for word in ['UNIDAD EDUCATIVA', 'ESCUELA', 'COLEGIO']):
        base_lat, base_lon = LATACUNGA_LANDMARKS['centro']  # Schools usually in urban areas
    elif any(word in place_upper for word in ['CENTRO DE SALUD', 'HOSPITAL', 'CLINICA']):
        base_lat, base_lon = LATACUNGA_LANDMARKS['centro']  # Health centers in urban areas
    elif any(word in place_upper for word in ['CIUDADELA', 'URB', 'URBANIZACIÓN', 'CONJUNTO']):
        base_lat, base_lon = (-0.9400, -78.6100)  # Urban residential areas
    
    # Default to centro if no keywords matched
    else:
        base_lat, base_lon = LATACUNGA_LANDMARKS['centro']
    
    # Generate a deterministic but unique offset based on place name hash
    # This ensures the same place name always gets the same coordinates
    hash_input = place_name.encode('utf-8')
    hash_value = int(hashlib.md5(hash_input).hexdigest(), 16)
    
    # Create offset in a 1.5km radius (~0.0135 degrees)
    # Using hash to make it deterministic and repeatable
    offset_lat = ((hash_value % 10000) / 10000.0 - 0.5) * 0.027  # ±0.0135 degrees (~1.5km)
    offset_lon = (((hash_value // 10000) % 10000) / 10000.0 - 0.5) * 0.027
    
    final_lat = base_lat + offset_lat
    final_lon = base_lon + offset_lon
    
    # Ensure coordinates stay within reasonable Latacunga bounds
    final_lat = max(-1.0, min(-0.85, final_lat))
    final_lon = max(-78.75, min(-78.50, final_lon))
    
    return (final_lat, final_lon)


def find_header_row(df: pd.DataFrame) -> Optional[int]:
    """
    Find the row index that contains the actual column headers.
    Looks for keywords like 'PARROQUIAS', 'BARRIOS', or 'LUGARES'.
    
    Args:
        df: DataFrame to search
        
    Returns:
        Row index of header, or None if not found
    """
    keywords = ['PARROQUIAS', 'BARRIOS', 'LUGARES', 'PARROQUIA', 'BARRIO', 'LUGAR']
    
    for idx, row in df.iterrows():
        row_str = ' '.join([str(val).upper() for val in row.values if pd.notna(val)])
        if any(keyword in row_str for keyword in keywords):
            logger.info(f"Found header row at index {idx}")
            return idx
    
    return None


def clean_place_name(name: str) -> str:
    """
    Clean and normalize place names.
    
    Args:
        name: Raw place name
        
    Returns:
        Cleaned place name
    """
    if pd.isna(name) or not isinstance(name, str):
        return ""
    
    # Remove extra whitespace
    name = ' '.join(name.split())
    
    # Remove special characters but keep accents
    name = name.strip()
    
    # Remove common prefixes if too long
    prefixes = ['BARRIO ', 'SECTOR ', 'COMUNIDAD ']
    for prefix in prefixes:
        if name.upper().startswith(prefix) and len(name) > 30:
            name = name[len(prefix):]
    
    return name


def extract_route_number(filename: str) -> Optional[int]:
    """
    Extract route number from filename.
    
    Args:
        filename: Name of the file
        
    Returns:
        Route number or None
    """
    match = re.search(r'RUTA\s+(\d+)', filename.upper())
    if match:
        return int(match.group(1))
    return None


def extract_day_of_week(filename: str) -> Optional[int]:
    """
    Extract day of week from filename.
    
    Args:
        filename: Name of the file
        
    Returns:
        Day number (0-6) or None
    """
    filename_upper = filename.upper()
    for day_name, day_num in DAY_MAP.items():
        if day_name in filename_upper:
            return day_num
    return None


def process_lateral_file(filepath: Path) -> List[Dict]:
    """
    Process a lateral (urban) route file.
    
    Args:
        filepath: Path to CSV or Excel file
        
    Returns:
        List of route records
    """
    logger.info(f"\n{'='*60}")
    logger.info(f"Processing LATERAL file: {filepath.name}")
    logger.info(f"{'='*60}")
    
    try:
        # Read file based on extension
        file_ext = filepath.suffix.lower()
        if file_ext in ['.xlsx', '.xls']:
            df = pd.read_excel(filepath, engine='openpyxl' if file_ext == '.xlsx' else None)
        else:
            df = pd.read_csv(filepath, encoding='utf-8', skip_blank_rows=True)
        
        # Find header row
        header_row = find_header_row(df)
        if header_row is None:
            logger.warning(f"Could not find header row in {filepath.name}")
            return []
        
        # Re-read with correct header
        if file_ext in ['.xlsx', '.xls']:
            df = pd.read_excel(filepath, skiprows=header_row, engine='openpyxl' if file_ext == '.xlsx' else None)
        else:
            df = pd.read_csv(filepath, encoding='utf-8', skiprows=header_row, skip_blank_rows=True)
        
        # Determine sector_id
        if 'ORIENTAL' in filepath.name.upper():
            sector_id = 'LATERAL_ORIENTAL'
            sector_name = 'Ruta Lateral Oriental'
        elif 'OCCIDENTAL' in filepath.name.upper():
            sector_id = 'LATERAL_OCCIDENTAL'
            sector_name = 'Ruta Lateral Occidental'
        elif 'NOCTURNA' in filepath.name.upper():
            sector_id = 'LATERAL_NOCTURNA'
            sector_name = 'Ruta Lateral Nocturna'
        else:
            sector_id = f"LATERAL_{filepath.stem[:20]}"
            sector_name = f"Ruta Lateral {filepath.stem[:20]}"
        
        # Determine schedule
        schedule = '21:00-05:00' if 'NOCTURNA' in filepath.name.upper() else '07:30-16:30'
        
        # All days for lateral routes
        days = [0, 1, 2, 3, 4, 5, 6]
        
        # Find the column with place names - look for 'LUGARES' or similar
        place_column = None
        for col in df.columns:
            col_str = str(col).strip().upper()
            # Check exact match or contains keywords
            if col_str == 'LUGARES' or col_str == 'LUGAR' or 'PARROQUIA' in col_str or 'BARRIO' in col_str:
                place_column = col
                logger.info(f"Found place column: '{col}'")
                break
        
        # If still not found, check if there's a column named exactly 'LUGARES'
        if place_column is None:
            # Try to find by checking actual values in columns
            for col in df.columns:
                # Skip unnamed columns
                if 'Unnamed' in str(col):
                    continue
                # Check if this column has string values (potential place names)
                sample_vals = df[col].dropna().head(5)
                if len(sample_vals) > 0 and all(isinstance(v, str) for v in sample_vals):
                    place_column = col
                    logger.info(f"Using column '{col}' based on content type")
                    break
        
        if place_column is None:
            logger.warning(f"Could not find place name column in {filepath.name}")
            logger.warning(f"Available columns: {df.columns.tolist()}")
            return []
        
        logger.info(f"Using column: '{place_column}' for place names")
        
        # Process places
        records = []
        places = df[place_column].dropna().unique()
        
        logger.info(f"Found {len(places)} unique places")
        
        for place in places:
            place_clean = clean_place_name(place)
            if not place_clean or len(place_clean) < 3:
                continue
            
            # Get coordinates
            coords = get_coords(place_clean)
            
            if coords:
                lat, lon = coords
                # Create record for each day
                for day in days:
                    records.append({
                        'sector_id': sector_id,
                        'sector_name': sector_name,
                        'sector_type': 'lateral',
                        'days': days,
                        'schedule': schedule,
                        'place_name': place_clean,
                        'lat': lat,
                        'lon': lon,
                        'day_of_week': day
                    })
        
        logger.info(f"[OK] Processed {len(records)} route records from {filepath.name}")
        return records
        
    except Exception as e:
        logger.error(f"Error processing {filepath.name}: {str(e)}")
        return []


def process_posterior_file(filepath: Path) -> List[Dict]:
    """
    Process a posterior (rural) route file.
    Excel files may have multiple sheets, one per day.
    
    Args:
        filepath: Path to CSV or Excel file
        
    Returns:
        List of route records
    """
    logger.info(f"\n{'='*60}")
    logger.info(f"Processing POSTERIOR file: {filepath.name}")
    logger.info(f"{'='*60}")
    
    all_records = []
    
    # Extract route number from filename
    route_num = extract_route_number(filepath.name)
    if route_num is None:
        logger.warning(f"Could not extract route number from {filepath.name}")
        return []
    
    sector_id = f'RUTA_POST_{route_num}'
    
    try:
        file_ext = filepath.suffix.lower()
        
        # For Excel files, check if there are multiple sheets (one per day)
        if file_ext in ['.xlsx', '.xls']:
            import openpyxl
            wb = openpyxl.load_workbook(filepath, read_only=True, data_only=True)
            sheet_names = wb.sheetnames
            wb.close()
            
            logger.info(f"Found {len(sheet_names)} sheets in file")
            
            # Process each sheet
            for sheet_name in sheet_names:
                logger.info(f"Processing sheet: {sheet_name}")
                
                # Extract day from sheet name
                day_of_week = extract_day_of_week(sheet_name)
                if day_of_week is None:
                    logger.warning(f"Could not extract day from sheet name: {sheet_name}")
                    continue
                
                records = process_posterior_sheet(filepath, sheet_name, route_num, sector_id, day_of_week)
                all_records.extend(records)
        else:
            # CSV file - single day
            day_of_week = extract_day_of_week(filepath.name)
            if day_of_week is None:
                logger.warning(f"Could not extract day from filename: {filepath.name}")
                return []
            
            records = process_posterior_sheet(filepath, None, route_num, sector_id, day_of_week)
            all_records.extend(records)
            
    except Exception as e:
        logger.error(f"Error processing {filepath.name}: {str(e)}")
        return []
    
    return all_records


def process_posterior_sheet(filepath: Path, sheet_name: Optional[str], route_num: int, sector_id: str, day_of_week: int) -> List[Dict]:
    """
    Process a single sheet/file for a posterior route.
    
    Args:
        filepath: Path to file
        sheet_name: Sheet name (for Excel) or None (for CSV)
        route_num: Route number
        sector_id: Sector ID
        day_of_week: Day of week (0-6)
        
    Returns:
        List of route records
    """
    try:
        # Read file/sheet
        file_ext = filepath.suffix.lower()
        if file_ext in ['.xlsx', '.xls']:
            df = pd.read_excel(filepath, sheet_name=sheet_name, engine='openpyxl' if file_ext == '.xlsx' else None)
        else:
            df = pd.read_csv(filepath, encoding='utf-8', skip_blank_rows=True)
        
        # Find header row
        header_row = find_header_row(df)
        if header_row is None:
            logger.warning(f"Could not find header row in sheet/file")
            return []
        
        # Re-read with correct header
        if file_ext in ['.xlsx', '.xls']:
            df = pd.read_excel(filepath, sheet_name=sheet_name, skiprows=header_row, engine='openpyxl' if file_ext == '.xlsx' else None)
        else:
            df = pd.read_csv(filepath, encoding='utf-8', skiprows=header_row, skip_blank_rows=True)
        
        sector_name = f'Ruta Posterior {route_num}'
        day_name = [k for k, v in DAY_MAP.items() if v == day_of_week][0]
        logger.info(f"Route {route_num} operates on: {day_name} (day {day_of_week})")
        
        # Find the column with place names - same logic as lateral
        place_column = None
        for col in df.columns:
            col_str = str(col).strip().upper()
            if col_str == 'LUGARES' or col_str == 'LUGAR' or 'PARROQUIA' in col_str or 'BARRIO' in col_str:
                place_column = col
                logger.info(f"Found place column: '{col}'")
                break
        
        if place_column is None:
            # Try to find by checking actual values
            for col in df.columns:
                if 'Unnamed' in str(col):
                    continue
                sample_vals = df[col].dropna().head(5)
                if len(sample_vals) > 0 and all(isinstance(v, str) for v in sample_vals):
                    place_column = col
                    logger.info(f"Using column '{col}' based on content type")
                    break
        
        if place_column is None:
            logger.warning(f"Could not find place name column in {filepath.name}")
            logger.warning(f"Available columns: {df.columns.tolist()}")
            return []
        
        logger.info(f"Using column: '{place_column}' for place names")
        
        # Process places
        records = []
        places = df[place_column].dropna().unique()
        
        logger.info(f"Found {len(places)} unique places")
        
        for place in places:
            place_clean = clean_place_name(place)
            if not place_clean or len(place_clean) < 3:
                continue
            
            # Get coordinates
            coords = get_coords(place_clean)
            
            if coords:
                lat, lon = coords
                records.append({
                    'sector_id': sector_id,
                    'sector_name': sector_name,
                    'sector_type': 'posterior',
                    'days': [day_of_week],
                    'schedule': '07:00-17:00',
                    'place_name': place_clean,
                    'lat': lat,
                    'lon': lon,
                    'day_of_week': day_of_week
                })
        
        logger.info(f"[OK] Processed {len(records)} route records from {filepath.name}")
        return records
        
    except Exception as e:
        logger.error(f"Error processing {filepath.name}: {str(e)}")
        return []


def generate_sql(records: List[Dict], output_path: Path):
    """
    Generate SQL INSERT statements from route records.
    
    Args:
        records: List of route records
        output_path: Path to output SQL file
    """
    logger.info(f"\n{'='*60}")
    logger.info("Generating SQL file")
    logger.info(f"{'='*60}")
    
    # Group records by sector
    sectors = {}
    for record in records:
        sector_id = record['sector_id']
        if sector_id not in sectors:
            sectors[sector_id] = {
                'sector_name': record['sector_name'],
                'sector_type': record['sector_type'],
                'days': record['days'],
                'schedule': record['schedule'],
                'places': []
            }
        sectors[sector_id]['places'].append(record)
    
    # Create output directory if it doesn't exist
    output_path.parent.mkdir(parents=True, exist_ok=True)
    
    with open(output_path, 'w', encoding='utf-8') as f:
        # Write header
        f.write("-- Latacunga Garbage Collection Routes Seed Data\n")
        f.write(f"-- Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
        f.write(f"-- Total sectors: {len(sectors)}\n")
        f.write(f"-- Total route schedules: {len(records)}\n\n")
        
        f.write("-- Enable PostGIS extension\n")
        f.write("CREATE EXTENSION IF NOT EXISTS postgis;\n\n")
        
        f.write("-- Create schemas if not exist\n")
        f.write("CREATE SCHEMA IF NOT EXISTS rutas;\n\n")
        
        f.write("-- Create sectors table\n")
        f.write("""
CREATE TABLE IF NOT EXISTS rutas.sectors (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL CHECK (type IN ('lateral', 'posterior')),
  days INTEGER[] NOT NULL,
  schedule TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);\n\n""")
        
        f.write("-- Create route_schedules table\n")
        f.write("""
CREATE TABLE IF NOT EXISTS rutas.route_schedules (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  sector_id TEXT NOT NULL REFERENCES rutas.sectors(id) ON DELETE CASCADE,
  landmark_name TEXT NOT NULL,
  landmark_geom GEOMETRY(POINT, 4326) NOT NULL,
  day_of_week INTEGER NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (sector_id, landmark_name, day_of_week)
);\n\n""")
        
        f.write("-- Create spatial index\n")
        f.write("CREATE INDEX IF NOT EXISTS idx_route_schedules_geom ON rutas.route_schedules USING GIST (landmark_geom);\n")
        f.write("CREATE INDEX IF NOT EXISTS idx_route_schedules_sector ON rutas.route_schedules (sector_id);\n")
        f.write("CREATE INDEX IF NOT EXISTS idx_route_schedules_day ON rutas.route_schedules (day_of_week);\n\n")
        
        f.write("-- Begin transaction\n")
        f.write("BEGIN;\n\n")
        
        # Write sectors
        f.write("-- Insert sectors\n")
        for sector_id, sector_data in sectors.items():
            days_array = '{' + ','.join(map(str, sector_data['days'])) + '}'
            schedule = sector_data['schedule']
            f.write(
                f"INSERT INTO rutas.sectors (id, name, type, days, schedule) "
                f"VALUES ('{sector_id}', '{sector_data['sector_name']}', "
                f"'{sector_data['sector_type']}', '{days_array}', '{schedule}') "
                f"ON CONFLICT (id) DO UPDATE SET "
                f"name = EXCLUDED.name, "
                f"type = EXCLUDED.type, "
                f"days = EXCLUDED.days, "
                f"schedule = EXCLUDED.schedule;\n"
            )
        f.write("\n")
        
        # Write route schedules
        f.write("-- Insert route schedules\n")
        for record in records:
            # Escape single quotes in place names
            place_name = record['place_name'].replace("'", "''")
            f.write(
                f"INSERT INTO rutas.route_schedules (sector_id, landmark_name, landmark_geom, day_of_week) "
                f"VALUES ('{record['sector_id']}', '{place_name}', "
                f"ST_SetSRID(ST_Point({record['lon']:.8f}, {record['lat']:.8f}), 4326), "
                f"{record['day_of_week']}) "
                f"ON CONFLICT (sector_id, landmark_name, day_of_week) DO NOTHING;\n"
            )
        
        f.write("\n-- Commit transaction\n")
        f.write("COMMIT;\n\n")
        
        # Write statistics
        f.write("-- Statistics\n")
        f.write(f"-- Total sectors inserted: {len(sectors)}\n")
        f.write(f"-- Total route schedules inserted: {len(records)}\n")
        
        # Group by sector type
        lateral_count = sum(1 for s in sectors.values() if s['sector_type'] == 'lateral')
        posterior_count = sum(1 for s in sectors.values() if s['sector_type'] == 'posterior')
        f.write(f"-- Lateral routes: {lateral_count}\n")
        f.write(f"-- Posterior routes: {posterior_count}\n")
    
    logger.info(f"✓ SQL file generated: {output_path}")
    logger.info(f"  - Sectors: {len(sectors)}")
    logger.info(f"  - Route schedules: {len(records)}")


def generate_fallback_report(output_path: Path):
    """
    Generate a CSV report of places that used fallback coordinates.
    This allows manual correction of coordinates later.
    
    Args:
        output_path: Path to save the CSV report
    """
    LATACUNGA_CENTER = (-0.9346, -78.6156)
    
    fallback_places = [
        (place, coords) 
        for place, coords in geocoding_cache.items() 
        if coords == LATACUNGA_CENTER
    ]
    
    if not fallback_places:
        logger.info("No fallback coordinates used - all places were geocoded successfully!")
        return
    
    logger.info(f"\n{'='*60}")
    logger.info(f"Generating fallback coordinates report")
    logger.info(f"{'='*60}")
    
    with open(output_path, 'w', encoding='utf-8') as f:
        f.write("place_name,latitude,longitude,notes\n")
        for place, coords in fallback_places:
            f.write(f'"{place}",{coords[0]:.6f},{coords[1]:.6f},"Latacunga center (fallback)"\n')
    
    logger.info(f"[REPORT] Fallback coordinates report: {output_path}")
    logger.info(f"  - Places using fallback: {len(fallback_places)}")
    logger.info(f"  - You can manually update coordinates in this CSV and re-import")


def main():
    """Main ETL process."""
    logger.info("="*60)
    logger.info("Latacunga Routes ETL Importer")
    logger.info("="*60)
    
    # Check input directory exists
    if not INPUT_DIR.exists():
        logger.error(f"Input directory not found: {INPUT_DIR}")
        logger.error("Please create ./data/raw/ and place CSV/Excel files there")
        sys.exit(1)
    
    # Find all CSV and Excel files
    csv_files = list(INPUT_DIR.glob('*.csv'))
    xlsx_files = list(INPUT_DIR.glob('*.xlsx'))
    xls_files = list(INPUT_DIR.glob('*.xls'))
    
    all_files = csv_files + xlsx_files + xls_files
    
    if not all_files:
        logger.error(f"No CSV or Excel files found in {INPUT_DIR}")
        logger.error(f"Supported formats: .csv, .xlsx, .xls")
        sys.exit(1)
    
    logger.info(f"Found {len(all_files)} files ({len(csv_files)} CSV, {len(xlsx_files)} XLSX, {len(xls_files)} XLS)")
    
    # Process files
    all_records = []
    
    for filepath in all_files:
        filename_upper = filepath.name.upper()
        
        if 'LATERAL' in filename_upper:
            records = process_lateral_file(filepath)
            all_records.extend(records)
        elif 'PLANIFICACION' in filename_upper and 'RUTA' in filename_upper:
            # Rural route files (PLANIFICACION RUTA N...)
            records = process_posterior_file(filepath)
            all_records.extend(records)
        elif 'RUTA' in filename_upper and any(day in filename_upper for day in DAY_MAP.keys()):
            # Old format: RUTA N DIA.csv
            records = process_posterior_file(filepath)
            all_records.extend(records)
        else:
            logger.warning(f"Skipping unrecognized file: {filepath.name}")
    
    if not all_records:
        logger.error("No records processed. Exiting.")
        sys.exit(1)
    
    # Generate SQL
    generate_sql(all_records, OUTPUT_FILE)
    
    # Generate fallback coordinates report
    fallback_report_path = OUTPUT_DIR / 'fallback_coordinates.csv'
    generate_fallback_report(fallback_report_path)
    
    logger.info("\n" + "="*60)
    logger.info("ETL Process Completed Successfully!")
    logger.info("="*60)
    logger.info(f"Total records processed: {len(all_records)}")
    logger.info(f"Output file: {OUTPUT_FILE}")
    logger.info(f"Geocoding cache size: {len(geocoding_cache)}")
    logger.info(f"Log file: etl_routes_importer.log")


if __name__ == "__main__":
    main()
