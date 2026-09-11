# Autenticación y gestión de usuarios — PATO User Service

## 1. Descripción

El microservicio de usuarios de PATO implementa autenticación basada en **sesiones opacas almacenadas en Valkey** (compatible con Redis), no en JWT autocontenido.

El flujo permite:

* Registrar usuarios.
* Validar los datos de registro.
* Almacenar contraseñas utilizando hashing con BCrypt.
* Verificar el correo electrónico mediante un código numérico temporal.
* Iniciar sesión utilizando correo y contraseña.
* Generar un token de sesión opaco tras un login exitoso y almacenarlo en Valkey.
* Renovar automáticamente la sesión con cada petición autenticada.
* Cerrar sesión (logout) invalidando el token en Valkey.
* Recuperar la contraseña mediante un flujo de "olvidé mi contraseña".
* Utilizar el token de sesión para proteger endpoints.

La arquitectura mantiene la separación de responsabilidades entre:

```text
Handler → Service → Model/Database
              ↓
           Valkey (sesiones y tokens)
```

---

# 2. Arquitectura involucrada

```text
internal/
├── config/
│   └── config.go
│
├── database/
│   ├── database.go
│   └── valkey.go
│
├── dto/
│   └── user.go
│
├── handlers/
│   └── users.go
│
├── middleware/
│   └── auth.go
│
├── models/
│   ├── user.go
│   └── email_token.go
│
├── services/
│   ├── user_service.go
│   └── email.go
│
└── utils/
    ├── security.go
    └── jwt.go
```

### Responsabilidad de cada componente

| Componente        | Responsabilidad                                          |
| ------------------ | --------------------------------------------------------- |
| `config`           | Cargar variables de entorno y configuración de sesión     |
| `database`         | Conectar con PostgreSQL                                   |
| `database/valkey`  | Conectar con Valkey (sesiones, códigos y tokens temporales) |
| `dto`              | Definir los datos recibidos y enviados por la API          |
| `models`           | Representar las tablas de la base de datos                 |
| `handlers`         | Recibir peticiones HTTP y devolver respuestas               |
| `middleware/auth`  | Validar el token de sesión en endpoints protegidos          |
| `services`         | Contener la lógica de negocio                               |
| `utils/security`   | Hash y validación de contraseñas, generación de tokens       |
| `services/email`   | Comunicación con el servicio de correos                      |

---

# 3. Configuración mediante `.env`

Ejemplo:

```env
# Database
APP_DB_IP=tu_ip
APP_DB_PORT=tu_puerto
APP_DB_USER=tu_usuario_db
APP_DB_PASSWORD=tu_contraseña_muy_segura
APP_DB_NAME=nombre_de_tu_db

# Valkey
APP_VALKEY_ADDR=localhost:6379
APP_VALKEY_USER=
APP_VALKEY_PASSWORD=
APP_VALKEY_DB=0

# Server
APP_PORT=8080
APP_ENV=development
APP_NAME=PATO User Service

# Email service
EMAIL_SERVICE_URL=http://localhost:8000
APP_BASE_URL=http://localhost:3000
```

El archivo `.env` **no debe subirse al repositorio**, por lo que debe estar incluido en `.gitignore`:

```gitignore
.env
```

---

# 4. Sesiones y Valkey

A diferencia de un JWT tradicional, el token de sesión es un **token opaco aleatorio**: no contiene información del usuario, solo sirve como clave para buscar la sesión en Valkey.

En `internal/config/config.go`:

```go
const SessionTTL = 20 * 24 * time.Hour
```

Flujo de creación de sesión (`UserService.Login`):

```text
Login exitoso
   ↓
GenerateSessionToken()  → token aleatorio (crypto/rand)
   ↓
HashToken(token)        → SHA-256
   ↓
Valkey.Set("session:<hash>", userID, TTL=20 días)
   ↓
Se devuelve el token (sin hashear) al cliente
```

Solo el **hash** del token se almacena en Valkey; el token en texto plano nunca se persiste, igual que ocurre con los códigos de verificación y de recuperación de contraseña.

Cada vez que el middleware valida una sesión, **renueva el TTL** (sliding expiration):

```go
s.Valkey.Expire(ctx, key, config.SessionTTL)
```

Por lo tanto, una sesión activa se mantiene viva mientras el usuario siga haciendo peticiones; si permanece inactiva 20 días, expira automáticamente.

---

# 5. Seguridad de contraseñas

Las contraseñas no se almacenan directamente en la base de datos.

Se utiliza **BCrypt** mediante:

```go
bcrypt.GenerateFromPassword()
```

```go
func HashPassword(password string) (string, error)
func VerifyPassword(hashedPassword, password string) bool
```

El flujo es:

```text
Contraseña ingresada
        ↓
     BCrypt
        ↓
   PasswordHash
        ↓
     PostgreSQL
```

---

# 6. Validación de contraseñas

Antes de crear un usuario, o al cambiar/reestablecer la contraseña, se valida que cumpla los requisitos establecidos.

Actualmente debe:

* Tener al menos 8 caracteres.
* Contener al menos una letra mayúscula.
* Contener al menos un número.
* Contener al menos un carácter especial.

Ejemplo válido:

```text
Password123!
```

La validación se encuentra en:

```text
internal/utils/security.go
```

---

# 7. Flujo de Login

```text
POST /api/v1/users/login
          │
          ▼
    UsersHandler.Login()
          │
          ▼
     LoginRequest
          │
          ▼
    UserService.Login()
          │
          ├── Buscar usuario por email
          │
          ├── Verificar contraseña
          │
          ├── Rechazar si email_verified = false
          │
          ├── Generar token de sesión aleatorio
          │
          └── Guardar hash del token en Valkey (TTL 20 días)
                    │
                    ▼
              Respuesta HTTP (token + expires_in)
                    │
                    ▼
                 Cliente
```

La ruta se registra en `cmd/api/main.go`:

```go
users.POST("/login", usersHandler.Login)
```

---

# 8. Endpoint de Login

### URL

```text
POST http://localhost:8080/api/v1/users/login
```

### Body

```json
{
    "email": "natalia@gmail.com",
    "password": "Password123!"
}
```

### Respuesta exitosa

```json
{
    "token": "d290f1ee-6c54-4b01-90e6-d701748f0851...",
    "expires_in": 1728000
}
```

`expires_in` está expresado en segundos (20 días = 1 728 000 segundos).

### Respuesta si el correo no está verificado

```json
{
    "error": "email not verified",
    "code": "EMAIL_NOT_VERIFIED"
}
```

HTTP `403 Forbidden`.

---

# 9. Verificación del correo antes del Login

`UserService.Login` exige `email_verified = true`: si el usuario existe y la contraseña es correcta pero la cuenta no ha verificado su correo, el login se rechaza con `403 EMAIL_NOT_VERIFIED` (ver `internal/services/user_service.go`, `ErrEmailNotVerified`). El registro genera un código de verificación que debe usarse para marcar `email_verified = true` antes de poder iniciar sesión.

Durante el registro se genera un código numérico:

```go
const emailVerificationTokenTTL = 15 * time.Minute
```

El código se guarda en Valkey bajo la clave:

```text
pato:email-verification:<email>
```

Flujo:

```text
Registro
   ↓
Crear usuario
   ↓
Generar código de verificación
   ↓
Guardar en Valkey (TTL 15 min)
   ↓
Enviar correo
   ↓
Usuario verifica email
   ↓
email_verified = true
   ↓
Código eliminado de Valkey
```

---

# 10. Tipos de tokens usados en el sistema

| Token                     |   Duración | Almacenamiento          | Uso                          |
| -------------------------- | ---------: | ------------------------ | ----------------------------- |
| Código de verificación de email | 15 minutos | Valkey (`pato:email-verification:<email>`) | Verificar correo |
| Token de sesión             |    20 días | Valkey (`session:<hash>`) | Autenticación                |
| Token de recuperación de contraseña | 15 minutos | Valkey (`password_reset:<hash>`) | Reestablecer contraseña |

Todos los tokens/códigos son aleatorios (generados con `crypto/rand`) y, salvo el código de verificación, se almacenan hasheados con SHA-256 mediante `utils.HashToken`.

---

# 11. Middleware de autenticación

Para proteger endpoints se utiliza un middleware que:

1. Obtiene el header `Authorization`.
2. Comprueba que utilice el esquema `Bearer`.
3. Extrae el token de sesión.
4. Consulta Valkey (`ValidateSession`) para resolver el `userID` asociado.
5. Renueva el TTL de la sesión (sliding expiration).
6. Permite continuar si la sesión es válida.

Flujo:

```text
Request
  │
  ▼
Authorization: Bearer <token>
  │
  ▼
AuthMiddleware
  │
  ├── ¿Existe token?
  │       └── No → 401
  │
  ├── ¿Sesión existe en Valkey?
  │       └── No → 401 (expirada o inválida)
  │
  └── Sí
       ├── Renovar TTL
       ├── c.Set("userID", ...)
       ├── c.Set("sessionToken", ...)
       ↓
    Endpoint
```

---

# 12. Logout

Permite invalidar la sesión activa eliminándola de Valkey.

### Handler y ruta

```go
protected.POST("/logout", usersHandler.Logout)
```

```text
POST /api/v1/users/logout
```

Requiere autenticación:

```http
Authorization: Bearer <token>
```

### Flujo (`UserService.Logout`)

```text
POST /api/v1/users/logout
          │
          ▼
   AuthMiddleware (extrae sessionToken)
          │
          ▼
   UserService.Logout()
          │
          └── Valkey.Del("session:<hash>")
                    │
                    ▼
              Respuesta HTTP
```

### Respuesta exitosa

```json
{
    "message": "Logged out successfully"
}
```

Tras el logout, el token deja de ser válido inmediatamente, aunque no haya expirado su TTL.

---

# 13. Recuperación de contraseña (Forgot / Reset)

## 13.1 Forgot Password

```text
POST /api/v1/users/forgot-password
```

### Body

```json
{
    "email": "natalia@gmail.com"
}
```

### Flujo (`UserService.ForgotPassword`)

```text
POST /api/v1/users/forgot-password
          │
          ▼
   UserService.ForgotPassword()
          │
          ├── Buscar usuario por email
          │      └── No existe → responder OK igualmente (no revelar existencia)
          │
          ├── Generar código de recuperación
          │
          ├── Guardar hash en Valkey ("password_reset:<hash>", TTL 15 min)
          │
          └── Enviar correo con el código
                    │
                    ▼
              Respuesta HTTP
```

### Respuesta

```json
{
    "message": "If the email is registered, a password reset token has been sent"
}
```

La respuesta es siempre la misma exista o no el correo, para evitar filtrar qué correos están registrados.

## 13.2 Reset Password

```text
POST /api/v1/users/reset-password
```

### Body

```json
{
    "token": "CODIGO_RECIBIDO",
    "password": "NuevaPassword456!",
    "confirm_password": "NuevaPassword456!"
}
```

### Flujo (`UserService.ResetPassword`)

```text
POST /api/v1/users/reset-password
          │
          ▼
   UserService.ResetPassword()
          │
          ├── Validar complejidad de la nueva contraseña
          │
          ├── Buscar token en Valkey ("password_reset:<hash>")
          │      └── No existe / expirado → error
          │
          ├── Hashear la nueva contraseña (BCrypt)
          │
          ├── Actualizar password_hash en base de datos
          │
          └── Eliminar el token de Valkey (uso único)
                    │
                    ▼
              Respuesta HTTP
```

### Respuesta exitosa

```json
{
    "message": "Password reset successfully"
}
```

---

# 14. Pruebas realizadas con Postman

## 14.1 Health Check

### Request

```http
GET http://localhost:8080/health
```

### Respuesta esperada

```json
{
    "status": "healthy",
    "app": "PATO User Service",
    "valkey": "healthy"
}
```

---

## 14.2 Registro

### Request

```http
POST http://localhost:8080/api/v1/users/register
```

### Body

```json
{
    "username": "natalia",
    "email": "natalia@gmail.com",
    "password": "Password123!",
    "confirm_password": "Password123!"
}
```

### Resultado

El usuario es almacenado en PostgreSQL y se genera el proceso de verificación del correo.

---

## 14.3 Verificación de correo

### Request

```http
POST http://localhost:8080/api/v1/users/verify-email
```

### Body

```json
{
    "email": "natalia@gmail.com",
    "token": "CODIGO_DE_VERIFICACION"
}
```

### Respuesta

```json
{
    "message": "Email verified successfully"
}
```

---

## 14.4 Login

### Request

```http
POST http://localhost:8080/api/v1/users/login
```

### Body

```json
{
    "email": "natalia@gmail.com",
    "password": "Password123!"
}
```

### Resultado esperado

```json
{
    "token": "d290f1ee-6c54-4b01-90e6-d701748f0851...",
    "expires_in": 1728000
}
```

---

## 14.5 Uso del token de sesión en Postman

1. Crear una petición.
2. Ir a **Authorization**.
3. Seleccionar **Bearer Token**.
4. Pegar el token obtenido durante el login.
5. Ejecutar la petición.

```http
Authorization: Bearer d290f1ee-6c54-4b01-90e6-d701748f0851...
```

---

## 14.6 Prueba de acceso sin token

```text
GET /api/v1/users/me
```

sin enviar el header `Authorization`, el servidor debe responder:

```text
401 Unauthorized
```

## 14.7 Prueba de acceso con token válido / logout

```text
Sin token           → 401 Unauthorized
Con token válido     → 200 OK
Con token inválido   → 401 Unauthorized
Tras hacer logout    → 401 Unauthorized (aunque el TTL no haya vencido)
```

---

# 15. Flujo completo del sistema

```text
                    ┌──────────────┐
                    │    Cliente   │
                    └──────┬───────┘
                           │
                           │ Register
                           ▼
                    ┌──────────────┐
                    │   Handler    │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ UserService  │
                    └──────┬───────┘
                           │
                  ┌────────┴────────┐
                  ▼                 ▼
             PostgreSQL       Email Service
                  │                 │
                  └────────┬────────┘
                           │
                      Verificación
                           │
                           ▼
                         Login
                           │
                           ▼
                    Verificar password
                           │
                           ▼
                Generar token de sesión
                           │
                           ▼
                 Guardar sesión en Valkey
                           │
                           ▼
                    Cliente autenticado
                           │
                           │ Bearer Token
                           ▼
                    AuthMiddleware
                           │
                     ┌─────┴─────┐
                     │           │
                Válido en     No existe/
                 Valkey        expirado
                     │           │
                     ▼           ▼
                  Endpoint      401
                     │
                     ▼
                  Logout
                     │
                     ▼
           Eliminar sesión de Valkey
```

---

# 16. Resumen de endpoints

| Método  | Endpoint                          | Descripción                        | Autenticación |
| ------- | ---------------------------------- | ------------------------------------ | -------------- |
| `GET`   | `/health`                          | Comprobar estado del servicio (incluye Valkey) | No  |
| `POST`  | `/api/v1/users/register`           | Registrar usuario                    | No             |
| `POST`  | `/api/v1/users/verify-email`       | Verificar correo                     | No             |
| `POST`  | `/api/v1/users/login`              | Iniciar sesión y obtener token       | No             |
| `POST`  | `/api/v1/users/forgot-password`    | Solicitar recuperación de contraseña | No             |
| `POST`  | `/api/v1/users/reset-password`     | Reestablecer contraseña con token    | No             |
| `GET`   | `/api/v1/users`                    | Listar todos los usuarios            | Sesión         |
| `GET`   | `/api/v1/users/me`                 | Obtener datos del usuario autenticado | Sesión        |
| `POST`  | `/api/v1/users/logout`             | Cerrar sesión                        | Sesión         |
| `PATCH` | `/api/v1/users/email`              | Cambiar el correo del usuario        | Sesión         |
| `PATCH` | `/api/v1/users/username`           | Cambiar el nombre de usuario         | Sesión         |
| `PATCH` | `/api/v1/users/password`           | Cambiar la contraseña del usuario    | Sesión         |

---

# 17. Cambio de correo electrónico

Permite a un usuario autenticado actualizar su correo, confirmando su identidad con la contraseña actual.

### Handler y ruta

```go
protected.PATCH("/email", usersHandler.UpdateEmail)
```

```text
PATCH /api/v1/users/email
```

Requiere autenticación:

```http
Authorization: Bearer <token>
```

### DTO

```go
type UpdateEmailRequest struct {
    NewEmail string `json:"new_email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}
```

### Body

```json
{
    "new_email": "nueva-natalia@gmail.com",
    "password": "Password123!"
}
```

### Flujo (`UserService.UpdateEmail`)

```text
PATCH /api/v1/users/email
          │
          ▼
   AuthMiddleware (extrae userID de la sesión)
          │
          ▼
    UpdateEmailRequest
          │
          ▼
   UserService.UpdateEmail()
          │
          ├── Buscar usuario por ID
          │
          ├── Verificar contraseña actual
          │
          ├── Rechazar si new_email == email actual
          │
          ├── Rechazar si new_email ya está registrado
          │
          ├── Actualizar email y marcar email_verified = false
          │
          └── Reenviar correo de verificación al nuevo email
                    │
                    ▼
              Respuesta HTTP
```

Al cambiar el correo, la cuenta queda como **no verificada** nuevamente (`email_verified = false`), por lo que el usuario debe verificar el nuevo correo, reutilizando el mismo mecanismo de verificación del registro (código numérico con TTL de 15 minutos).

### Respuesta exitosa

```json
{
    "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "username": "natalia",
    "email": "nueva-natalia@gmail.com",
    "created_at": "2026-08-08T..."
}
```

### Posibles errores

| Código HTTP | Code                | Causa                                                                 |
| ----------- | -------------------- | ---------------------------------------------------------------------- |
| 401         | `UNAUTHORIZED`        | Falta el token o es inválido/expirado                                  |
| 400         | `INVALID_REQUEST`     | JSON mal formado                                                       |
| 400         | `VALIDATION_ERROR`    | `new_email` no es un email válido o falta `password`                    |
| 400         | `EMAIL_UPDATE_FAILED` | Contraseña incorrecta, email igual al actual, o email ya registrado    |

---

# 17.1 Cambio de nombre de usuario

Permite a un usuario autenticado actualizar su nombre de usuario (`username`).

### Handler y ruta

```go
protected.PATCH("/username", usersHandler.UpdateUsername)
```

```text
PATCH /api/v1/users/username
```

Requiere autenticación:

```http
Authorization: Bearer <token>
```

### DTO

```go
type UpdateUsernameRequest struct {
    NewUsername string `json:"new_username" validate:"required,min=2,max=50"`
}
```

### Body

```json
{
    "new_username": "natalia_nueva"
}
```

### Flujo (`UserService.UpdateUsername`)

```text
PATCH /api/v1/users/username
          │
          ▼
   AuthMiddleware (extrae userID de la sesión)
          │
          ▼
    UpdateUsernameRequest
          │
          ▼
   UserService.UpdateUsername()
          │
          ├── Buscar usuario por ID
          │
          ├── Rechazar si new_username == username actual
          │
          └── Actualizar username
                    │
                    ▼
              Respuesta HTTP
```

Nota: al igual que en el registro, esta operación no valida unicidad del `username` a nivel de aplicación ni de base de datos (no existe un índice único sobre esa columna).

### Respuesta exitosa

```json
{
    "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "username": "natalia_nueva",
    "email": "natalia@gmail.com",
    "created_at": "2026-08-08T..."
}
```

### Posibles errores

| Código HTTP | Code                     | Causa                                                    |
| ----------- | ------------------------ | --------------------------------------------------------- |
| 401         | `UNAUTHORIZED`           | Falta el token o es inválido/expirado                    |
| 400         | `INVALID_REQUEST`        | JSON mal formado                                          |
| 400         | `VALIDATION_ERROR`       | `new_username` vacío o fuera del rango de 2 a 50 caracteres |
| 400         | `USERNAME_UPDATE_FAILED` | `new_username` igual al actual, o usuario no encontrado  |

---

# 18. Cambio de contraseña

Permite a un usuario autenticado cambiar su contraseña, validando la contraseña actual antes de aplicar la nueva.

### Handler y ruta

```go
protected.PATCH("/password", usersHandler.ChangePassword)
```

```text
PATCH /api/v1/users/password
```

Requiere autenticación:

```http
Authorization: Bearer <token>
```

### DTO

```go
type ChangePasswordRequest struct {
    CurrentPassword    string `json:"current_password" validate:"required"`
    NewPassword        string `json:"new_password" validate:"required,min=8"`
    ConfirmNewPassword string `json:"confirm_new_password" validate:"required,eqfield=NewPassword"`
}
```

### Body

```json
{
    "current_password": "Password123!",
    "new_password": "NuevaPassword456!",
    "confirm_new_password": "NuevaPassword456!"
}
```

### Flujo (`UserService.ChangePassword`)

```text
PATCH /api/v1/users/password
          │
          ▼
   AuthMiddleware (extrae userID de la sesión)
          │
          ▼
    ChangePasswordRequest
          │
          ▼
   UserService.ChangePassword()
          │
          ├── Buscar usuario por ID
          │
          ├── Verificar contraseña actual (BCrypt)
          │
          ├── Validar reglas de complejidad de la nueva contraseña
          │
          ├── Rechazar si la nueva contraseña es igual a la actual
          │
          ├── Hashear la nueva contraseña (BCrypt)
          │
          └── Actualizar password_hash en base de datos
                    │
                    ▼
              Respuesta HTTP
```

La nueva contraseña debe cumplir las mismas reglas de validación usadas en el registro (mínimo 8 caracteres, una mayúscula, un número y un carácter especial).

### Respuesta exitosa

```json
{
    "message": "Password changed successfully"
}
```

### Posibles errores

| Código HTTP | Code                    | Causa                                                                       |
| ----------- | -------------------------- | ------------------------------------------------------------------------------ |
| 401         | `UNAUTHORIZED`             | Falta el token o es inválido/expirado                                          |
| 400         | `INVALID_REQUEST`          | JSON mal formado                                                                |
| 400         | `VALIDATION_ERROR`         | Contraseñas no coinciden o `new_password` tiene menos de 8 caracteres            |
| 400         | `PASSWORD_CHANGE_FAILED`   | Contraseña actual incorrecta, nueva contraseña inválida, o igual a la actual    |

---

# 19. Listado de usuarios

Permite a cualquier usuario autenticado obtener el listado completo de usuarios registrados.

### Handler y ruta

```go
protected.GET("", usersHandler.ListUsers)
```

```text
GET /api/v1/users
```

Requiere autenticación:

```http
Authorization: Bearer <token>
```

### Ejemplo de uso (Postman)

**Request**

```http
GET http://localhost:8080/api/v1/users
Authorization: Bearer d290f1ee-6c54-4b01-90e6-d701748f0851...
```

Este endpoint no recibe body, ya que es una petición `GET`.

### Flujo (`UserService.ListUsers`)

```text
GET /api/v1/users
          │
          ▼
   AuthMiddleware (valida la sesión)
          │
          ▼
   UserService.ListUsers()
          │
          └── SELECT * FROM app_user ORDER BY created_at DESC
                    │
                    ▼
              Respuesta HTTP
```

### Respuesta exitosa

```json
[
    {
        "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
        "username": "natalia",
        "email": "natalia@gmail.com",
        "created_at": "2026-08-08T..."
    },
    {
        "id": "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy",
        "username": "pedro",
        "email": "pedro@gmail.com",
        "created_at": "2026-08-07T..."
    }
]
```

### Posibles errores

| Código HTTP | Code                | Causa                                   |
| ----------- | -------------------- | ----------------------------------------- |
| 401         | `UNAUTHORIZED`        | Falta el token o es inválido/expirado     |
| 500         | `USERS_LIST_FAILED`   | Error al consultar la base de datos       |

---

# 20. Consideraciones de seguridad

* El archivo `.env` no debe subirse al repositorio.
* Las contraseñas nunca deben almacenarse en texto plano; se utiliza BCrypt.
* Los tokens de sesión, verificación y recuperación son aleatorios (`crypto/rand`) y se almacenan hasheados (SHA-256) en Valkey, nunca en texto plano.
* Las sesiones se pueden invalidar en cualquier momento (logout), a diferencia de un JWT stateless que sigue siendo válido hasta su expiración.
* El TTL de sesión se renueva con cada petición autenticada (sliding expiration de 20 días de inactividad).
* Los tokens deben enviarse mediante HTTPS en ambientes de producción.
* Los mensajes de autenticación y recuperación de contraseña evitan revelar si un correo está registrado o no.

---

# 21. Resultado

Con esta implementación, el PATO User Service cuenta con un mecanismo de autenticación basado en sesiones opacas respaldadas por Valkey que permite:

```text
Registro
   ↓
Verificación de correo
   ↓
Login
   ↓
Sesión almacenada en Valkey
   ↓
Autenticación mediante Bearer Token
   ↓
Acceso a endpoints protegidos
   ↓
Logout (invalidación explícita) o expiración por inactividad
```

La sesión tiene actualmente una duración de **20 días de inactividad** (renovable), el código de verificación de correo dura **15 minutos**, y el token de recuperación de contraseña también dura **15 minutos**.
