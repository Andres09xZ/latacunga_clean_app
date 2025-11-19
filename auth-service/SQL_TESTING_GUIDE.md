-- SQL Testing Guide - Auth Service
-- Consultas útiles para verificar data y testing manual

-- =============================================
-- VERIFICACIÓN DE ESQUEMA
-- =============================================

-- Ver todas las tablas del schema usuario
\dt usuario.*

-- Ver estructura de operadores
\d usuario.operators

-- Ver estructura de ciudadanos
\d usuario.citizens

-- Ver estructura de OTP requests
\d usuario.otp_requests

-- Ver estructura de eventos
\d usuario.outbox_events

-- =============================================
-- OPERADORES
-- =============================================

-- Ver todos los operadores
SELECT id, name, email, role, active, created_at 
FROM usuario.operators 
ORDER BY created_at DESC;

-- Ver solo operadores activos
SELECT * FROM usuario.operators 
WHERE active = true;

-- Buscar operador por email
SELECT * FROM usuario.operators 
WHERE email = 'juan@ejemplo.com';

-- Ver operadores por role
SELECT COUNT(*) as total, role 
FROM usuario.operators 
GROUP BY role;

-- Contar operadores activos por role
SELECT role, COUNT(*) as active_count 
FROM usuario.operators 
WHERE active = true 
GROUP BY role;

-- =============================================
-- CIUDADANOS
-- =============================================

-- Ver todos los ciudadanos
SELECT id, phone_e164, verified_at, created_at 
FROM usuario.citizens 
ORDER BY created_at DESC;

-- Ver ciudadanos verificados
SELECT * FROM usuario.citizens 
WHERE verified_at IS NOT NULL;

-- Ver ciudadanos sin verificar
SELECT * FROM usuario.citizens 
WHERE verified_at IS NULL;

-- Buscar ciudadano por teléfono
SELECT * FROM usuario.citizens 
WHERE phone_e164 = '+593999000111';

-- Contar ciudadanos por estado de verificación
SELECT 
  COUNT(*) as total,
  COUNT(verified_at) as verified,
  COUNT(*) - COUNT(verified_at) as unverified
FROM usuario.citizens;

-- =============================================
-- OTP REQUESTS
-- =============================================

-- Ver todas las solicitudes OTP
SELECT id, phone_e164, status, attempts, requested_at, verified_at, error_code
FROM usuario.otp_requests 
ORDER BY requested_at DESC 
LIMIT 20;

-- Ver OTP solicitudes por teléfono
SELECT * FROM usuario.otp_requests 
WHERE phone_e164 = '+593999000111' 
ORDER BY requested_at DESC;

-- Ver OTP pendientes (sin verificar)
SELECT * FROM usuario.otp_requests 
WHERE status = 'requested' 
ORDER BY requested_at DESC;

-- Ver OTP verificadas
SELECT * FROM usuario.otp_requests 
WHERE status = 'verified' 
ORDER BY verified_at DESC;

-- Ver OTP fallidas
SELECT phone_e164, COUNT(*) as failed_count, error_code
FROM usuario.otp_requests 
WHERE status = 'failed' 
GROUP BY phone_e164, error_code;

-- Ver OTP expiradas (más de 5 minutos)
SELECT * FROM usuario.otp_requests 
WHERE status = 'requested' 
  AND requested_at < NOW() - INTERVAL '5 minutes'
ORDER BY requested_at DESC;

-- Ver tasa de éxito de OTP
SELECT 
  COUNT(*) as total,
  COUNT(CASE WHEN status = 'verified' THEN 1 END) as verified,
  COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
  COUNT(CASE WHEN status = 'expired' THEN 1 END) as expired,
  ROUND(100.0 * COUNT(CASE WHEN status = 'verified' THEN 1 END) / 
    COUNT(*), 2) as success_rate_percent
FROM usuario.otp_requests;

-- Ver número de intentos por OTP
SELECT 
  id, phone_e164, status, attempts, 
  CASE 
    WHEN attempts >= 5 THEN 'BLOCKED'
    WHEN attempts >= 3 THEN 'WARNING'
    ELSE 'OK'
  END as attempt_status
FROM usuario.otp_requests 
WHERE requested_at > NOW() - INTERVAL '1 hour'
ORDER BY attempts DESC;

-- =============================================
-- IDEMPOTENCIA
-- =============================================

-- Ver todas las claves de idempotencia
SELECT key, request_fingerprint, created_at 
FROM usuario.idempotency_keys 
ORDER BY created_at DESC 
LIMIT 20;

-- Limpiar claves de idempotencia expiradas (>24h)
DELETE FROM usuario.idempotency_keys 
WHERE created_at < NOW() - INTERVAL '24 hours';

-- Contar claves por hora
SELECT DATE_TRUNC('hour', created_at) as hour, COUNT(*) as total
FROM usuario.idempotency_keys 
GROUP BY hour 
ORDER BY hour DESC;

-- =============================================
-- OUTBOX EVENTS
-- =============================================

-- Ver todos los eventos
SELECT id, aggregate_type, aggregate_id, type, status, created_at, published_at
FROM usuario.outbox_events 
ORDER BY created_at DESC 
LIMIT 50;

-- Ver eventos pendientes por publicar
SELECT id, aggregate_type, type, created_at 
FROM usuario.outbox_events 
WHERE status = 'pending' 
ORDER BY created_at ASC;

-- Ver eventos fallidos
SELECT id, aggregate_type, type, created_at 
FROM usuario.outbox_events 
WHERE status = 'failed' 
ORDER BY created_at DESC;

-- Ver eventos publicados
SELECT COUNT(*) as published 
FROM usuario.outbox_events 
WHERE status = 'published';

-- Ver eventos por tipo
SELECT type, COUNT(*) as total
FROM usuario.outbox_events 
GROUP BY type 
ORDER BY total DESC;

-- Ver timeline de eventos de un ciudadano
SELECT type, status, created_at, published_at
FROM usuario.outbox_events 
WHERE aggregate_type = 'citizen' 
  AND aggregate_id = '550e8400-e29b-41d4-a716-446655440000'::uuid
ORDER BY created_at ASC;

-- Marcar evento como publicado
UPDATE usuario.outbox_events 
SET status = 'published', published_at = NOW()
WHERE id = '550e8400-e29b-41d4-a716-446655440000'::uuid;

-- =============================================
-- QUERIES DE AUDITORÍA
-- =============================================

-- Línea de tiempo: últimos accesos exitosos
SELECT 
  a.created_at,
  'OPERATOR_LOGIN' as event_type,
  e.payload::jsonb->>'email' as email,
  NULL as phone
FROM usuario.outbox_events e
WHERE e.type = 'operator.login' 
  AND e.status = 'published'
UNION ALL
SELECT
  a.created_at,
  'CITIZEN_VERIFIED' as event_type,
  NULL as email,
  e.payload::jsonb->>'phone_e164' as phone
FROM usuario.outbox_events e
WHERE e.type = 'citizen.verified'
  AND e.status = 'published'
ORDER BY created_at DESC
LIMIT 20;

-- Ver operadores más activos (por logins)
SELECT 
  e.payload::jsonb->>'email' as email,
  COUNT(*) as login_count,
  MAX(e.published_at) as last_login
FROM usuario.outbox_events e
WHERE e.type = 'operator.login' 
  AND e.status = 'published'
GROUP BY email
ORDER BY login_count DESC;

-- Ver teléfonos con más intentos fallidos de OTP
SELECT 
  phone_e164,
  COUNT(*) as total_requests,
  COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
  COUNT(CASE WHEN status = 'verified' THEN 1 END) as verified,
  COUNT(CASE WHEN status = 'expired' THEN 1 END) as expired,
  ROUND(100.0 * COUNT(CASE WHEN status = 'failed' THEN 1 END) / 
    COUNT(*), 2) as failure_rate_percent
FROM usuario.otp_requests 
GROUP BY phone_e164
ORDER BY failed DESC
LIMIT 20;

-- =============================================
-- TESTING DATA CLEANUP
-- =============================================

-- Eliminar todos los eventos de test
DELETE FROM usuario.outbox_events 
WHERE created_at > NOW() - INTERVAL '1 hour';

-- Eliminar todas las claves de idempotencia de test
DELETE FROM usuario.idempotency_keys 
WHERE created_at > NOW() - INTERVAL '1 hour';

-- Eliminar solicitudes OTP recientes
DELETE FROM usuario.otp_requests 
WHERE requested_at > NOW() - INTERVAL '1 hour';

-- Eliminar ciudadano de test (CUIDADO: cascade puede afectar)
DELETE FROM usuario.citizens 
WHERE phone_e164 LIKE '+593999%';

-- Eliminar operador de test
DELETE FROM usuario.operators 
WHERE email LIKE '%test%' OR email LIKE '%ejemplo%';

-- =============================================
-- STATS Y MONITOREO
-- =============================================

-- Resumen general del servicio
SELECT 
  'Operadores Total' as metric, COUNT(*) as value
FROM usuario.operators
UNION ALL
SELECT 
  'Operadores Activos', COUNT(*)
FROM usuario.operators 
WHERE active = true
UNION ALL
SELECT 
  'Ciudadanos Total', COUNT(*)
FROM usuario.citizens
UNION ALL
SELECT 
  'Ciudadanos Verificados', COUNT(*)
FROM usuario.citizens 
WHERE verified_at IS NOT NULL
UNION ALL
SELECT 
  'OTP Solicitudes (24h)', COUNT(*)
FROM usuario.otp_requests 
WHERE requested_at > NOW() - INTERVAL '24 hours'
UNION ALL
SELECT 
  'OTP Verificadas (24h)', COUNT(*)
FROM usuario.otp_requests 
WHERE status = 'verified' 
  AND verified_at > NOW() - INTERVAL '24 hours'
UNION ALL
SELECT 
  'Eventos Pendientes', COUNT(*)
FROM usuario.outbox_events 
WHERE status = 'pending'
UNION ALL
SELECT 
  'Eventos Fallidos', COUNT(*)
FROM usuario.outbox_events 
WHERE status = 'failed';

-- Dashboard: actividad por hora
SELECT 
  DATE_TRUNC('hour', created_at) as hour,
  COUNT(*) as otp_requests,
  COUNT(CASE WHEN status = 'verified' THEN 1 END) as verified,
  COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
FROM usuario.otp_requests 
WHERE requested_at > NOW() - INTERVAL '24 hours'
GROUP BY hour
ORDER BY hour DESC;

-- Dashboard: errores más comunes
SELECT 
  error_code,
  COUNT(*) as occurrences,
  ROUND(100.0 * COUNT(*) / (SELECT COUNT(*) FROM usuario.otp_requests 
    WHERE error_code IS NOT NULL), 2) as percentage
FROM usuario.otp_requests 
WHERE error_code IS NOT NULL
GROUP BY error_code
ORDER BY occurrences DESC;

-- =============================================
-- PERFORMANCE CHECKS
-- =============================================

-- Tamaño de tablas
SELECT 
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname = 'usuario'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Índices disponibles
SELECT 
  indexname,
  tablename,
  indexdef
FROM pg_indexes
WHERE schemaname = 'usuario'
ORDER BY tablename, indexname;

-- Consultas lentas (si está habilitado query logging)
-- SELECT * FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;
