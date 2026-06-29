# Xelemetry

## Descripción General

Xelemetry es una API que registra requests para saber cuándo hay corriente en la oficina. Cuando llega la corriente, se enciende un servidor local que comienza a hacer requests en forma de polling a esta API para registrar los horarios de corriente, debido a la situación actual de Cuba con los apagones eléctricos. Esto constribuirá a gestionar de mejor manera el tiempo del personal.

## Endpoints del API

### POST /check

Registra un nuevo check (momento en que hay corriente).

- **Método**: `POST`
- **URL**: `/check`
- **Body**: No requiere body (se crea automáticamente con la fecha/hora actual)
- **Respuesta exitosa**:
  - **Código**: `200 OK`
  - **Body**: Vacío (el registro se crea en la base de datos)

### GET /check

Obtiene la lista de checks registrados.

- **Método**: `GET`
- **URL**: `/check`
- **Query Parameters**:
  - `limit` (opcional): Cantidad máximo de registros a devolver. Valor por defecto: `40`. Rango: `1` a `100`.
  - `from` (opcional): Fecha/hora de inicio para filtrar registros (formato compatible con SQLite).
  - `to` (opcional): Fecha/hora de fin para filtrar registros (formato compatible con SQLite).
- **Respuesta exitosa**:
  - **Código**: `200 OK`
  - **Body**: Array de objetos `Check`
    ```json
    [
      {
        "ID": 1,
        "Time": "2024-01-01 12:00:00"
      }
    ]
    ```

## Estructura de la Base de Datos

La aplicación utiliza **SQLite** como base de datos, gestionada a través de **GORM**.

### Tabla: `checks`

| Campo | Tipo      | Descripción                              |
|-------|-----------|------------------------------------------|
| `id`  | `INTEGER` | Clave primaria, autoincremental          |
| `time`| `DATETIME`| Fecha y hora del registro. Por defecto: `current_timestamp` |

### Modelo (Go)

```go
type Check struct {
    ID   int
    Time *time.Time `gorm:"default:current_timestamp"`
}
```

## Despliegue

### Requisitos

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)

### Variables de Entorno

| Variable | Descripción                              | Ejemplo |
|----------|------------------------------------------|---------|
| `PORT`   | Puerto en el que correrá la aplicación   | `8080`  |

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
