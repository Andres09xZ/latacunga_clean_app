#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Generate Zones Script
Crea polígonos automáticos tipo "mancha orgánica" que representen el área geográfica 
de cada ruta de recolección por día.

Autor: Desarrollador Senior
Fecha: 2025-11-22
Versión: 2.0 - Con filtros anti-outliers y geometría orgánica
"""

import os
import re
import time
import json
import math
import logging
from typing import List, Tuple, Dict, Optional

import pandas as pd
import requests
from shapely.geometry import Point, MultiPoint, LineString, shape, mapping
from shapely.ops import unary_union
import geojson
import openpyxl

# ============================================================
# CONFIGURACIÓN
# ============================================================

RAW_DATA_DIR = "./data/raw"
OUTPUT_FILE = "./migrations/zonas_recoleccion.geojson"
LOG_FILE = "./scripts/generate_zones.log"

# Configuración de geocoding
NOMINATIM_URL = "https://nominatim.openstreetmap.org/search"
RATE_LIMIT_DELAY = 1.1  # Segundos entre peticiones
MAX_BARRIOS_PER_FILE = 8  # Límite de barrios para no saturar la API
USER_AGENT = "LatacungaRouteZoneGenerator/2.0 (Contact: admin@latacunga.gob.ec)"

# Palabras clave para identificar columna de barrios
KEYWORDS_BARRIOS = ["BARRIOS", "LUGARES", "SECTORES", "LUGAR", "SECTOR", "BARRIO"]

# NUEVOS PARÁMETROS PARA FILTROS Y GEOMETRÍA ORGÁNICA
LATACUNGA_CENTER = (-0.933, -78.614)  # Centro de Latacunga (lat, lon)
MAX_DISTANCE_FROM_CENTER_KM = 25.0     # Filtro global: máximo 25km del centro
MAX_DISTANCE_FROM_CENTROID_KM = 5.0    # Filtro local: máximo 5km del centroide del grupo
POINT_BUFFER_DEGREES = 0.004           # Buffer para cada punto (~400 metros)
SIMPLIFY_TOLERANCE = 0.001             # Simplificación de geometría final

# Límites de Ecuador para validación
ECUADOR_BOUNDS = {
    'min_lat': -5.0,
    'max_lat': 2.0,
    'min_lon': -92.0,
    'max_lon': -75.0
}

# ============================================================
# LOGGING
# ============================================================

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler(LOG_FILE, encoding='utf-8'),
        logging.StreamHandler()
    ]
)

logger = logging.getLogger(__name__)

# ============================================================
# FUNCIONES DE UTILIDAD
# ============================================================

def haversine_distance(lat1: float, lon1: float, lat2: float, lon2: float) -> float:
    """
    Calcula la distancia en kilómetros entre dos coordenadas usando Haversine.
    
    Args:
        lat1, lon1: Coordenadas del primer punto
        lat2, lon2: Coordenadas del segundo punto
        
    Returns:
        Distancia en kilómetros
    """
    # Radio de la Tierra en km
    R = 6371.0
    
    # Convertir grados a radianes
    lat1_rad = math.radians(lat1)
    lon1_rad = math.radians(lon1)
    lat2_rad = math.radians(lat2)
    lon2_rad = math.radians(lon2)
    
    # Diferencias
    dlat = lat2_rad - lat1_rad
    dlon = lon2_rad - lon1_rad
    
    # Fórmula de Haversine
    a = math.sin(dlat / 2)**2 + math.cos(lat1_rad) * math.cos(lat2_rad) * math.sin(dlon / 2)**2
    c = 2 * math.atan2(math.sqrt(a), math.sqrt(1 - a))
    
    distance = R * c
    return distance

def is_within_latacunga_area(lat: float, lon: float) -> bool:
    """
    Filtro global: Verifica si una coordenada está dentro del área de Latacunga.
    Rechaza coordenadas a más de 25km del centro.
    
    Args:
        lat, lon: Coordenadas a verificar
        
    Returns:
        True si está dentro del área válida
    """
    center_lat, center_lon = LATACUNGA_CENTER
    distance = haversine_distance(center_lat, center_lon, lat, lon)
    
    if distance > MAX_DISTANCE_FROM_CENTER_KM:
        logger.warning(f"[FILTER] Rejected point ({lat:.6f}, {lon:.6f}) - {distance:.2f}km from center (max {MAX_DISTANCE_FROM_CENTER_KM}km)")
        return False
    
    return True

def calculate_centroid(points: List[Tuple[float, float]]) -> Tuple[float, float]:
    """
    Calcula el centroide (punto promedio) de una lista de coordenadas.
    
    Args:
        points: Lista de tuplas (lat, lon)
        
    Returns:
        Tupla (centroid_lat, centroid_lon)
    """
    if not points:
        return (0.0, 0.0)
    
    avg_lat = sum(p[0] for p in points) / len(points)
    avg_lon = sum(p[1] for p in points) / len(points)
    
    return (avg_lat, avg_lon)

def filter_outliers_from_group(points: List[Tuple[float, float]]) -> List[Tuple[float, float]]:
    """
    Filtro local: Elimina puntos que están muy lejos del centroide del grupo.
    
    Args:
        points: Lista de tuplas (lat, lon)
        
    Returns:
        Lista filtrada de coordenadas
    """
    if len(points) <= 2:
        # Si hay muy pocos puntos, no filtrar
        return points
    
    # Calcular centroide
    centroid_lat, centroid_lon = calculate_centroid(points)
    logger.info(f"[FILTER] Group centroid: ({centroid_lat:.6f}, {centroid_lon:.6f})")
    
    # Filtrar puntos
    filtered_points = []
    for lat, lon in points:
        distance = haversine_distance(centroid_lat, centroid_lon, lat, lon)
        
        if distance <= MAX_DISTANCE_FROM_CENTROID_KM:
            filtered_points.append((lat, lon))
        else:
            logger.warning(f"[FILTER] Rejected outlier ({lat:.6f}, {lon:.6f}) - {distance:.2f}km from centroid (max {MAX_DISTANCE_FROM_CENTROID_KM}km)")
    
    logger.info(f"[FILTER] Kept {len(filtered_points)} of {len(points)} points after outlier removal")
    
    return filtered_points

def clean_text(text: str) -> str:
    """Limpia texto eliminando espacios extras y caracteres especiales."""
    if pd.isna(text):
        return ""
    return str(text).strip()

def extract_route_info(filename: str) -> Tuple[str, str]:
    """
    Extrae el ID de ruta y el día del nombre del archivo.
    
    Args:
        filename: Nombre del archivo (ej: "PLANIFICACION RUTA 1 LUNES.csv")
        
    Returns:
        Tupla (route_id, day) (ej: ("RUTA_1", "LUNES"))
    """
    filename_upper = filename.upper()
    
    # Extraer día de la semana
    days = ["LUNES", "MARTES", "MIERCOLES", "MIÉRCOLES", "JUEVES", "VIERNES", "SABADO", "SÁBADO", "DOMINGO"]
    day = None
    for d in days:
        if d in filename_upper:
            day = d.replace("Á", "A").replace("É", "E").replace("Í", "I").replace("Ó", "O")
            break
    
    # Extraer ID de ruta
    route_id = None
    
    # Patrón para rutas rurales: RUTA N o RUTA 1, etc.
    ruta_match = re.search(r'RUTA\s*N?[\s_-]*(\d+)', filename_upper)
    if ruta_match:
        route_num = ruta_match.group(1)
        route_id = f"RUTA_{route_num}"
    
    # Patrón para rutas urbanas laterales
    elif "LATERAL" in filename_upper:
        if "ORIENTAL" in filename_upper:
            route_id = "LATERAL_ORIENTAL"
        elif "OCCIDENTAL" in filename_upper:
            route_id = "LATERAL_OCCIDENTAL"
        elif "NOCTURNA" in filename_upper:
            route_id = "LATERAL_NOCTURNA"
        else:
            route_id = "LATERAL"
    
    # Default
    if not route_id:
        route_id = "RUTA_UNKNOWN"
    if not day:
        day = "TODOS"
    
    return route_id, day

def find_barrios_column(df: pd.DataFrame) -> Optional[str]:
    """
    Busca la columna que contiene los nombres de barrios/lugares.
    
    Args:
        df: DataFrame con los datos
        
    Returns:
        Nombre de la columna o None si no se encuentra
    """
    # Buscar en los nombres de columnas
    for col in df.columns:
        col_upper = str(col).upper()
        for keyword in KEYWORDS_BARRIOS:
            if keyword in col_upper:
                logger.info(f"[OK] Found barrios column by name: {col}")
                return col
    
    # Buscar por contenido (primera columna con strings)
    for col in df.columns:
        if "Unnamed" not in str(col):
            sample = df[col].dropna().head(10)
            if len(sample) > 0:
                # Verificar si contiene principalmente strings
                string_count = sum(isinstance(x, str) for x in sample)
                if string_count / len(sample) > 0.7:
                    logger.info(f"[OK] Found barrios column by content: {col}")
                    return col
    
    return None

def read_excel_sheets(filepath: str) -> List[Tuple[str, pd.DataFrame, str]]:
    """
    Lee todas las hojas de un archivo Excel.
    
    Args:
        filepath: Ruta al archivo Excel
        
    Returns:
        Lista de tuplas (nombre_hoja, DataFrame, columna_barrios)
    """
    results = []
    
    try:
        # Cargar el workbook para obtener nombres de hojas
        wb = openpyxl.load_workbook(filepath, read_only=True)
        sheet_names = wb.sheetnames
        wb.close()
        
        for sheet_name in sheet_names:
            # Intentar leer cada hoja
            for header_row in range(10):
                try:
                    df = pd.read_excel(filepath, header=header_row, sheet_name=sheet_name)
                    barrios_col = find_barrios_column(df)
                    if barrios_col:
                        results.append((sheet_name, df, barrios_col))
                        break
                except Exception as e:
                    continue
            
    except Exception as e:
        logger.error(f"[ERROR] Reading Excel sheets: {str(e)}")
    
    return results

def read_csv_with_dynamic_header(filepath: str) -> Tuple[pd.DataFrame, str]:
    """
    Lee un CSV detectando dinámicamente la fila de encabezado.
    
    Args:
        filepath: Ruta al archivo CSV
        
    Returns:
        Tupla (DataFrame, nombre de columna de barrios)
    """
    # Intentar detectar la fila de encabezado en las primeras 10 filas
    for header_row in range(10):
        try:
            df = pd.read_csv(filepath, header=header_row, encoding='utf-8')
            barrios_col = find_barrios_column(df)
            if barrios_col:
                return df, barrios_col
        except Exception as e:
            continue
    
    # Si no se encuentra, leer sin encabezado y usar primera columna
    df = pd.read_csv(filepath, header=None, encoding='utf-8')
    return df, df.columns[0]

def is_valid_coordinate(lat: float, lon: float) -> bool:
    """Verifica si una coordenada está dentro de los límites de Ecuador."""
    return (ECUADOR_BOUNDS['min_lat'] <= lat <= ECUADOR_BOUNDS['max_lat'] and
            ECUADOR_BOUNDS['min_lon'] <= lon <= ECUADOR_BOUNDS['max_lon'])

def geocode_barrio(barrio: str) -> Optional[Tuple[float, float]]:
    """
    Obtiene coordenadas de un barrio usando Nominatim.
    Aplica filtro global para rechazar coordenadas fuera del área de Latacunga.
    
    Args:
        barrio: Nombre del barrio
        
    Returns:
        Tupla (lat, lon) o None si no se encuentra o está fuera del área
    """
    if not barrio or len(barrio) < 3:
        return None
    
    # Limpiar nombre
    barrio_clean = clean_text(barrio)
    
    # Construir query
    query = f"{barrio_clean}, Latacunga, Cotopaxi, Ecuador"
    
    params = {
        'q': query,
        'format': 'json',
        'limit': 1,
        'addressdetails': 1
    }
    
    headers = {
        'User-Agent': USER_AGENT
    }
    
    try:
        response = requests.get(NOMINATIM_URL, params=params, headers=headers, timeout=10)
        response.raise_for_status()
        
        data = response.json()
        
        if data and len(data) > 0:
            lat = float(data[0]['lat'])
            lon = float(data[0]['lon'])
            
            # Validar límites de Ecuador
            if not is_valid_coordinate(lat, lon):
                logger.warning(f"[X] Invalid coords (outside Ecuador) for: {barrio_clean}")
                return None
            
            # FILTRO GLOBAL: Verificar que esté dentro del área de Latacunga
            if not is_within_latacunga_area(lat, lon):
                logger.warning(f"[X] Coordinates too far from Latacunga for: {barrio_clean}")
                return None
            
            logger.info(f"[OK] Geocoded: {barrio_clean} -> ({lat:.6f}, {lon:.6f})")
            return (lat, lon)
        else:
            logger.warning(f"[X] Not found: {barrio_clean}")
            return None
            
    except Exception as e:
        logger.error(f"[ERROR] Geocoding {barrio_clean}: {str(e)}")
        return None

def process_barrios_from_dataframe(df: pd.DataFrame, barrios_col: str) -> List[Tuple[float, float]]:
    """
    Procesa barrios de un DataFrame y retorna coordenadas.
    
    Args:
        df: DataFrame con los datos
        barrios_col: Nombre de la columna con barrios
        
    Returns:
        Lista de coordenadas [(lat, lon), ...]
    """
    # Extraer barrios únicos
    barrios = df[barrios_col].dropna().unique()
    barrios = [clean_text(b) for b in barrios if clean_text(b) and len(clean_text(b)) > 2]
    
    # Limitar a MAX_BARRIOS_PER_FILE
    if len(barrios) > MAX_BARRIOS_PER_FILE:
        logger.info(f"Limiting from {len(barrios)} to {MAX_BARRIOS_PER_FILE} barrios")
        # Seleccionar distribución uniforme
        step = len(barrios) // MAX_BARRIOS_PER_FILE
        barrios = barrios[::step][:MAX_BARRIOS_PER_FILE]
    
    logger.info(f"Processing {len(barrios)} barrios: {barrios[:3]}...")
    
    # Geocodificar barrios
    coords = []
    for i, barrio in enumerate(barrios):
        coord = geocode_barrio(barrio)
        if coord:
            coords.append(coord)
        
        # Rate limiting (excepto en la última iteración)
        if i < len(barrios) - 1:
            time.sleep(RATE_LIMIT_DELAY)
    
    logger.info(f"[RESULT] Geocoded {len(coords)} of {len(barrios)} barrios")
    
    return coords

def create_zone_geometry(points: List[Tuple[float, float]]) -> Optional[dict]:
    """
    Crea geometría de zona usando técnica de "mancha orgánica".
    
    NUEVO ALGORITMO:
    1. Aplica filtro local para eliminar outliers del grupo
    2. Convierte cada punto en un círculo (buffer)
    3. Fusiona todos los círculos en una forma orgánica
    4. Simplifica la geometría resultante
    
    Args:
        points: Lista de tuplas (lat, lon)
        
    Returns:
        Diccionario con geometría GeoJSON o None
    """
    if not points or len(points) == 0:
        return None
    
    try:
        # PASO 1: Filtrar outliers del grupo
        filtered_points = filter_outliers_from_group(points)
        
        if len(filtered_points) == 0:
            logger.warning("[ZONE] No points left after outlier filtering")
            return None
        
        logger.info(f"[ZONE] Creating organic geometry from {len(filtered_points)} points")
        
        # PASO 2: Convertir (lat, lon) a (lon, lat) para Shapely
        shapely_points = [Point(lon, lat) for lat, lon in filtered_points]
        
        if len(shapely_points) == 1:
            # Un solo punto: crear círculo simple
            geometry = shapely_points[0].buffer(POINT_BUFFER_DEGREES)
            logger.info(f"[ZONE] Created single circle (buffer={POINT_BUFFER_DEGREES})")
            
        else:
            # PASO 3: Crear círculo alrededor de cada punto
            circles = [point.buffer(POINT_BUFFER_DEGREES) for point in shapely_points]
            logger.info(f"[ZONE] Created {len(circles)} individual circles")
            
            # PASO 4: Fusionar todos los círculos en una forma orgánica
            merged_geometry = unary_union(circles)
            logger.info(f"[ZONE] Merged circles into organic shape")
            
            # PASO 5: Simplificar ligeramente para suavizar bordes
            geometry = merged_geometry.simplify(SIMPLIFY_TOLERANCE, preserve_topology=True)
            logger.info(f"[ZONE] Simplified geometry (tolerance={SIMPLIFY_TOLERANCE})")
        
        # Convertir a GeoJSON
        geom_dict = mapping(geometry)
        return geom_dict
        
    except Exception as e:
        logger.error(f"[ERROR] Creating geometry: {str(e)}")
        return None

# ============================================================
# FUNCIÓN PRINCIPAL
# ============================================================

def main():
    """Función principal del script."""
    logger.info("=" * 60)
    logger.info("Generate Zones Script - Starting")
    logger.info("=" * 60)
    
    # Verificar que existe la carpeta de datos
    if not os.path.exists(RAW_DATA_DIR):
        logger.error(f"[ERROR] Directory not found: {RAW_DATA_DIR}")
        return
    
    # Buscar archivos CSV y Excel
    data_files = []
    for root, dirs, files in os.walk(RAW_DATA_DIR):
        for file in files:
            if file.endswith(('.csv', '.CSV', '.xlsx', '.xls', '.XLSX', '.XLS')):
                data_files.append(os.path.join(root, file))
    
    logger.info(f"Found {len(data_files)} data files (CSV and Excel)")
    
    if len(data_files) == 0:
        logger.warning("[WARNING] No data files found!")
        return
    
    # Diccionario para agrupar coordenadas por zona
    zones_data = {}  # Key: "RUTA_1_LUNES", Value: [(lat, lon), ...]
    
    # Procesar cada archivo
    for filepath in data_files:
        filename = os.path.basename(filepath)
        file_ext = os.path.splitext(filepath)[1].lower()
        
        logger.info(f"\n{'=' * 60}")
        logger.info(f"Processing: {filename}")
        logger.info(f"{'=' * 60}")
        
        # Si es Excel con múltiples hojas, procesar cada hoja
        if file_ext in ['.xlsx', '.xls']:
            sheets_data = read_excel_sheets(filepath)
            
            if not sheets_data:
                logger.warning(f"[WARNING] No valid sheets found in {filename}")
                continue
            
            for sheet_name, df, barrios_col in sheets_data:
                logger.info(f"\n--- Processing sheet: {sheet_name} ---")
                
                # Extraer información de ruta y día (del nombre de archivo + hoja)
                route_id, day = extract_route_info(filename)
                
                # Si la hoja tiene día de semana, usarlo
                sheet_upper = sheet_name.upper()
                days = ["LUNES", "MARTES", "MIERCOLES", "MIÉRCOLES", "JUEVES", "VIERNES", "SABADO", "SÁBADO", "DOMINGO"]
                for d in days:
                    if d in sheet_upper:
                        day = d.replace("Á", "A").replace("É", "E").replace("Í", "I").replace("Ó", "O")
                        break
                
                zone_key = f"{route_id}_{day}"
                logger.info(f"Zone identified as: {zone_key}")
                
                # Procesar barrios de esta hoja
                coords = process_barrios_from_dataframe(df, barrios_col)
                
                # Agregar coordenadas a la zona
                if zone_key not in zones_data:
                    zones_data[zone_key] = []
                zones_data[zone_key].extend(coords)
        
        else:
            # Procesar CSV
            route_id, day = extract_route_info(filename)
            zone_key = f"{route_id}_{day}"
            
            logger.info(f"Zone identified as: {zone_key}")
            
            # Leer CSV
            try:
                df, barrios_col = read_csv_with_dynamic_header(filepath)
                logger.info(f"Read CSV with {len(df)} rows, using column: {barrios_col}")
            except Exception as e:
                logger.error(f"[ERROR] Reading {filename}: {str(e)}")
                continue
            
            # Procesar barrios
            coords = process_barrios_from_dataframe(df, barrios_col)
            
            # Agregar coordenadas a la zona
            if zone_key not in zones_data:
                zones_data[zone_key] = []
            zones_data[zone_key].extend(coords)
    
    # ============================================================
    # GENERAR GEOMETRÍAS Y GEOJSON
    # ============================================================
    
    logger.info(f"\n{'=' * 60}")
    logger.info(f"Generating zone geometries for {len(zones_data)} zones")
    logger.info(f"{'=' * 60}")
    
    features = []
    
    for zone_key, coords in zones_data.items():
        logger.info(f"\nZone: {zone_key} - {len(coords)} points")
        
        if len(coords) == 0:
            logger.warning(f"[SKIP] No coordinates for {zone_key}")
            continue
        
        # Crear geometría
        geometry = create_zone_geometry(coords)
        
        if geometry:
            # Crear feature GeoJSON
            feature = geojson.Feature(
                geometry=geometry,
                properties={
                    'zone_name': zone_key.replace('_', ' '),
                    'points_count': len(coords),
                    'route_id': '_'.join(zone_key.split('_')[:-1]),
                    'day': zone_key.split('_')[-1]
                }
            )
            features.append(feature)
            logger.info(f"[OK] Created geometry for {zone_key}")
        else:
            logger.warning(f"[SKIP] Could not create geometry for {zone_key}")
    
    # Crear FeatureCollection
    feature_collection = geojson.FeatureCollection(features)
    
    # Guardar archivo
    os.makedirs(os.path.dirname(OUTPUT_FILE), exist_ok=True)
    
    with open(OUTPUT_FILE, 'w', encoding='utf-8') as f:
        geojson.dump(feature_collection, f, indent=2, ensure_ascii=False)
    
    logger.info(f"\n{'=' * 60}")
    logger.info(f"[SUCCESS] Generated {len(features)} zone polygons")
    logger.info(f"[SUCCESS] Output saved to: {OUTPUT_FILE}")
    logger.info(f"{'=' * 60}")
    
    # Resumen
    logger.info(f"\nZones created:")
    for feature in features:
        props = feature['properties']
        logger.info(f"  - {props['zone_name']}: {props['points_count']} points")

if __name__ == "__main__":
    main()
