# Xelemetry

## Descripción General

Xelemetry es una API que registra requests para saber cuándo hay corriente en la oficina. Cuando llega la corriente, se enciende un servidor local que comienza a hacer requests en forma de polling a esta API para registrar los horarios de corriente, debido a la situación actual de Cuba con los apagones eléctricos. Esto contribuirá a gestionar de mejor manera el tiempo del personal.

## Endpoints del API

---

### POST /location

Crea una nueva ubicación/locación.

- **Método**: `POST`
- **URL**: `/location`
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
- **Respuesta exitosa**:
  - **Código**: `201 Created`
  - **Body**:
    ```json
    {
      "ID": "uuid-string",
      "Nombre": "Oficina Central"
    }
    ```

---

### GET /location

Lista todas las ubicaciones.

- **Método**: `GET`
- **URL**: `/location`
- **Respuesta exitosa**:
  - **Código**: `200 OK`
  - **Body**:
    ```json
    [
      {
        "ID": "uuid-string",
        "Nombre": "Oficina Central"
      }
    ]
    ```

---

### POST /check

Registra un nuevo check (momento en que hay corriente).

- **Método**: `POST`
- **URL**: `/check`
- **Body** (JSON):
  ```json
  {
    "location_id": "uuid-string"
  }
  ```
- **Parámetros**:
  | Campo | Tipo | Requerido | Descripción |
  |-------|------|----------|-------------|
  | `location_id` | string (UUID) | Sí | ID de la ubicación |
- **Respuesta exitosa**:
  - **Código**: `201 Created`
  - **Body**:
    ```json
    {
      "ID": 1,
      "Time": "2024-01-01T12:00:00Z",
      "LocationID": "uuid-string"
    }
    ```

---

### GET /check

Obtiene la lista de checks registrados.

- **Método**: `GET`
- **URL**: `/check`
- **Query Parameters**:
  | Parámetro | Tipo | Requerido | Descripción |
  |-----------|------|----------|-------------|
  | `limit` | int | No | Cantidad máxima de registros. Default: `40`. Rango: `1` a `100` |
  | `from` | string | No | Fecha/hora de inicio para filtrar (formato SQLite) |
  | `to` | string | No | Fecha/hora de fin para filtrar (formato SQLite) |
  | `location_id` | string (UUID) | No | Filtrar por ubicación |
- **Respuesta exitosa**:
  - **Código**: `200 OK`
  - **Body**:
    ```json
    [
      {
        "ID": 1,
        "Time": "2024-01-01T12:00:00Z",
        "LocationID": "uuid-string"
      }
    ]
    ```

---

### GET /uptime

Obtiene la lista de registros de uptime (tiempo que una ubicación estuvo conectada).

- **Método**: `GET`
- **URL**: `/uptime`
- **Query Parameters**:
  | Parámetro | Tipo | Requerido | Descripción |
  |-----------|------|----------|-------------|
  | `limit` | int | No | Cantidad máxima de registros. Default: `40`. Rango: `1` a `100` |
  | `from` | string | No | Fecha/hora de inicio para filtrar (formato SQLite) |
  | `to` | string | No | Fecha/hora de fin para filtrar (formato SQLite) |
  | `location_id` | string (UUID) | No | Filtrar por ubicación |
- **Respuesta exitosa**:
  - **Código**: `200 OK`
  - **Body**:
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

---

### GET /ws

Endpoint WebSocket para rastrear el uptime de una ubicación. Mantiene la conexión abierta mientras hay corriente y registra la duración al desconectarse.

- **Método**: `GET`
- **URL**: `/ws`
- **Query Parameters**:
  | Parámetro | Tipo | Requerido | Descripción |
  |-----------|------|----------|-------------|
  | `location_id` | string (UUID) | Sí | ID de la ubicación a monitorear |
- **Comportamiento**:
  1. Conectar al WebSocket con `location_id`
  2. Mantener la conexión abierta
  3. Al desconectarse, se calcula automáticamente la duración y se guarda en la base de datos
- **Respuesta exitosa**:
  - Conexión WebSocket establecida
  - No retorna respuesta HTTP tradicional

---

## Estructura de la Base de Datos

La aplicación utiliza **SQLite** como base de datos, gestionada a través de **GORM**.

### Tabla: `locations`

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `id` | TEXT (UUID) | Clave primaria |
| `nombre` | TEXT | Nombre de la ubicación (único) |

### Tabla: `checks`

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `id` | INTEGER | Clave primaria, autoincremental |
| `time` | DATETIME | Fecha y hora del registro. Default: `current_timestamp` |
| `location_id` | TEXT (UUID) | FK hacia locations |

### Tabla: `uptimes`

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `id` | INTEGER | Clave primaria, autoincremental |
| `duration` | INTEGER | Duración en segundos |
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
