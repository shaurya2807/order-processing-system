# Scalable Event-Driven Order Processing System in Go

Production-style distributed microservices system with async messaging, caching, object storage, encryption, search, and gRPC.

---

## Architecture

```
Client
  │
  ├─── REST  :8080  ──▶ order-service
  └─── gRPC  :50051 ──▶ order-service
                              │
                 ┌────────────┼────────────────────┐
                 │            │                    │
                 ▼            ▼                    ▼
           PostgreSQL       Redis            Amazon SQS
           (storage)       (cache)           (events)
                                               │
                              ┌────────────────┴────────────────┐
                              ▼                                  ▼
                    inventory-service               notification-service
                    (stock deduction)               (email notification)
                              
           Also async after order creation:
           ├─── AWS S3 + KMS  (encrypted order artifact)
           └─── OpenSearch    (order indexed for search)
```

---

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go 1.26 |
| HTTP | Gin |
| RPC | gRPC |
| Database | PostgreSQL (AWS RDS) |
| Cache | Redis (AWS ElastiCache) |
| Queue | Amazon SQS (LocalStack) |
| Storage | AWS S3 + KMS encryption (LocalStack) |
| Search | OpenSearch |
| Logging | Uber Zap (structured) |
| Containers | Docker Compose |

---

## How to Run

**Start everything:**

```bash
docker compose up
```

This starts PostgreSQL, Redis, LocalStack, OpenSearch, order-service, inventory-service, and notification-service. The `orders` table, SQS queue, and S3 bucket are created automatically on first boot.

**Create an order (PowerShell):**

```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders `
  -ContentType "application/json" `
  -Body '{"customer_id": 42, "total_amount": 129.99}'
```

**Create an order (bash):**

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 42, "total_amount": 129.99}'
```

**Get an order:**

```bash
curl http://localhost:8080/orders/1
```

---

## API Endpoints

### `POST /orders` — REST

Creates a new order with `pending` status, persists it to PostgreSQL, and kicks off async operations: S3 upload, OpenSearch indexing, and SQS event publication.

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

---

### `GET /orders/:id` — REST

Retrieves a single order by ID. Checks Redis first; falls back to PostgreSQL on a cache miss and repopulates the cache.

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

### `CreateOrder` — gRPC :50051

Creates an order via the typed gRPC interface. Equivalent to `POST /orders` but uses protobuf wire format. Proto definition: [`proto/order.proto`](proto/order.proto).

### `GetOrder` — gRPC :50051

Retrieves an order by ID via gRPC with the same Redis-first cache logic as the REST endpoint. Proto definition: [`proto/order.proto`](proto/order.proto).

---

## Project Structure

```
order-processing-system/
├── cmd/
│   ├── order-service/
│   │   └── main.go                   # Wires deps, starts HTTP + gRPC servers
│   ├── inventory-service/
│   │   └── main.go                   # Starts SQS consumer loop
│   └── notification-service/
│       └── main.go                   # Starts SQS consumer loop
├── configs/
│   └── config.go                     # Reads env vars into typed config structs
├── init/
│   ├── init.sql                      # Creates orders table on first boot
│   └── localstack-init.sh            # Creates SQS queue and S3 bucket on LocalStack ready
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
│       └── order_service.go          # Business logic — orchestrates all downstream operations
├── pkg/
│   ├── cache/
│   │   └── redis.go                  # Redis client — get/set/delete with 5min TTL
│   ├── grpc/
│   │   ├── server.go                 # gRPC server implementation
│   │   └── order_grpc.go             # Generated protobuf types and service descriptors
│   ├── logger/
│   │   └── logger.go                 # Zap logger factory
│   ├── queue/
│   │   └── sqs.go                    # SQS client and OrderCreatedEvent type
│   ├── search/
│   │   └── opensearch.go             # OpenSearch client — index and query orders
│   └── storage/
│       └── s3.go                     # S3 client — upload artifacts with KMS encryption
├── proto/
│   └── order.proto                   # Service contract: CreateOrder and GetOrder RPCs
├── Dockerfile.order
├── Dockerfile.inventory
├── Dockerfile.notification
├── docker-compose.yml
├── .env
├── go.mod
└── go.sum
```

---

## Full Flow

### `POST /orders`

1. Order saved to PostgreSQL with `pending` status; an ID is returned.
2. Redis cache invalidated for that order key to prevent stale reads.
3. gRPC clients can call `CreateOrder` on `:50051` — same service, same logic.
4. Order artifact uploaded to AWS S3 with KMS encryption *(async goroutine — does not block response)*.
5. Order document indexed in OpenSearch *(async goroutine — does not block response)*.
6. `OrderCreated` event (order ID, customer ID, total amount) published to Amazon SQS `orders-queue`.
7. inventory-service polls SQS, receives the event, deducts stock, then deletes the message.
8. notification-service polls the same queue independently, sends an email notification, then deletes the message.

### `GET /orders/:id`

1. Check Redis first for the order key (5-minute TTL).
2. Cache hit — return immediately (~200 microseconds, no database query).
3. Cache miss — fetch from PostgreSQL, populate Redis with 5-minute TTL, return.

---

## Key Engineering Decisions

**gRPC for internal service contracts** — Typed proto definitions enforce a schema between callers and the service. `proto/order.proto` is the single source of truth for request/response shapes, with generated code guaranteeing wire compatibility.

**SQS decoupling** — The order-service publishes to a queue and returns immediately. If inventory-service or notification-service is down, the order still succeeds and the event is durably held until the consumer recovers.

**Delete only after successful processing** — Consumers delete a message from SQS only after successfully handling it. A crash before the delete causes SQS to re-deliver after the visibility timeout, giving automatic at-least-once retry with no custom retry logic required.

**Async S3 and OpenSearch** — Uploading artifacts and indexing documents happen in background goroutines, so the HTTP response is never held hostage to the latency of those operations. Order creation stays fast even if S3 or OpenSearch is slow.

**Redis cache-aside** — Reads check the cache before hitting the database. Writes invalidate the cache so the next read always fetches a fresh copy from PostgreSQL. TTL of 5 minutes bounds staleness without requiring explicit invalidation on every update path.

**KMS envelope encryption** — S3 objects are encrypted with a KMS-managed key. The data key never leaves KMS in plaintext, so even direct S3 bucket access yields only ciphertext. This satisfies compliance requirements for sensitive order data at rest.

**Structured Zap logging** — Every significant event (order received, cache hit/miss, stock deducted, notification sent, message deleted) is logged as structured JSON, making logs directly queryable in any log aggregation system (OpenSearch, CloudWatch Logs Insights, etc.) without parsing.
