# PATO User Service — Backend

Servicio de usuarios de PATO.

```
.
├── cmd/
│   └── api/
│       └── main.go             # main xdxd 
├── internal/
│   ├── config/
│   │   └── config.go           # Variables de entorno
│   ├── database/
│   │   └── database.go         # Conexión a La BD
│   ├── dto/
│   │   └── user.go             # Request/response y validación
│   ├── handlers/
│   │   └── users.go            # POST  para regstrar usuarios
│   ├── models/
│   │   └── user.go             # Modelo de un usuario(tabla app_user)
│   ├── services/
│   │   └── user_service.go     # Lógica de negocio
│   └── utils/
│       └── security.go         # Hash bcrypt, validación de contraseña
├── go.mod
├── go.sum
├── .env / .env.example
└── README.md
```



**Reglas de contraseña:** mínimo 8 caracteres, 1 mayúscula, 1 número, 1 carácter especial. Debe coincidir con `confirm_password`. Se guarda cifrada con bcrypt.
