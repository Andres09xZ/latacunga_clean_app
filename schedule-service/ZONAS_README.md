# 🗺️ Zonas Macro de Recolección - Latacunga

## Estado Actual
✅ **5 zonas macro cargadas correctamente en Neon PostgreSQL**

Las zonas son **persistentes** y se mantienen en la base de datos cloud.

## Zonas Cargadas

1. **URBANO_CENTRAL** - NOCTURNO (21:00)
2. **URBANO_NORTE** - DIURNO (Mar-Jue-Sab)
3. **URBANO_SUR** - DIURNO (Lun-Mie-Vie)
4. **RURAL_NORTE** - RURAL (Rutas 2,3,4)
5. **RURAL_SUR** - RURAL (Rutas 1,5)

## ⚠️ Si las zonas se eliminan accidentalmente

Ejecutar el script de restauración:

```powershell
.\scripts\restore_zones.ps1
```

Este script:
1. Verifica y corrige el schema de la BD
2. Carga las 5 zonas desde `migrations/zonas_macro.geojson`
3. Inicializa las métricas (zone_metrics)
4. Verifica que todo esté correcto

## Scripts Disponibles

- `scripts/fix_schema.go` - Agrega columnas faltantes (schedule_config, status)
- `scripts/load_macro_zones.go` - Carga las 5 zonas macro
- `scripts/init_zone_metrics.go` - Inicializa métricas de acumulación
- `scripts/check_db.go` - Verifica zonas en la BD
- `scripts/restore_zones.ps1` - Restauración completa

## Verificar Estado

```powershell
go run scripts/check_db.go
```

Debe mostrar:
```
✅ Total zones in DB: 5
```

## Endpoints API

```bash
# Listar todas las zonas
curl http://localhost:8083/api/v1/planning/zones

# Ver métricas de una zona
curl http://localhost:8083/api/v1/planning/zones/1/metrics
```

## Notas Importantes

- Las zonas están en **Neon PostgreSQL** (cloud, persistente)
- El archivo fuente es `migrations/zonas_macro.geojson`
- **NO ejecutar** `load_zones_to_db.py` - ese script carga 25 zonas diferentes
- Las geometrías son MultiPolygon en formato WGS84 (EPSG:4326)
