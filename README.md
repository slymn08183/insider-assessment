# Insider Assessment — Event Ingestion Service

High-throughput event ingestion service built with Go. Accepts events via HTTP, queues them in Redis, batch-writes to PostgreSQL, and serves aggregated metrics.

## How to Run

```bash
# Start PostgreSQL and Redis
docker-compose up -d

# Run the app
go run ./cmd/server/
```

## Architecture

```
Client --> [Gin API] --> [Redis Queue + Dedup] --> [Worker] --> [PostgreSQL]
```

Events hit the API, get deduplicated via Redis (`SET with NX` with 24h TTL), and are pushed to a Redis list. A background worker pulls events in batches and bulk-inserts into PostgreSQL. The API returns `202 Accepted` without waiting for the DB.

## API

### POST /events

```bash
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_name": "product_view",
    "channel": "web",
    "campaign_id": "cmp_987",
    "user_id": "user_123",
    "timestamp": 1723475612,
    "tags": ["electronics", "homepage"],
    "metadata": {"product_id": "prod-789", "price": 129.99, "currency": "TRY"}
  }'
```
```json
{"status": "accepted", "event_hash": "a1b2c3..."}
```

### POST /events/bulk

```bash
curl -X POST http://localhost:8080/events/bulk \
  -H "Content-Type: application/json" \
  -d '[
    {"event_name": "product_view", "user_id": "user_1", "timestamp": 1723475612},
    {"event_name": "click", "user_id": "user_2", "timestamp": 1723475613}
  ]'
```
```json
{"accepted": 2, "rejected": 0, "duplicates": 0}
```

### GET /metrics

```bash
curl "http://localhost:8080/metrics?event_name=product_view&group_by=daily&from=2026-01-02T15:04:05Z&to=2056-01-02T15:04:05Z"
```
```json
{
  "total_count": 99996,
  "unique_users": 20000,
  "breakdown": [
    {"period": "2026-03-31T03:00:00+03:00", "count": 51723, "unique_users": 19500},
    {"period": "2026-04-01T03:00:00+03:00", "count": 48273, "unique_users": 19244}
  ]
}
```

Query params: `event_name` (required), `from`/`to` (2026-01-02T15:04:05Z), `channel`, `group_by` (`daily`/`hourly`).

### GET /health

```bash
curl http://localhost:8080/health
```

### Swagger UI

API docs are available at [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html) once the server is running.

## Design Decisions

**Why PostgreSQL?** It handles the required 2K-20K events/sec with batch inserts comfortably. I didn't use ClickHouse because PostgreSQL already meets the min and max throughput requirements, and I didn't want to spend implementation time learning a new tool when the one I know well does the job.

**Why Redis?** Two reasons. First, I needed fast duplicate checking. Redis `SET NX` checks if a hash exists and sets it in a single call, which is perfect for dedup. Second, since Redis was already there for dedup, I used its list data structure as an event queue instead of adding another dependency like RabbitMQ (which I'm more familiar with and would have used otherwise).

**Batch worker.** Without batching, every event would be a separate DB insert, which would choke PostgreSQL under load. The worker collects events from the Redis queue and writes them in batches of 500 (or every 2 seconds if the batch isn't full yet). This turns 500 DB writes into 1.

**Bulk endpoint pipeline.** The first version of `/events/bulk` processed events one by one, each making its own Redis calls. With 100 events per request, that meant 200 Redis round-trips.

```
Before pipeline (100 events/request):
  Avg: 1.06s    P99: 1.21s    Events/sec: 4,609

After pipeline (same 100 events, 2 round-trips total):
  Avg: 87ms     P99: 278ms    Events/sec: 53,990
```

Redis pipeline batches all commands into a single network call. Instead of 200 round-trips, we do 2: one for all dedup checks, one for all enqueues.

**Graceful shutdown.** Not in the requirements but easy to add and essential in production. On Ctrl+C or deploy: stop accepting requests, wait for in-flight ones, flush the worker's remaining batch to DB, then close connections. No events lost during normal shutdowns.

**Timestamp validation** (min 2020, max +1h) is hardcoded for consistency. In production these would be configurable per environment.

**Dedup TTL is 24 hours.** After that, the Redis key expires and the hash is gone. If the same event shows up again, Redis won't catch it, but the DB's unique constraint will. This keeps Redis memory from growing forever.

**Worker crash resilience.** Right now, the worker pulls events from Redis one by one into an in-memory buffer and flushes when it hits 500. If the process gets killed hard (kill -9, OOM), whatever's in that buffer is gone - up to 499 events. Events still sitting in the Redis queue survive, but the ones already pulled out don't.

The fix I'd use in production is an acknowledgment pattern with `LMOVE` Instead of popping events and hoping for the best, you move them from the main queue to a "processing" list. Write to DB, then delete from the processing list. If the worker crashes mid-write, those events are still in the processing list on restart, you check it first and retry. The event is always somewhere: either in the queue, in the processing list, or in the database.

I didn't implement this because 48 hours is tight and the current approach handles normal shutdowns cleanly (graceful shutdown flushes the buffer). But this is the first thing I'd add before going to production.

## Load Test Results

Tested on Windows 11 with Docker Desktop (PostgreSQL + Redis in containers). Real Linux would do better.

**Single event endpoint:**
```
go run test/loadtest.go -n 20000
```
```
Total:      20,000 events
RPS:        2,583 req/sec
Avg:        19ms
P50:        17ms
P99:        53ms
```

**Bulk endpoint (100 events per request):**
```
go run test/loadtest.go -n 20000 -bulk
```
```
Total:      20,000 events (200 requests)
Events/sec: 53,990
Avg:        87ms
P50:        63ms
P99:        278ms
```

**Load test parameters:**
```bash
go run test/loadtest.go [flags]

  -n       Total number of events (default: 20000)
  -c       Concurrent workers (default: 50)
  -bulk    Use bulk endpoint instead of single
  -bs      Events per bulk request (default: 100)
  -url     Base URL (default: http://127.0.0.1:8080)
```

## Configuration

All via environment variables. Defaults work for local dev with Docker Compose.

| Variable           | Default     | Description                                        |
| ------------------ | ----------- | -------------------------------------------------- |
| `SERVER_PORT`      | `8080`      | HTTP port                                          |
| `DB_HOST`          | `localhost` | PostgreSQL host                                    |
| `DB_PORT`          | `5432`      | PostgreSQL port                                    |
| `DB_USER`          | `postgres`  | PostgreSQL user                                    |
| `DB_PASSWORD`      | `postgres`  | PostgreSQL password                                |
| `DB_NAME`          | `insider`   | PostgreSQL database                                |
| `REDIS_HOST`       | `localhost` | Redis host                                         |
| `REDIS_PORT`       | `6379`      | Redis port                                         |
| `BATCH_SIZE`       | `500`       | Events per batch flush                             |
| `FLUSH_INTERVAL`   | `2s`        | Max wait before flush                              |
| `AUTO_MIGRATE`     | `true`      | Run DB migrations on startup (disable in prod)     |

## What I'd Do Next

- Rate limiting per client/IP
- ClickHouse for analytics reads at higher scale
- Multiple worker goroutines for parallel consumption
- Prometheus metrics and Grafana dashboards
- CI/CD pipeline