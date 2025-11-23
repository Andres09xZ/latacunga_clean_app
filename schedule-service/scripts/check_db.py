import os, psycopg2
from dotenv import load_dotenv

load_dotenv()
db_url = os.getenv('DB_URL')
# Fix channel_binding parameter
if 'channel_binding=' in db_url and '&channel_binding=' not in db_url and '?channel_binding=' not in db_url:
    db_url = db_url.replace('channel_binding=', '&channel_binding=')
conn = psycopg2.connect(db_url)
cur = conn.cursor()

cur.execute('SELECT COUNT(*) FROM cleaning_zones')
count = cur.fetchone()[0]
print(f'✅ Total zones in DB: {count}')

if count > 0:
    cur.execute('SELECT id, zone_name, route_name, schedule_day, schedule_config, status FROM cleaning_zones LIMIT 5')
    rows = cur.fetchall()
    print('\n📋 Sample zones:')
    for r in rows:
        print(f'  ID={r[0]}, Name={r[1]}, Route={r[2]}, Day={r[3]}, Config={r[4]}, Status={r[5]}')
else:
    print('⚠️  No zones found in database')

cur.close()
conn.close()
