import json, os, psycopg2
from pathlib import Path
from dotenv import load_dotenv

load_dotenv()
DB_URL=os.getenv('DB_URL')
if not DB_URL:
    raise SystemExit('DB_URL env var required')

GEOJSON_PATH=Path(__file__).parent.parent/'migrations'/'zonas_recoleccion.geojson'
with open(GEOJSON_PATH,'r',encoding='utf-8') as f:
    data=json.load(f)

conn=psycopg2.connect(DB_URL)
cur=conn.cursor()

# Map Spanish day names to integers (0=Sunday, 1=Monday, etc.)
day_map = {
    'DOMINGO': 0, 'LUNES': 1, 'MARTES': 2, 'MIÉRCOLES': 3, 'MIERCOLES': 3,
    'JUEVES': 4, 'VIERNES': 5, 'SÁBADO': 6, 'SABADO': 6
}

for feat in data['features']:
    props = feat['properties']
    name = props['zone_name']
    route = props.get('route_id', 'UNKNOWN')
    day_str = props.get('day', 'LUNES').upper()
    schedule_day = day_map.get(day_str, 1)
    points = props.get('points_count', 0)
    geom = json.dumps(feat['geometry'])
    cur.execute("INSERT INTO cleaning_zones(zone_name,route_name,schedule_day,points_count,geom) VALUES (%s,%s,%s,%s,ST_GeomFromGeoJSON(%s)) ON CONFLICT (route_name,schedule_day) DO UPDATE SET zone_name=EXCLUDED.zone_name,points_count=EXCLUDED.points_count,geom=EXCLUDED.geom",(name,route,schedule_day,points,geom))

conn.commit()
cur.close(); conn.close()
print('✅ Seed completed: inserted/updated', len(data['features']))
