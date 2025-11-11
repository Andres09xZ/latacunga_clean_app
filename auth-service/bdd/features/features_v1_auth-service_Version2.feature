Feature: Autenticación y control de acceso (JWT + OTP)
  Como sistema
  Quiero autenticar usuarios con JWT y controlar acceso por roles
  Para proteger endpoints y soportar login por OTP para rol "user"

  Background:
    Given existen los roles ["user", "admin", "operador", "trabajador", "super_admin"]
    And access_token expira en 15 minutos
    And refresh_token expira en 30 días
    And OTP: 6 dígitos, expira en 5 minutos, max 5 intentos
    And el endpoint de login es POST "/api/v1/auth/login"
    And el endpoint de refresh es POST "/api/v1/auth/refresh"
    And el endpoint de logout es POST "/api/v1/auth/logout"
    And los endpoints OTP son POST "/api/v1/auth/otp/send" y POST "/api/v1/auth/otp/verify"

  @auth @login
  Scenario Outline: Login exitoso por email/contraseña según rol
    Given existe un usuario con email "<email>" y contraseña "password123" y rol "<role>"
    When hago POST a "/api/v1/auth/login" con:
      """
      { "email": "<email>", "password": "password123" }
      """
    Then la respuesta es 200
    And el cuerpo contiene "access_token" y "refresh_token"
    And el "access_token" contiene claim "role" = "<role>"
    And el "access_token" tiene expiración <= 15 minutos
    And el "refresh_token" tiene expiración <= 30 días

    Examples:
      | email               | role       |
      | user@ciudad.com     | user       |
      | admin@muni.com      | admin      |
      | op1@recolector.com  | operador   |
      | t1@obras.com        | trabajador |

  @auth @login
  Scenario: Login fallido por credenciales incorrectas
    Given existe un usuario con email "user@ciudad.com" y contraseña "password123" y rol "user"
    When hago POST a "/api/v1/auth/login" con:
      """
      { "email": "user@ciudad.com", "password": "mala" }
      """
    Then la respuesta es 401
    And el cuerpo contiene "message" con "Credenciales inválidas"

  @auth @refresh
  Scenario: Refresh token exitoso
    Given que tengo un "refresh_token" válido
    When hago POST a "/api/v1/auth/refresh" con:
      """
      { "refresh_token": "test_refresh_token_valid" }
      """
    Then la respuesta es 200
    And el cuerpo contiene un nuevo "access_token" y un nuevo "refresh_token"

  @auth @refresh
  Scenario: Refresh token inválido o expirado
    Given que tengo un "refresh_token" inválido o expirado
    When hago POST a "/api/v1/auth/refresh" con:
      """
      { "refresh_token": "token_incorrecto" }
      """
    Then la respuesta es 401
    And el cuerpo contiene "message" con "Token inválido o expirado"

  @auth @logout
  Scenario: Logout invalida refresh tokens activos
    Given que estoy autenticado con access_token válido
    And tengo un "refresh_token" activo
    When hago POST a "/api/v1/auth/logout" con:
      """
      { "refresh_token": "test_refresh_token_valid" }
      """
    Then la respuesta es 204
    And el "refresh_token" queda invalidado

  @otp
  Scenario: Solicitud de OTP exitosa (E.164)
    Given que no estoy autenticado
    And existe el número "+593983020282"
    When hago POST a "/api/v1/auth/otp/send" con:
      """
      { "phone": "+593983020282" }
      """
    Then la respuesta es 200
    And el cuerpo contiene "message" con "otp_sent"

  @otp
  Scenario: Verificación de OTP crea usuario si no existe y devuelve tokens
    Given que se solicitó OTP para "+593983020282"
    And el código generado fue "123456"
    And no existe usuario con ese teléfono
    When hago POST a "/api/v1/auth/otp/verify" con:
      """
      { "phone": "+593983020282", "code": "123456" }
      """
    Then la respuesta es 200
    And el sistema crea un usuario con rol "user"
    And el cuerpo contiene "access_token" y "refresh_token"
    And el access_token contiene claim "role" = "user"

  @otp
  Scenario: Verificación de OTP con código incorrecto
    Given que se solicitó OTP para "+593983020282"
    And el código generado fue "123456"
    When hago POST a "/api/v1/auth/otp/verify" con:
      """
      { "phone": "+593983020282", "code": "000000" }
      """
    Then la respuesta es 400
    And el cuerpo contiene "message" con "OTP inválido"

  @otp
  Scenario: Límite de intentos de OTP excedido
    Given que se solicitó OTP para "+593983020282"
    And he realizado 5 intentos fallidos
    When hago POST a "/api/v1/auth/otp/verify" con:
      """
      { "phone": "+593983020282", "code": "111111" }
      """
    Then la respuesta es 429
    And el cuerpo contiene "message" con "Límite de intentos excedido, solicite un nuevo OTP"

  @otp
  Scenario: Verificación de OTP con código incorrecto
    Given que se solicitó OTP para "+593983020282"
    And el código generado fue "123456"
    When hago POST a "/api/v1/auth/otp/verify" con:
      """
      { "phone": "+593983020282", "code": "000000" }
      """
    Then la respuesta es 400
    And el cuerpo contiene "message" con "OTP inválido"

  @otp
  Scenario: Límite de intentos de OTP excedido
    Given que se solicitó OTP para "+593983020282"
    And he realizado 5 intentos fallidos
    When hago POST a "/api/v1/auth/otp/verify" con:
      """
      { "phone": "+593983020282", "code": "111111" }
      """
    Then la respuesta es 429
    And el cuerpo contiene "message" con "Límite de intentos excedido, solicite un nuevo OTP"

  @roles @register
  Scenario Outline: Registro por formulario controlado por super_admin
    Given que estoy autenticado con access_token rol "super_admin"
    When hago POST a "/api/v1/auth/register" con:
      """
      { "email": "<email>", "password": "password123", "role": "<role>" }
      """
    Then la respuesta es 201
    And el nuevo usuario queda registrado con rol "<role>"

    Examples:
      | email               | role       |
      | oper1@ciudad.com    | operador   |
      | trab1@obras.com     | trabajador |
      | admin2@muni.com     | admin      |

  @roles @register
  Scenario: Registro público acepta rol solicitado (no user)
    Given que no estoy autenticado
    When hago POST a "/api/v1/auth/register" con:
      """
      { "email": "nuevo@ciudad.com", "password": "password123", "role": "admin" }
      """
    Then la respuesta es 201
    And el usuario se registra con rol "admin"

  @security
  Scenario: Access token expirado devuelve 401
    Given que estoy autenticado con access_token expirado
    When hago GET a "/api/v1/admin/users"
    Then la respuesta es 401
    And el cuerpo contiene "message" con "Token expirado"

  @security
  Scenario: JWT mal firmado es rechazado
    Given que presento un access_token con firma inválida
    When hago GET a "/api/v1/admin/users"
    Then la respuesta es 401
    And el cuerpo contiene "message" con "Token inválido"