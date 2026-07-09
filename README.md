# Xelemetry

## Descripción General

Xelemetry es una API que registra requests para saber cuándo hay corriente en la oficina. Cuando llega la corriente, se enciende un servidor local que comienza a hacer requests en forma de polling a esta API para registrar los horarios de corriente, debido a la situación actual de Cuba con los apagones eléctricos. Esto contribuirá a gestionar de mejor manera el tiempo del personal.

## Autenticación

Todos los endpoints marcados con **🔒 Requiere token** deben incluir el header:

```
Authorization: Bearer <token>
```

El token se obtiene mediante `POST /login`.

---

## Endpoints públicos (no requieren autenticación)

### POST /user

Crea un nuevo usuario.

- **Método**: `POST`
- **URL**: `/user`
- **Body** (JSON):
  ```json
  {
    "user_name": "juan",
    "pass_word": "miPassword123"
  }
  ```
- **Parámetros**:
  | Campo | Tipo | Requerido | Descripción |
  |-------|------|----------|-------------|
  | `user_name` | string | Sí | Nombre de usuario |
  | `pass_word` | string | Sí | Contraseña (se hashea con argon2) |
- **Respuesta exitosa**: `201 Created`
  ```json
  {
    "ID": "uuid-string",
    "UserName": "juan",
    "PassWord": "$argon2id$v=19$m=...",
    "Locations": null
  }
  ```

---

### POST /login

Inicia sesión y devuelve un token JWT.

- **Método**: `POST`
- **URL**: `/login`
- **Body** (JSON):
  ```json
  {
    "user_name": "juan",
    "pass_word": "miPassword123"
  }
  ```
- **Parámetros**:
  | Campo | Tipo | Requerido | Descripción |
  |-------|------|----------|-------------|
  | `user_name` | string | Sí | Nombre de usuario |
  | `pass_word` | string | Sí | Contraseña |
- **Respuesta exitosa**: `200 OK`
  ```json
  {
    "token": "eyJhbGciOiJIUzI1NiIs..."
  }
  ```
- **Errores**:
  - `401 Unauthorized`: credenciales inválidas

---

## Endpoints protegidos (requieren token 🔒)

### POST /location 🔒

Crea una nueva ubicación asociada al usuario autenticado.

- **Método**: `POST`
- **URL**: `/location`
- **Header**: `Authorization: Bearer <token>`
- **Body** (JSON):
  ```json
  {
    "nombre": "Oficina Central"
  }
  ```
- **Parámetros**:
  | Campo | Tipo | Requerido | Descripción |
  |-------|------|----------|-------------|
  | `nombre` | string | Sí | Nombre de la ubicación |
- **Respuesta exitosa**: `201 Created`
  ```json
  {
    "ID": "uuid-string",
    "Nombre": "Oficina Central",
    "UserID": "uuid-string",
    "Uptimes": null
  }
  ```

---

### GET /location 🔒

Lista las ubicaciones del usuario autenticado.

- **Método**: `GET`
- **URL**: `/location`
- **Header**: `Authorization: Bearer <token>`
- **Respuesta exitosa**: `200 OK`
  ```json
  [
    {
      "ID": "uuid-string",
      "Nombre": "Oficina Central",
      "UserID": "uuid-string",
      "Uptimes": null
    }
  ]
  ```

---

### GET /uptime 🔒

Obtiene los uptimes de una ubicación específica. Solo accesible si la ubicación pertenece al usuario autenticado.

- **Método**: `GET`
- **URL**: `/uptime`
- **Header**: `Authorization: Bearer <token>`
- **Query Parameters**:
  | Parámetro | Tipo | Requerido | Descripción |
  |-----------|------|----------|-------------|
  | `location_id` | string (UUID) | **Sí** | ID de la ubicación |
  | `limit` | int | No | Cantidad máxima. Default: `40`. Rango: `1`–`100` |
  | `from` | string | No | Filtro desde (formato ISO/SQLite) |
  | `to` | string | No | Filtro hasta (formato ISO/SQLite) |
- **Respuesta exitosa**: `200 OK`
  ```json
  [
    {
      "ID": 1,
      "Duration": 3600,
      "StartTime": "2024-01-01T12:00:00Z",
      "LocationID": "uuid-string"
    }
  ]
  ```
- **Errores**:
  - `404 Not Found`: la ubicación no existe o no pertenece al usuario

---

### GET /ws

Endpoint WebSocket para rastrear el uptime de una ubicación en tiempo real. No requiere autenticación.

- **Método**: `GET`
- **URL**: `/ws`
- **Query Parameters**:
  | Parámetro | Tipo | Requerido | Descripción |
  |-----------|------|----------|-------------|
  | `location_id` | string (UUID) | Sí | ID de la ubicación a monitorear |
- **Comportamiento**:
  1. Se conecta al WebSocket con `location_id`
  2. Se crea un registro `uptime` con `duration = null`
  3. Se mantiene la conexión abierta mientras hay corriente
  4. Al desconectarse, se actualiza ese registro con la duración real en segundos
- **Respuesta**: Conexión WebSocket establecida (no retorna HTTP)

---

## Estructura de la Base de Datos

La aplicación utiliza **SQLite** como base de datos, gestionada a través de **GORM**.

### Tabla: `users`

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `id` | TEXT (UUID) | Clave primaria |
| `user_name` | TEXT | Nombre de usuario |
| `pass_word` | TEXT | Contraseña del usuario |

### Tabla: `locations`

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `id` | TEXT (UUID) | Clave primaria |
| `nombre` | TEXT | Nombre de la ubicación (único) |
| `user_id` | TEXT (UUID) | FK hacia users |

### Tabla: `uptimes`

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `id` | INTEGER | Clave primaria, autoincremental |
| `duration` | INTEGER | Duración en segundos (nulo mientras la conexión está activa) |
| `start_time` | DATETIME | Hora de inicio. Default: `current_timestamp` |
| `location_id` | TEXT (UUID) | FK hacia locations |

---

## Despliegue

### Requisitos

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)

### Variables de Entorno

| Variable | Descripción | Ejemplo |
|----------|-------------|---------|
| `PORT` | Puerto en el que correrá la aplicación | `8080` |
| `JWT_SECRET` | Secreto para firmar tokens JWT | `mi-secreto-super-seguro` |

> ⚠️ `JWT_SECRET` es obligatorio. Si no se configura, la API falla al iniciar.

### Pasos para el Despliegue

1. Clonar el repositorio:

   ```bash
   git clone <url-del-repositorio>
   cd xelemetry
   ```

2. Crear un archivo `.env` con la variable de entorno necesaria:

   ```bash
   echo "PORT=8080" > .env
   ```

3. Construir y levantar los contenedores con Docker Compose:

   ```bash
   docker compose up -d --build
   ```

4. Ejecutar la migración de la base de datos:

   ```bash
   docker compose exec api ./migration
   ```

5. La API estará disponible en `http://localhost:<PORT>`.

### Despliegue Local (sin Docker)

1. Asegúrate de tener [Go](https://go.dev/dl/) instalado.
2. Exporta la variable de entorno:

   ```bash
   export PORT=8080
   ```

3. Ejecuta la aplicación:

   ```bash
   go run cmd/migration/main.go
   go run cmd/api/main.go
   ```
