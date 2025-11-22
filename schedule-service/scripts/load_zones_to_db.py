#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Script para ejecutar la migración PostGIS y cargar las zonas desde GeoJSON

Este script:
1. Conecta a la base de datos PostgreSQL city_cleaning
2. Ejecuta la migración 005 (crear tabla cleaning_zones)
3. Carga los datos del GeoJSON generado
4. Verifica que todo se cargó correctamente

Autor: Sistema de Zonificación Automática
Fecha: 2025-11-22
"""

import os
import sys
import json
import logging
from typing import Dict, Any

# Intentar importar psycopg2 (driver PostgreSQL)
try:
    import psycopg2
    from psycopg2.extras import RealDictCursor
except ImportError:
    print("=" * 60)
    print("ERROR: psycopg2 no está instalado")
    print("=" * 60)
    print("\nPara instalar, ejecuta:")
    print("  pip install psycopg2-binary")
    print("\nO si prefieres el paquete completo:")
    print("  pip install psycopg2")
    print("=" * 60)
    sys.exit(1)

# ============================================================
# CONFIGURACIÓN
# ============================================================

# Rutas
MIGRATION_FILE = "./migrations/005_create_cleaning_zones_postgis.sql"
GEOJSON_FILE = "./migrations/zonas_recoleccion.geojson"
LOG_FILE = "./scripts/load_zones.log"

# Configuración de base de datos
# NOTA: Se usa la URL de Neon PostgreSQL del archivo .env
import os
from urllib.parse import urlparse

# Leer DB_URL del archivo .env
def load_db_config_from_env():
    env_path = '.env'
    if os.path.exists(env_path):
        with open(env_path, 'r') as f:
            for line in f:
                if line.startswith('DB_URL='):
                    db_url = line.split('=', 1)[1].strip()
                    parsed = urlparse(db_url)
                    return {
                        'host': parsed.hostname,
                        'port': parsed.port or 5432,
                        'database': parsed.path[1:],  # Remove leading /
                        'user': parsed.username,
                        'password': parsed.password,
                        'sslmode': 'require'
                    }
    # Fallback a configuración local
    return {
        'host': 'localhost',
        'port': 5432,
        'database': 'neondb',
        'user': 'neondb_owner',
        'password': 'postgres'
    }

DB_CONFIG = load_db_config_from_env()

# Mapeo de días (nombre -> número)
DAY_MAPPING = {
    'DOMINGO': 0,
    'LUNES': 1,
    'MARTES': 2,
    'MIERCOLES': 3,
    'MIÉRCOLES': 3,  # Con acento
    'JUEVES': 4,
    'VIERNES': 5,
    'SABADO': 6,
    'SÁBADO': 6,  # Con acento
    'TODOS': 1  # Default a Lunes si no se especifica
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
# FUNCIONES DE BASE DE DATOS
# ============================================================

def get_db_connection():
    """Crea y retorna una conexión a PostgreSQL."""
    try:
        conn = psycopg2.connect(**DB_CONFIG)
        logger.info(f"[OK] Conectado a base de datos: {DB_CONFIG['database']}")
        return conn
    except psycopg2.Error as e:
        logger.error(f"[ERROR] No se pudo conectar a la base de datos: {e}")
        logger.error(f"[INFO] Verifica que PostgreSQL esté corriendo y que los datos de conexión sean correctos:")
        logger.error(f"  Host: {DB_CONFIG['host']}")
        logger.error(f"  Port: {DB_CONFIG['port']}")
        logger.error(f"  Database: {DB_CONFIG['database']}")
        logger.error(f"  User: {DB_CONFIG['user']}")
        sys.exit(1)

def execute_migration(conn):
    """Ejecuta el archivo de migración SQL."""
    logger.info("=" * 60)
    logger.info("EJECUTANDO MIGRACIÓN: 005_create_cleaning_zones_postgis.sql")
    logger.info("=" * 60)
    
    if not os.path.exists(MIGRATION_FILE):
        logger.error(f"[ERROR] No se encuentra el archivo: {MIGRATION_FILE}")
        return False
    
    try:
        with open(MIGRATION_FILE, 'r', encoding='utf-8') as f:
            sql = f.read()
        
        cursor = conn.cursor()
        cursor.execute(sql)
        conn.commit()
        cursor.close()
        
        logger.info("[OK] Migración ejecutada exitosamente")
        return True
        
    except psycopg2.Error as e:
        logger.error(f"[ERROR] Fallo al ejecutar la migración: {e}")
        conn.rollback()
        return False

def check_postgis(conn):
    """Verifica que PostGIS esté instalado."""
    try:
        cursor = conn.cursor()
        cursor.execute("SELECT PostGIS_Version();")
        version = cursor.fetchone()[0]
        cursor.close()
        logger.info(f"[OK] PostGIS instalado - Versión: {version}")
        return True
    except psycopg2.Error as e:
        logger.error(f"[ERROR] PostGIS no está disponible: {e}")
        logger.error("[INFO] Para instalar PostGIS:")
        logger.error("  1. Conecta a PostgreSQL como superusuario")
        logger.error("  2. Ejecuta: CREATE EXTENSION IF NOT EXISTS postgis;")
        return False

# ============================================================
# FUNCIONES DE CARGA DE DATOS
# ============================================================

def extract_day_from_zone_name(zone_name: str) -> int:
    """
    Extrae el número de día (0-6) del nombre de la zona.
    
    Args:
        zone_name: Nombre de la zona (ej: "RUTA 1 LUNES")
        
    Returns:
        Número de día (0=Dom, 1=Lun, ... 6=Sáb)
    """
    zone_upper = zone_name.upper()
    
    for day_name, day_num in DAY_MAPPING.items():
        if day_name in zone_upper:
            return day_num
    
    # Default a Lunes si no se encuentra
    logger.warning(f"[WARNING] No se pudo extraer el día de: {zone_name} - Usando Lunes por defecto")
    return 1

def load_geojson_to_db(conn):
    """Carga los datos del GeoJSON a la base de datos."""
    logger.info("=" * 60)
    logger.info("CARGANDO DATOS DESDE GEOJSON")
    logger.info("=" * 60)
    
    if not os.path.exists(GEOJSON_FILE):
        logger.error(f"[ERROR] No se encuentra el archivo: {GEOJSON_FILE}")
        return False
    
    try:
        # Leer GeoJSON
        with open(GEOJSON_FILE, 'r', encoding='utf-8') as f:
            geojson_data = json.load(f)
        
        features = geojson_data.get('features', [])
        logger.info(f"[INFO] Encontradas {len(features)} zonas en el GeoJSON")
        
        if len(features) == 0:
            logger.warning("[WARNING] No hay zonas para cargar")
            return False
        
        cursor = conn.cursor()
        loaded = 0
        skipped = 0
        
        for feature in features:
            try:
                # Extraer propiedades
                props = feature.get('properties', {})
                zone_name = props.get('zone_name', 'UNKNOWN')
                route_id = props.get('route_id', 'UNKNOWN')
                points_count = props.get('points_count', 0)
                
                # Extraer día del nombre de la zona
                schedule_day = extract_day_from_zone_name(zone_name)
                
                # Convertir geometría a formato GeoJSON string
                geometry = feature.get('geometry')
                if not geometry:
                    logger.warning(f"[SKIP] Zona sin geometría: {zone_name}")
                    skipped += 1
                    continue
                
                geometry_json = json.dumps(geometry)
                
                # Insertar en la base de datos
                sql = """
                INSERT INTO cleaning_zones 
                    (zone_name, route_name, schedule_day, points_count, geom)
                VALUES 
                    (%s, %s, %s, %s, ST_GeomFromGeoJSON(%s))
                ON CONFLICT (route_name, schedule_day) 
                DO UPDATE SET
                    zone_name = EXCLUDED.zone_name,
                    points_count = EXCLUDED.points_count,
                    geom = EXCLUDED.geom,
                    updated_at = CURRENT_TIMESTAMP
                """
                
                cursor.execute(sql, (
                    zone_name,
                    route_id,
                    schedule_day,
                    points_count,
                    geometry_json
                ))
                
                loaded += 1
                logger.info(f"[OK] Cargada zona: {zone_name} (Día: {schedule_day}, Puntos: {points_count})")
                
            except Exception as e:
                logger.error(f"[ERROR] Fallo al cargar zona {zone_name}: {e}")
                skipped += 1
                continue
        
        conn.commit()
        cursor.close()
        
        logger.info("=" * 60)
        logger.info(f"[SUCCESS] Zonas cargadas: {loaded}")
        logger.info(f"[INFO] Zonas omitidas: {skipped}")
        logger.info("=" * 60)
        
        return loaded > 0
        
    except Exception as e:
        logger.error(f"[ERROR] Fallo al cargar GeoJSON: {e}")
        conn.rollback()
        return False

def verify_data(conn):
    """Verifica que los datos se cargaron correctamente."""
    logger.info("=" * 60)
    logger.info("VERIFICANDO DATOS CARGADOS")
    logger.info("=" * 60)
    
    try:
        cursor = conn.cursor(cursor_factory=RealDictCursor)
        
        # Contar zonas totales
        cursor.execute("SELECT COUNT(*) as total FROM cleaning_zones")
        total = cursor.fetchone()['total']
        logger.info(f"[INFO] Total de zonas en DB: {total}")
        
        # Zonas por ruta
        cursor.execute("SELECT * FROM zones_by_route ORDER BY route_name")
        routes = cursor.fetchall()
        logger.info("\n[INFO] Resumen por ruta:")
        for row in routes:
            logger.info(f"  {row['route_name']}: {row['total_zones']} zonas, {row['total_area_km2']:.4f} km²")
        
        # Zonas por día
        cursor.execute("SELECT * FROM zones_by_day ORDER BY schedule_day")
        days = cursor.fetchall()
        logger.info("\n[INFO] Resumen por día:")
        for row in days:
            logger.info(f"  {row['day_name']}: {row['total_zones']} zonas, {row['total_area_km2']:.4f} km²")
        
        # Ejemplo de búsqueda espacial
        logger.info("\n[INFO] Prueba de búsqueda espacial:")
        cursor.execute("""
            SELECT * FROM find_zone_by_point(-0.933, -78.614)
        """)
        result = cursor.fetchone()
        if result:
            logger.info(f"  Punto (-0.933, -78.614) está en: {result['zone_name']}")
            logger.info(f"  Ruta: {result['route_name']}, Día: {result['day_name']}")
        else:
            logger.info("  Punto no encontrado en ninguna zona")
        
        cursor.close()
        logger.info("=" * 60)
        
        return True
        
    except Exception as e:
        logger.error(f"[ERROR] Fallo al verificar datos: {e}")
        return False

# ============================================================
# FUNCIÓN PRINCIPAL
# ============================================================

def main():
    """Función principal del script."""
    logger.info("=" * 60)
    logger.info("SCRIPT: Cargar Zonas de Recolección a PostgreSQL/PostGIS")
    logger.info("=" * 60)
    
    # Paso 1: Conectar a la base de datos
    conn = get_db_connection()
    
    try:
        # Paso 2: Verificar PostGIS
        if not check_postgis(conn):
            logger.error("[ERROR] PostGIS no está disponible. Abortando.")
            return
        
        # Paso 3: Ejecutar migración
        if not execute_migration(conn):
            logger.error("[ERROR] Fallo al ejecutar la migración. Abortando.")
            return
        
        # Paso 4: Cargar datos del GeoJSON
        if not load_geojson_to_db(conn):
            logger.error("[ERROR] Fallo al cargar los datos. Revisa el log para detalles.")
            return
        
        # Paso 5: Verificar datos
        verify_data(conn)
        
        logger.info("=" * 60)
        logger.info("[SUCCESS] ✅ Proceso completado exitosamente!")
        logger.info("=" * 60)
        logger.info("\n[PRÓXIMOS PASOS]:")
        logger.info("  1. Visualiza las zonas en QGIS o pgAdmin")
        logger.info("  2. Prueba la función de búsqueda espacial:")
        logger.info("     SELECT * FROM find_zone_by_point(-0.933, -78.614);")
        logger.info("  3. Consulta las vistas agregadas:")
        logger.info("     SELECT * FROM zones_by_route;")
        logger.info("     SELECT * FROM zones_by_day;")
        
    except Exception as e:
        logger.error(f"[ERROR] Error inesperado: {e}")
        import traceback
        logger.error(traceback.format_exc())
    
    finally:
        conn.close()
        logger.info("\n[INFO] Conexión cerrada")

if __name__ == "__main__":
    main()
