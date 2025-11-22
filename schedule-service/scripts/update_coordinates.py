#!/usr/bin/env python3
"""
Update Coordinates Script
=========================
This script allows you to update coordinates in the generated SQL file
using a CSV file with corrected coordinates.

Usage:
    python update_coordinates.py fallback_coordinates_corrected.csv

CSV Format:
    place_name,latitude,longitude,notes
    "Oficinas Regional Oriental",-0.9346,-78.6156,"Manually corrected"
"""

import sys
import csv
import re
from pathlib import Path

def update_sql_coordinates(csv_file: Path, sql_file: Path):
    """
    Update coordinates in SQL file based on CSV corrections.
    
    Args:
        csv_file: Path to CSV with corrected coordinates
        sql_file: Path to SQL file to update
    """
    print("="*60)
    print("Coordinate Update Tool")
    print("="*60)
    
    # Read corrections from CSV
    corrections = {}
    with open(csv_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f)
        for row in reader:
            place_name = row['place_name']
            lat = float(row['latitude'])
            lon = float(row['longitude'])
            corrections[place_name] = (lat, lon)
    
    print(f"Loaded {len(corrections)} coordinate corrections from {csv_file}")
    
    # Read SQL file
    with open(sql_file, 'r', encoding='utf-8') as f:
        sql_content = f.read()
    
    # Update coordinates
    updated_count = 0
    for place_name, (new_lat, new_lon) in corrections.items():
        # Escape single quotes for SQL
        escaped_name = place_name.replace("'", "''")
        
        # Pattern to match INSERT statements for this place
        # Example: ST_Point(-78.61560000, -0.93460000)
        pattern = rf"(INSERT INTO rutas\.route_schedules.*'{escaped_name}'.*ST_Point\()(-?\d+\.\d+),\s*(-?\d+\.\d+)(\))"
        
        def replace_coords(match):
            nonlocal updated_count
            updated_count += 1
            return f"{match.group(1)}{new_lon:.8f}, {new_lat:.8f}{match.group(4)}"
        
        sql_content = re.sub(pattern, replace_coords, sql_content, flags=re.MULTILINE)
    
    # Write updated SQL
    backup_file = sql_file.with_suffix('.sql.backup')
    sql_file.rename(backup_file)
    print(f"Created backup: {backup_file}")
    
    with open(sql_file, 'w', encoding='utf-8') as f:
        f.write(sql_content)
    
    print(f"✓ Updated {updated_count} coordinate entries in {sql_file}")
    print(f"✓ Original file backed up to {backup_file}")
    print("="*60)


def main():
    if len(sys.argv) != 2:
        print("Usage: python update_coordinates.py <corrected_csv_file>")
        print("\nExample:")
        print("  python update_coordinates.py fallback_coordinates_corrected.csv")
        sys.exit(1)
    
    csv_file = Path(sys.argv[1])
    if not csv_file.exists():
        print(f"Error: CSV file not found: {csv_file}")
        sys.exit(1)
    
    sql_file = Path('./migrations/seed_routes.sql')
    if not sql_file.exists():
        print(f"Error: SQL file not found: {sql_file}")
        print("Please run etl_routes_importer.py first to generate the SQL file.")
        sys.exit(1)
    
    update_sql_coordinates(csv_file, sql_file)


if __name__ == "__main__":
    main()
