# Scalable Event-Driven Order Processing System in Go

Production-style distributed microservices system with async messaging, caching, object storage, encryption, and search.

---

## Architecture

```
Client
  │
  ▼
order-service (REST API :8080)
  │
  ├─── PostgreSQL (persistent order storage)
  ├─── Redis/ElastiCache (order lookup cache, 5min TTL)
  ├─── Amazon SQS (async event publishing)
  │         │
  │         ├──▶ inventory-service (stock deduction)
  │         └──▶ notification-service (email notification)
  ├─── AWS S3 + KMS (order artifact storage, encrypted)
  └─── OpenSearch (order indexing for search and analytics)
```

---

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go 1.26 |
| HTTP | Gin |
| Database | PostgreSQL (AWS RDS equivalent) |
| Cache | Redis (AWS ElastiCache equivalent) |
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

**Validation error** `400 Bad Request`

```json
{
  "error": "Key: 'CreateOrderRequest.TotalAmount' Error:Field validation for 'TotalAmount' failed on the 'gt' tag"
}
```

---

### `GET /orders/:id`

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
│   ├── logger/
│   │   └── logger.go                 # Zap logger factory
│   ├── queue/
│   │   └── sqs.go                    # SQS client and OrderCreatedEvent type
│   ├── search/
│   │   └── opensearch.go             # OpenSearch client — index and query orders
│   └── storage/
│       └── s3.go                     # S3 client — upload artifacts with KMS encryption
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

### `POST /orders`

1. **PostgreSQL** — order is saved with `pending` status and an ID is returned.
2. **Redis** — cache entry for that order ID is invalidated to prevent stale reads.
3. **S3 + KMS** *(async goroutine)* — order artifact is uploaded to S3 with KMS envelope encryption; does not block the HTTP response.
4. **OpenSearch** *(async goroutine)* — order document is indexed for search and analytics; does not block the HTTP response.
5. **SQS** — `OrderCreated` event (order ID, customer ID, total amount) is published to `orders-queue`.
6. **inventory-service** — polls the queue, receives the event, deducts stock, then deletes the message from SQS.
7. **notification-service** — polls the same queue independently, receives the event, sends an email notification, then deletes the message from SQS.

### `GET /orders/:id`

1. **Redis** — cache is checked first for the order.
2. **Cache hit** — order is returned immediately (sub-millisecond latency), no database query.
3. **Cache miss** — order is fetched from PostgreSQL, stored in Redis with a 5-minute TTL, then returned.

---

## Key Engineering Decisions

**SQS decoupling** — The order-service publishes to a queue and returns immediately. Downstream services are fully independent: if inventory-service or notification-service is down, the order still succeeds and the event is durably held until the consumer recovers.

**Delete only after successful processing** — Consumers delete a message from SQS only after successfully handling it. A crash or error before the delete causes SQS to re-deliver the message after the visibility timeout, giving automatic at-least-once retry with no custom retry logic required.

**Async S3 and OpenSearch** — Uploading artifacts and indexing documents happen in background goroutines, so the HTTP response is not held hostage to the latency of those operations. Order creation stays fast even if S3 or OpenSearch is slow.

**Redis cache-aside pattern** — Reads check the cache before hitting the database. Writes invalidate the cache so the next read always fetches a fresh copy from PostgreSQL. TTL is set to 5 minutes to bound staleness without requiring explicit invalidation for every update path.

**KMS envelope encryption** — S3 objects are encrypted with a KMS-managed key. The data key never leaves KMS in plaintext, so even direct S3 bucket access yields only ciphertext. This satisfies compliance requirements for sensitive order data at rest.

**Structured logging with Zap** — Every significant event (order received, cache hit/miss, stock deducted, notification sent, message deleted) is logged as structured JSON. This makes logs directly queryable in any log aggregation system (e.g., OpenSearch, CloudWatch Logs Insights) without parsing.
