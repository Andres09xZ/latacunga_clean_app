#!/usr/bin/env python3
"""
Script para insertar zonas de recolección en cleaning_zones
Autor: Sistema de Zonificación
Fecha: 2025-11-22

Uso:
    python scripts/seed_zones.py
    
Requisitos:
    pip install psycopg2-binary python-dotenv
"""

import os
import sys
import json
import psycopg2
from dotenv import load_dotenv

# Cargar variables de entorno
load_dotenv()

# ============================================================
# DATOS HARDCODEADOS - "Verdad Absoluta"
# ============================================================
ZONES = [
    {
        "zone_name": "URBANO_CENTRAL",
        "route_type": "URBANO",
        "schedule_time": "21:00:00",
        "coordinates": [
            [-78.625, -0.925],
            [-78.610, -0.925],
            [-78.610, -0.945],
            [-78.625, -0.945],
            [-78.625, -0.925]
        ]
    },
    {
        "zone_name": "URBANO_NORTE",
        "route_type": "URBANO",
        "schedule_time": "07:30:00",
        "coordinates": [
            [-78.625, -0.925],
            [-78.610, -0.925],
            [-78.600, -0.890],
            [-78.630, -0.890],
            [-78.625, -0.925]
        ]
    },
    {
        "zone_name": "URBANO_SUR",
        "route_type": "URBANO",
        "schedule_time": "07:30:00",
        "coordinates": [
            [-78.625, -0.945],
            [-78.610, -0.945],
            [-78.600, -0.980],
            [-78.640, -0.980],
            [-78.625, -0.945]
        ]
    },
    {
        "zone_name": "RURAL_NORTE",
        "route_type": "RURAL",
        "schedule_time": "07:30:00",
        "coordinates": [
            [-78.630, -0.890],
            [-78.600, -0.890],
            [-78.550, -0.800],
            [-78.700, -0.800],
            [-78.630, -0.890]
        ]
    },
    {
        "zone_name": "RURAL_SUR",
        "route_type": "RURAL",
        "schedule_time": "07:30:00",
        "coordinates": [
            [-78.640, -0.980],
            [-78.600, -0.980],
            [-78.550, -1.050],
            [-78.700, -1.050],
            [-78.640, -0.980]
        ]
    }
]

def create_geojson_polygon(coordinates):
    """
    Crea un objeto GeoJSON Polygon a partir de coordenadas
    
    Args:
        coordinates: Lista de [lon, lat] formando un polígono cerrado
        
    Returns:
        str: GeoJSON string válido
    """
    geojson = {
        "type": "Polygon",
        "coordinates": [coordinates]
    }
    return json.dumps(geojson)

def seed_zones(db_url):
    """
    Inserta las zonas de recolección en la base de datos
    
    Args:
        db_url: PostgreSQL connection string
    """
    print("=" * 60)
    print("🚀 INICIANDO SEED DE ZONAS DE RECOLECCIÓN")
    print("=" * 60)
    
    try:
        # Conectar a la base de datos
        print(f"\n📡 Conectando a base de datos...")
        conn = psycopg2.connect(db_url)
        cursor = conn.cursor()
        print("✅ Conexión exitosa")
        
        # Limpiar datos existentes
        print("\n🧹 Limpiando tabla cleaning_zones...")
        cursor.execute("TRUNCATE TABLE cleaning_zones RESTART IDENTITY CASCADE;")
        conn.commit()
        print("✅ Tabla limpiada")
        
        # Insertar zonas
        print(f"\n📍 Insertando {len(ZONES)} zonas...")
        print("-" * 60)
        
        insert_query = """
            INSERT INTO cleaning_zones 
                (zone_name, route_name, route_type, schedule_time, schedule_day, geom, threshold, current_score, status) 
            VALUES 
                (%s, %s, %s, %s, %s, ST_GeomFromGeoJSON(%s), %s, %s, %s)
            RETURNING id, zone_name;
        """
        
        for idx, zone in enumerate(ZONES, 1):
            # Crear GeoJSON
            geojson = create_geojson_polygon(zone["coordinates"])
            
            # Determinar threshold por tipo
            threshold = 50 if zone["zone_name"] == "URBANO_CENTRAL" else (30 if "URBANO" in zone["zone_name"] else 20)
            
            # Generar route_name basado en zone_name (compatibilidad con tabla original)
            route_name = f"RUTA_{idx}"
            
            # Schedule_day: 0 para todos (se maneja restricción de días en Go)
            schedule_day = 0
            
            # Ejecutar INSERT
            cursor.execute(
                insert_query,
                (
                    zone["zone_name"],
                    route_name,
                    zone["route_type"],
                    zone["schedule_time"],
                    schedule_day,
                    geojson,
                    threshold,
                    0,  # current_score inicial
                    "ACUMULANDO"  # status inicial
                )
            )
            
            # Obtener ID insertado
            result = cursor.fetchone()
            zone_id, zone_name = result
            
            print(f"  ✅ [{idx}/{len(ZONES)}] ID {zone_id}: {zone_name} ({zone['route_type']}) - Threshold: {threshold}")
        
        # Commit de la transacción
        conn.commit()
        print("-" * 60)
        print(f"✅ {len(ZONES)} zonas insertadas exitosamente")
        
        # Verificar inserción
        print("\n🔍 Verificando datos insertados...")
        cursor.execute("""
            SELECT 
                id, 
                zone_name, 
                route_type, 
                schedule_time,
                threshold,
                current_score,
                status,
                ST_AsText(geom) as geom_wkt
            FROM cleaning_zones
            ORDER BY id;
        """)
        
        zones_inserted = cursor.fetchall()
        print(f"✅ {len(zones_inserted)} zonas encontradas en base de datos")
        
        # Mostrar resumen
        print("\n📊 RESUMEN DE ZONAS:")
        print("-" * 60)
        for row in zones_inserted:
            zone_id, name, route_type, time, threshold, score, status, geom_preview = row
            geom_short = geom_preview[:50] + "..." if len(geom_preview) > 50 else geom_preview
            print(f"  ID {zone_id}: {name}")
            print(f"    Tipo: {route_type} | Hora: {time} | Threshold: {threshold}")
            print(f"    Status: {status} | Score: {score}")
            print(f"    Geom: {geom_short}")
            print()
        
        # Cerrar conexión
        cursor.close()
        conn.close()
        
        print("=" * 60)
        print("✅ SEED COMPLETADO EXITOSAMENTE")
        print("=" * 60)
        print("\n🔗 Ahora puedes probar:")
        print("   GET http://localhost:8083/api/zones")
        print("   GET http://localhost:8083/api/v1/zones (con JWT)\n")
        
        return True
        
    except psycopg2.Error as e:
        print(f"\n❌ Error de PostgreSQL: {e}")
        if conn:
            conn.rollback()
        return False
        
    except Exception as e:
        print(f"\n❌ Error inesperado: {e}")
        return False

def main():
    """Función principal"""
    # Obtener DB_URL
    db_url = os.getenv("DB_URL")
    
    if not db_url:
        print("❌ ERROR: Variable DB_URL no encontrada")
        print("   Asegúrate de tener un archivo .env con:")
        print("   DB_URL=postgresql://user:pass@host:port/database")
        sys.exit(1)
    
    # Ejecutar seed
    success = seed_zones(db_url)
    
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
