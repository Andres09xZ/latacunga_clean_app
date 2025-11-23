import json, os, psycopg2
from dotenv import load_dotenv

load_dotenv()
DB_URL = os.getenv('DB_URL')

# Zones macro data
zones_data = [
    {
        "zone_name": "URBANO_CENTRAL",
        "route_name": "URBANO_CENTRAL",
        "schedule_day": 0,  # All days
        "schedule_config": "NOCTURNO (21:00)",
        "geometry": {
            "type": "MultiPolygon",
            "coordinates": [[[[
                [-78.625, -0.925],
                [-78.61, -0.925],
                [-78.61, -0.945],
                [-78.625, -0.945],
                [-78.625, -0.925]
            ]]]]
        }
    },
    {
        "zone_name": "URBANO_NORTE",
        "route_name": "URBANO_NORTE",
        "schedule_day": 2,  # Tuesday
        "schedule_config": "DIURNO (Mar-Jue-Sab)",
        "geometry": {
            "type": "MultiPolygon",
            "coordinates": [[[[
                [-78.625, -0.925],
                [-78.61, -0.925],
                [-78.6, -0.89],
                [-78.63, -0.89],
                [-78.625, -0.925]
            ]]]]
        }
    },
    {
        "zone_name": "URBANO_SUR",
        "route_name": "URBANO_SUR",
        "schedule_day": 1,  # Monday
        "schedule_config": "DIURNO (Lun-Mie-Vie)",
        "geometry": {
            "type": "MultiPolygon",
            "coordinates": [[[[
                [-78.625, -0.945],
                [-78.61, -0.945],
                [-78.6, -0.98],
                [-78.64, -0.98],
                [-78.625, -0.945]
            ]]]]
        }
    },
    {
        "zone_name": "RURAL_NORTE",
        "route_name": "RURAL_NORTE",
        "schedule_day": 3,  # Wednesday
        "schedule_config": "RURAL (Rutas 2,3,4)",
        "geometry": {
            "type": "MultiPolygon",
            "coordinates": [[[[
                [-78.63, -0.89],
                [-78.6, -0.89],
                [-78.55, -0.8],
                [-78.7, -0.8],
                [-78.63, -0.89]
            ]]]]
        }
    },
    {
        "zone_name": "RURAL_SUR",
        "route_name": "RURAL_SUR",
        "schedule_day": 5,  # Friday
        "schedule_config": "RURAL (Rutas 1,5)",
        "geometry": {
            "type": "MultiPolygon",
            "coordinates": [[[[
                [-78.64, -0.98],
                [-78.6, -0.98],
                [-78.55, -1.05],
                [-78.7, -1.05],
                [-78.64, -0.98]
            ]]]]
        }
    }
]

print(f'🔌 Connecting to database...')
conn = psycopg2.connect(DB_URL)
cur = conn.cursor()

print(f'🗑️  Clearing existing zones...')
cur.execute('DELETE FROM cleaning_zones')

print(f'📝 Inserting {len(zones_data)} macro zones...')
for zone in zones_data:
    geom_json = json.dumps(zone['geometry'])
    cur.execute('''
        INSERT INTO cleaning_zones(zone_name, route_name, schedule_day, schedule_config, points_count, geom, status)
        VALUES (%s, %s, %s, %s, %s, ST_GeomFromGeoJSON(%s), 'ACUMULANDO')
    ''', (zone['zone_name'], zone['route_name'], zone['schedule_day'], zone['schedule_config'], 4, geom_json))
    print(f'  ✅ {zone["zone_name"]}')

conn.commit()
cur.close()
conn.close()

print(f'\n✅ Successfully inserted {len(zones_data)} macro zones!')
