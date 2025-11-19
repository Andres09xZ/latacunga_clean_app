# Lenguaje: es
@servicio_usuario @otp @ciudadano
Característica: Autenticación OTP para ciudadanos
  Como ciudadano
  Quiero acceder mediante OTP
  Para poder emitir novedades rápidamente

  Esquema del Escenario: Validación de OTP
    Dado que registro mi teléfono "<telefono>"
    Cuando solicito un OTP
    Entonces se crea una entrada en otp_requests con estado "requested"
    Y se envía el OTP vía Twilio
    Y recibo un código OTP válido por 5 minutos

    Cuando envío el OTP "<codigo>"
    Entonces si es válido antes de expirar
    Y la verificación es exitosa
    Entonces mi citizen queda verificado con verified_at registrado
    Y recibo un token con scope "novedades:crear"
    Y puedo crear reportes de novedades

    Ejemplos:
      | telefono      | codigo |
      | +593999000111 | 123456 |
      | +593987654321 | 654321 |

  Escenario: Rate limiting de OTP
    Dado que solicito múltiples OTP en menos de 1 minuto
    Cuando supero el límite de 3 solicitudes por minuto
    Entonces el sistema rechaza nuevas solicitudes con estado 429
    Y registra el evento en otp_requests con status "failed"
    Y registra el error_code "RATE_LIMIT_EXCEEDED"

  Escenario: Expiración de OTP
    Dado que solicité un OTP hace 6 minutos
    Cuando intento verificarlo
    Entonces el sistema rechaza con error "OTP_EXPIRED"
    Y actualiza el status en otp_requests a "expired"

  Escenario: Intentos fallidos de OTP
    Dado que he intentado 5 veces con códigos incorrectos
    Cuando intento una sexta vez
    Entonces el sistema bloquea con error "MAX_ATTEMPTS_EXCEEDED"
    Y registra attempts = 6 en otp_requests

# Lenguaje: es
@servicio_usuario @operadores @admin
Característica: Autenticación de operadores y administradores
  Como operador o administrador
  Quiero registrarme e iniciar sesión
  Para gestionar mis tareas y acceder al sistema

  Escenario: Registro de operador válido
    Dado que envío nombre, email y contraseña válidos
    Y la contraseña tiene al menos 8 caracteres
    Cuando creo una cuenta de operador
    Entonces se crea una entrada en operators con role "operador"
    Y la contraseña se guarda hasheada
    Y recibo confirmación de registro exitoso

  Escenario: Registro duplicado falla
    Dado que ya existe un operador con email "juan@ejemplo.com"
    Cuando intento registrarme con el mismo email
    Entonces el sistema rechaza con error "EMAIL_ALREADY_EXISTS"
    Y devuelve estado 409 Conflict

  Escenario: Registro de admin
    Dado que soy un usuario autorizado
    Y envío role = "admin" en el registro
    Cuando creo una cuenta de administrador
    Entonces se crea en operators con role "admin"
    Y tiene permiso total sobre el sistema

  Escenario: Inicio de sesión de operador
    Dado que tengo credenciales válidas
    Y mi cuenta está active = true
    Cuando inicio sesión con email y contraseña
    Entonces recibo un token JWT con claims:
      | claim      | valor                    |
      | sub        | <operator_id>            |
      | email      | juan@ejemplo.com         |
      | role       | operador                 |
      | scope      | tareas:gestionar         |
      | exp        | +8 horas desde ahora     |
    Y el token tiene tipo "Bearer"

  Escenario: Falso de operador inactivo
    Dado que mi cuenta está active = false
    Cuando intento iniciar sesión
    Entonces recibo error "ACCOUNT_INACTIVE"
    Y no recibo token

  Escenario: Falso de credenciales incorrectas
    Dado que intento iniciar sesión
    Cuando uso email correcto pero contraseña incorrecta
    Entonces recibo error "INVALID_CREDENTIALS"
    Y no revelo información de qué campo fue incorrecto

# Lenguaje: es
@servicio_usuario @idempotencia
Característica: Idempotencia en endpoints críticos
  Como cliente de la API
  Quiero que mis solicitudes sean idempotentes
  Para prevenir duplicados accidentales

  Escenario: Solicitud idempotente detecta duplicado
    Dado que envío una solicitud POST con header "Idempotency-Key: abc123"
    Cuando se procesa exitosamente
    Entonces se registra en idempotency_keys la respuesta
    Y si reenvío la misma solicitud con el mismo Idempotency-Key
    Entonces recibo la MISMA respuesta sin procesar nuevamente
    Y la operación no se duplica

  Escenario: Idempotency-Key expira después de 24 horas
    Dado que tengo un Idempotency-Key registrado hace 25 horas
    Cuando hago una nueva solicitud con la misma clave
    Entonces se procesa como nueva solicitud
    Y se crea nuevo registro en idempotency_keys

# Lenguaje: es
@servicio_usuario @eventos @outbox
Característica: Event Sourcing con patrón Outbox
  Como sistema
  Quiero mantener consistencia eventual entre servicios
  Para asegurar que los eventos se publiquen correctamente

  Escenario: Evento de usuario registrado
    Dado que un ciudadano se registra con teléfono "+593999000111"
    Cuando se verifica exitosamente
    Entonces se crea evento en outbox_events con:
      | campo           | valor                      |
      | aggregate_type  | citizen                    |
      | type            | citizen.verified           |
      | status          | pending                    |
      | payload         | {"phone":"+593999000111"}  |
    Y el evento se publica a otros servicios

  Escenario: Evento de operador creado
    Dado que se registra un nuevo operador
    Cuando se completa el registro
    Entonces se crea evento con type "operator.created"
    Y el status inicia como "pending"
    Y se publica cuando se marca status = "published"

  Escenario: Reintentos en eventos fallidos
    Dado que un evento quedó con status "failed"
    Y se reintentó 3 veces
    Cuando supera máximo de reintentos
    Entonces el evento se marca para revisión manual
    Y se registra en monitoreo
