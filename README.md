# order-processing-system

A production-ready order management microservice built in Go, following clean layered architecture principles.

---

## Architecture

The service is structured around a strict separation of concerns across four layers:

```
HTTP Request
     │
     ▼
┌─────────────┐
│   Handler   │  Parses and validates HTTP input, writes HTTP responses
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Service   │  Owns business logic and orchestrates domain operations
└──────┬──────┘
       │
       ▼
┌──────────────────┐
│   Repository     │  Executes SQL queries against PostgreSQL
└──────────────────┘
```

Each layer depends only on the layer below it. Handlers never touch the database; repositories never know about HTTP. This keeps the service easy to test, extend, and reason about.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.22+ |
| HTTP Framework | [Gin](https://github.com/gin-gonic/gin) |
| Database | PostgreSQL (via [pgx v5](https://github.com/jackc/pgx)) |
| Message Queue | Amazon SQS *(coming soon)* |
| Logging | [Uber Zap](https://github.com/uber-go/zap) |
| Containerization | Docker *(coming soon)* |

---

## Prerequisites

- **Go** 1.22 or higher
- **PostgreSQL** 14 or higher running locally
- A `orders` database created in PostgreSQL

```sql
CREATE DATABASE orders;
```

- The following table created in that database:

```sql
CREATE TABLE orders (
    id           BIGSERIAL PRIMARY KEY,
    customer_id  BIGINT        NOT NULL,
    status       VARCHAR(20)   NOT NULL DEFAULT 'pending',
    total_amount NUMERIC(10,2) NOT NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
```

---

## Running Locally

**1. Clone the repository**

```bash
git clone https://github.com/shaurya2807/order-processing-system.git
cd order-processing-system
```

**2. Configure environment variables**

Copy the example env file and adjust values to match your local PostgreSQL setup:

```bash
cp .env.example .env
```

```env
APP_ENV=development
SERVER_PORT=8080

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=orders
DB_SSLMODE=disable
```

**3. Install dependencies**

```bash
go mod download
```

**4. Run the service**

```bash
go run ./cmd/order-service
```

The server starts on `http://localhost:8080`.

---

## API Endpoints

### `POST /orders`

Creates a new order. New orders are initialized with `pending` status.

**Request**

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": 42,
    "total_amount": 129.99
  }'
```

**Response** `201 Created`

```json
{
  "id": 1,
  "customer_id": 42,
  "status": "pending",
  "total_amount": 129.99,
  "created_at": "2026-04-23T10:15:30Z"
}
```

**Validation error** `400 Bad Request`

```json
{
  "error": "Key: 'CreateOrderRequest.TotalAmount' Error:Field validation for 'TotalAmount' failed on the 'gt' tag"
}
```

---

### `GET /orders/:id`

Retrieves a single order by its ID.

**Request**

```bash
curl http://localhost:8080/orders/1
```

**Response** `200 OK`

```json
{
  "id": 1,
  "customer_id": 42,
  "status": "pending",
  "total_amount": 129.99,
  "created_at": "2026-04-23T10:15:30Z"
}
```

**Not found** `404 Not Found`

```json
{
  "error": "order not found"
}
```

**Order statuses**: `pending` → `processing` → `completed` | `cancelled`

---

## Project Structure

```
order-processing-system/
├── cmd/
│   └── order-service/
│       └── main.go              # Entry point — wires dependencies, starts HTTP server
├── configs/
│   └── config.go                # Reads environment variables into typed config structs
├── internal/
│   ├── handler/
│   │   └── order_handler.go     # HTTP layer — request parsing, response writing
│   ├── model/
│   │   └── order.go             # Domain types: Order, CreateOrderRequest, OrderStatus
│   ├── repository/
│   │   └── order_repository.go  # Database access — raw SQL via pgx
│   └── service/
│       └── order_service.go     # Business logic — sits between handler and repository
├── pkg/
│   └── logger/
│       └── logger.go            # Zap logger factory (dev vs. production config)
├── .env                         # Local environment config (git-ignored)
├── go.mod
└── go.sum
```

---

## Coming Soon

- **SQS event publishing** — order lifecycle events (`order.created`, `order.updated`) published to Amazon SQS after each state change
- **inventory-service consumer** — reserves stock when an order is created; releases on cancellation
- **notification-service consumer** — sends customer-facing emails and push notifications on order status transitions
- **Datadog observability** — distributed tracing, APM metrics, and structured log correlation across all consumers
