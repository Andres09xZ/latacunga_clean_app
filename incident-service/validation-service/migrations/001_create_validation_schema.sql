-- Crear esquema para el servicio de validación
CREATE SCHEMA IF NOT EXISTS validacion;

-- Tabla para almacenar incidentes pendientes de validación
CREATE TABLE IF NOT EXISTS validacion.incidentes_pendientes (
    incidente_id UUID PRIMARY KEY,
    tipo VARCHAR(50) NOT NULL,
    descripcion TEXT NOT NULL,
    ciudadano_id UUID NOT NULL,
    estado VARCHAR(50) NOT NULL DEFAULT 'pendiente_validacion',
    fecha_evento TIMESTAMP WITH TIME ZONE NOT NULL,
    dia_incidente DATE NOT NULL,
    num_fotos INTEGER DEFAULT 0,
    direccion TEXT,
    latitud DECIMAL(10, 8) NOT NULL,
    longitud DECIMAL(11, 8) NOT NULL,
    geometria GEOMETRY(Point, 4326),
    
    -- Datos de validación
    validado_por VARCHAR(100),
    fecha_validacion TIMESTAMP WITH TIME ZONE,
    notas_validacion TEXT,
    
    -- Auditoría
    recibido_en TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    actualizado_en TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Índices para mejorar el rendimiento
CREATE INDEX IF NOT EXISTS idx_incidentes_pendientes_estado 
    ON validacion.incidentes_pendientes(estado);

CREATE INDEX IF NOT EXISTS idx_incidentes_pendientes_fecha 
    ON validacion.incidentes_pendientes(fecha_evento DESC);

CREATE INDEX IF NOT EXISTS idx_incidentes_pendientes_tipo 
    ON validacion.incidentes_pendientes(tipo);

CREATE INDEX IF NOT EXISTS idx_incidentes_pendientes_geometria 
    ON validacion.incidentes_pendientes USING GIST(geometria);

-- Trigger para actualizar 'actualizado_en'
CREATE OR REPLACE FUNCTION validacion.actualizar_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.actualizado_en = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_actualizar_timestamp
    BEFORE UPDATE ON validacion.incidentes_pendientes
    FOR EACH ROW
    EXECUTE FUNCTION validacion.actualizar_timestamp();

COMMENT ON TABLE validacion.incidentes_pendientes IS 'Almacena incidentes recibidos del incident-service que requieren validación manual';
COMMENT ON COLUMN validacion.incidentes_pendientes.estado IS 'Estados: pendiente_validacion, validado, rechazado';
