# Order Processing System

Event-driven microservices system built with Go and AWS SQS.

---

## Architecture

```
Client → order-service (REST API, :8080)
              ↓ PostgreSQL
              ↓ publishes to SQS
SQS Queue → inventory-service  (consumer)
SQS Queue → notification-service (consumer)
```

A single SQS queue (`orders-queue`) fans out to both consumers. Each consumer independently polls, processes, and deletes messages. LocalStack provides a local SQS emulator so no AWS account is needed.

---

## Tech Stack

| | |
|---|---|
| Language | Go 1.26 |
| HTTP Framework | Gin |
| Database | PostgreSQL |
| Message Queue | Amazon SQS (LocalStack) |
| Logging | Uber Zap |
| Containerization | Docker |

---

## How to Run

**Start everything:**

```bash
docker compose up
```

This starts PostgreSQL, LocalStack, order-service, inventory-service, and notification-service. The `orders` table and SQS queue are created automatically on first boot.

**Create an order (bash):**

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 42, "total_amount": 129.99}'
```

**Create an order (PowerShell):**

```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders `
  -ContentType "application/json" `
  -Body '{"customer_id": 42, "total_amount": 129.99}'
```

**Get an order:**

```bash
curl http://localhost:8080/orders/1
```

---

## API Endpoints

### `POST /orders`

Creates a new order with `pending` status, persists it to PostgreSQL, and publishes an `OrderCreated` event to SQS.

**Request body**

```json
{
  "customer_id": 42,
  "total_amount": 129.99
}
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

Retrieves a single order by ID.

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

---

## Project Structure

```
order-processing-system/
├── cmd/
│   ├── order-service/
│   │   └── main.go                   # Wires deps, starts HTTP server
│   ├── inventory-service/
│   │   └── main.go                   # Starts SQS consumer loop
│   └── notification-service/
│       └── main.go                   # Starts SQS consumer loop
├── configs/
│   └── config.go                     # Reads env vars into typed config structs
├── init/
│   ├── init.sql                      # Creates orders table on first boot
│   └── localstack-init.sh            # Creates SQS queue on LocalStack ready
├── internal/
│   ├── handler/
│   │   └── order_handler.go          # HTTP layer — request parsing, response writing
│   ├── inventory/
│   │   └── consumer.go               # SQS consumer — deducts stock
│   ├── model/
│   │   └── order.go                  # Domain types: Order, CreateOrderRequest
│   ├── notification/
│   │   └── consumer.go               # SQS consumer — sends email notification
│   ├── repository/
│   │   └── order_repository.go       # Database access via pgx
│   └── service/
│       └── order_service.go          # Business logic — orchestrates handler and repo
├── pkg/
│   ├── logger/
│   │   └── logger.go                 # Zap logger factory
│   └── queue/
│       └── sqs.go                    # SQS client and OrderCreatedEvent type
├── Dockerfile.order
├── Dockerfile.inventory
├── Dockerfile.notification
├── docker-compose.yml
├── .env
├── go.mod
└── go.sum
```

---

## How It Works

1. **Order created** — `POST /orders` saves the order to PostgreSQL, then publishes an `OrderCreatedEvent` (order ID, customer ID, total amount) to the `orders-queue` SQS queue.

2. **inventory-service** — polls the queue, receives the event, logs the stock deduction, then deletes the message from SQS.

3. **notification-service** — polls the same queue independently, receives the event, logs an email notification, then deletes the message from SQS.

4. **Failure handling** — if either consumer fails to process a message, it skips the delete step. SQS re-delivers the message after the visibility timeout expires, giving the consumer another attempt automatically.

---

## Key Engineering Decisions

**SQS over direct service calls** — Decouples the order-service from downstream consumers. If inventory-service or notification-service is down, the order still succeeds and the event is durably queued. No consumer restart is required to catch up; SQS holds the messages until they are processed.

**Delete only after successful processing** — Messages are only deleted from SQS once the consumer has fully handled them. A failure before the delete causes the message to re-appear after the visibility timeout, giving automatic at-least-once retry without any custom retry logic.

**Structured logging with Zap** — Every event (order received, stock deducted, notification sent, message deleted) is logged as a structured JSON field set rather than a formatted string. This makes logs queryable in any log aggregation system and avoids string-formatting overhead in the hot path.
