# Implementación de autenticación con JWT — PATO User Service

## 1. Descripción

Se implementó un sistema de autenticación basado en **JSON Web Tokens (JWT)** para el microservicio de usuarios de PATO.

El flujo permite:

* Registrar usuarios.
* Validar los datos de registro.
* Almacenar contraseñas utilizando hashing con BCrypt.
* Verificar el correo electrónico mediante un token temporal.
* Iniciar sesión utilizando correo y contraseña.
* Generar un JWT después de un login exitoso.
* Configurar la duración del JWT mediante la configuración del servicio.
* Utilizar el JWT posteriormente para proteger endpoints.

La arquitectura mantiene la separación de responsabilidades entre:

```text
Handler → Service → Model/Database
              ↓
             JWT
```

---

# 2. Arquitectura involucrada

La implementación utiliza las siguientes capas:

```text
internal/
├── config/
│   └── config.go
│
├── database/
│   └── database.go
│
├── dto/
│   └── user.go
│
├── handlers/
│   └── users.go
│
├── models/
│   ├── user.go
│   └── email_token.go
│
├── services/
│   ├── user.go
│   └── email.go
│
└── utils/
    ├── security.go
    └── jwt.go
```

### Responsabilidad de cada componente

| Componente       | Responsabilidad                                     |
| ---------------- | --------------------------------------------------- |
| `config`         | Cargar variables de entorno y configuración del JWT |
| `database`       | Conectar con PostgreSQL y ejecutar migraciones      |
| `dto`            | Definir los datos recibidos y enviados por la API   |
| `models`         | Representar las tablas de la base de datos          |
| `handlers`       | Recibir peticiones HTTP y devolver respuestas       |
| `services`       | Contener la lógica de negocio                       |
| `utils/security` | Hash y validación de contraseñas                    |
| `utils/jwt`      | Generación y validación de JWT                      |
| `services/email` | Comunicación con el servicio de correos             |

---

# 3. Configuración mediante `.env`

Se agregó la configuración del JWT al sistema existente de variables de entorno.

Ejemplo:

```env
# Database
APP_DB_IP=tu_ip
APP_DB_PORT=tu_puerto
APP_DB_USER=tu_usuario_db
APP_DB_PASSWORD=tu_contraseña_muy_segura
APP_DB_NAME=nombre_de_tu_db

# JWT
JWT_SECRET=secret-generado-de-forma-segura
```

El archivo `.env` **no debe subirse al repositorio**, por lo que debe estar incluido en `.gitignore`:

```gitignore
.env
```

---

# 4. Configuración del JWT

En `internal/config/config.go` se agregó la estructura:

```go
type JWTConfig struct {
    Secret     string
    Expiration time.Duration
}
```

Esta estructura se incorpora a la configuración principal:

```go
type Config struct {
    Database     DatabaseConfig
    Server       ServerConfig
    EmailService EmailServiceConfig
    JWT          JWTConfig
}
```

Durante la carga de configuración:

```go
JWT: JWTConfig{
    Secret:     getEnv("JWT_SECRET", ""),
    Expiration: 30 * 24 * time.Hour,
},
```

Actualmente, el JWT tiene una duración de:

```text
30 días
```

La duración se calcula como:

```text
30 × 24 horas = 720 horas
```

Por lo tanto, cada vez que un usuario inicia sesión se genera un nuevo JWT cuya fecha de expiración se establece 30 días después.

---

# 5. Seguridad de contraseñas

Las contraseñas no se almacenan directamente en la base de datos.

Se utiliza **BCrypt** mediante:

```go
bcrypt.GenerateFromPassword()
```

La función:

```go
func HashPassword(password string) (string, error)
```

recibe la contraseña original y devuelve su hash.

Para comprobar una contraseña:

```go
func VerifyPassword(hashedPassword, password string) bool
```

utiliza:

```go
bcrypt.CompareHashAndPassword()
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

Durante el login:

```text
Contraseña ingresada
        ↓
CompareHashAndPassword()
        ↓
¿Coincide?
   ├── Sí → continuar
   └── No → rechazar login
```

---

# 6. Validación de contraseñas

Antes de crear un usuario se valida que la contraseña cumpla los requisitos establecidos.

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

# 7. DTO para Login

Se agregó un DTO específico para recibir las credenciales:

```go
type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}
```

También se agregó su método de validación:

```go
func (r *LoginRequest) Validate() error {
    return validate.Struct(r)
}
```

El objetivo del DTO es separar los datos recibidos mediante HTTP de los modelos utilizados para persistencia.

---

# 8. Flujo de Login

El proceso de autenticación sigue este flujo:

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
          ├── Verificar email
          │
          ├── Verificar contraseña
          │
          └── Generar JWT
                    │
                    ▼
              Respuesta HTTP
                    │
                    ▼
                 Cliente
```

---

# 9. Handler de Login

El handler es responsable de:

1. Recibir la petición.
2. Convertir el JSON a `LoginRequest`.
3. Validar los datos.
4. Invocar al `UserService`.
5. Devolver el JWT y los datos del usuario.

La ruta se registra en `main.go`:

```go
users.POST("/login", usersHandler.Login)
```

Por lo tanto, el endpoint completo es:

```text
POST /api/v1/users/login
```

---

# 10. Endpoint de Login

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
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
        "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
        "username": "natalia",
        "email": "natalia@gmail.com",
        "created_at": "2026-08-08T..."
    }
}
```

El campo:

```json
"token"
```

contiene el JWT generado para el usuario.

---

# 11. Verificación del correo antes del Login

El sistema requiere que el usuario haya verificado su correo antes de poder autenticarse.

Durante el registro se genera un token:

```go
const emailVerificationTokenTTL = 15 * time.Minute
```

Por lo tanto:

```text
Token de verificación → 15 minutos
JWT de sesión          → 30 días
```

El flujo completo es:

```text
Registro
   ↓
Crear usuario
   ↓
Generar token de verificación
   ↓
Enviar correo
   ↓
Usuario verifica email
   ↓
email_verified = true
   ↓
Login permitido
```

---

# 12. Diferencia entre los tokens

El sistema utiliza dos tipos de tokens con objetivos diferentes.

| Token                    |   Duración | Uso              |
| ------------------------ | ---------: | ---------------- |
| Email Verification Token | 15 minutos | Verificar correo |
| JWT                      |    30 días | Autenticación    |

El token de verificación **no es un JWT**. Es un token aleatorio que se genera mediante:

```go
crypto/rand
```

y cuyo hash SHA-256 se almacena en la base de datos.

El JWT, en cambio, se utiliza posteriormente para autenticar las peticiones.

---


# 13. Generación del JWT

El JWT utiliza un secreto configurado mediante:

```env
JWT_SECRET=...
```

El secreto es utilizado para firmar el token.

La información relevante del JWT incluye:

```text
sub → identificador del usuario
username → nombre de usuario
email → correo
iat → fecha de emisión
exp → fecha de expiración
```

Conceptualmente:

```text
Usuario
   ↓
Login exitoso
   ↓
GenerateJWT()
   ↓
JWT firmado
   ↓
Cliente
```

El cliente posteriormente envía el token mediante:

```http
Authorization: Bearer <JWT>
```

---

# 14. Duración del JWT

Actualmente:

```go
Expiration: 30 * 24 * time.Hour
```

Esto significa que si un usuario inicia sesión el:

```text
8 de agosto a las 10:00
```

su JWT expirará aproximadamente el:

```text
7 de septiembre a las 10:00
```

Si el usuario vuelve a iniciar sesión después, recibe un nuevo JWT con una nueva fecha de expiración.

La expiración se encuentra dentro del propio JWT mediante el claim:

```text
exp
```

---

# 15. Middleware de autenticación

Para proteger endpoints se utiliza un middleware que:

1. Obtiene el header `Authorization`.
2. Comprueba que utilice el esquema `Bearer`.
3. Extrae el JWT.
4. Valida su firma.
5. Comprueba su expiración.
6. Obtiene la información del usuario.
7. Permite continuar si el token es válido.

Flujo:

```text
Request
  │
  ▼
Authorization: Bearer JWT
  │
  ▼
AuthMiddleware
  │
  ├── ¿Existe token?
  │       └── No → 401
  │
  ├── ¿Firma válida?
  │       └── No → 401
  │
  ├── ¿Token expirado?
  │       └── Sí → 401
  │
  └── Sí
       ↓
    Endpoint
```

---

# 16. Pruebas realizadas con Postman

## 16.1 Health Check

### Request

```http
GET http://localhost:8080/health
```

### Respuesta esperada

```json
{
    "status": "healthy",
    "app": "PATO User Service"
}
```

---

## 16.2 Registro

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

# 17. Verificación de correo

### Request

```http
POST http://localhost:8080/api/v1/users/verify-email
```

### Body

```json
{
    "token": "TOKEN_DE_VERIFICACION"
}
```

### Respuesta

```json
{
    "message": "Email verified successfully"
}
```

---

# 18. Login

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
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
        "id": "...",
        "username": "natalia",
        "email": "natalia@gmail.com",
        "created_at": "..."
    }
}
```

---

# 19. Uso del JWT en Postman

Para probar un endpoint protegido:

1. Crear una petición.
2. Ir a **Authorization**.
3. Seleccionar **Bearer Token**.
4. Pegar el JWT obtenido durante el login.
5. Ejecutar la petición.

Postman enviará:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

Si el token es válido, el middleware permitirá acceder al endpoint.

---

# 20. Prueba de acceso sin JWT

Para comprobar que el middleware realmente protege el endpoint:

```text
GET /api/v1/users/me
```

sin enviar el header `Authorization`.

El servidor debe responder:

```text
401 Unauthorized
```

Esto confirma que el endpoint no puede ser utilizado sin autenticación.

---

# 21. Prueba de acceso con JWT

Al enviar:

```http
Authorization: Bearer <JWT>
```

el servidor debe validar correctamente el token y permitir el acceso:

```text
200 OK
```

Por lo tanto:

```text
Sin JWT → 401 Unauthorized
Con JWT válido → 200 OK
Con JWT inválido → 401 Unauthorized
Con JWT expirado → 401 Unauthorized
```

---


# 22. Flujo completo del sistema

El flujo final de autenticación es:

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
                       GenerateJWT
                           │
                           ▼
                         JWT
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
                   Válido      Inválido
                     │           │
                     ▼           ▼
                  Endpoint      401
```

---

# 23. Resumen de endpoints

| Método | Endpoint                     | Descripción                   | Autenticación |
| ------ | ---------------------------- | ----------------------------- | ------------- |
| `GET`  | `/health`                    | Comprobar estado del servicio | No            |
| `POST` | `/api/v1/users/register`     | Registrar usuario             | No            |
| `POST` | `/api/v1/users/verify-email` | Verificar correo              | No            |
| `POST` | `/api/v1/users/login`        | Iniciar sesión y obtener JWT  | No            |
| `GET`  | `/api/v1/users/me`           | Ejemplo de endpoint protegido | JWT           |

---

# 24. Consideraciones de seguridad

* El `JWT_SECRET` debe mantenerse fuera del código fuente.
* El archivo `.env` no debe subirse al repositorio.
* Las contraseñas nunca deben almacenarse en texto plano.
* Se utiliza BCrypt para almacenar contraseñas.
* Los tokens de verificación no se almacenan directamente; se almacena su hash.
* Los JWT deben enviarse mediante HTTPS en ambientes de producción.
* Los endpoints protegidos deben validar la firma y expiración del JWT.
* Los mensajes de autenticación deben evitar revelar información sensible sobre usuarios existentes.

---

# 25. Resultado

Con esta implementación, el PATO User Service cuenta con un mecanismo básico de autenticación basado en JWT que permite:

```text
Registro
   ↓
Verificación de correo
   ↓
Login
   ↓
Generación de JWT
   ↓
Autenticación mediante Bearer Token
   ↓
Acceso a endpoints protegidos
```

El JWT tiene actualmente una duración de **30 días**, mientras que el token de verificación de correo tiene una duración de **15 minutos**.
