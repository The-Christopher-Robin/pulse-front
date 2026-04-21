# PulseFront

[![ci](https://github.com/The-Christopher-Robin/pulse-front/actions/workflows/ci.yml/badge.svg)](https://github.com/The-Christopher-Robin/pulse-front/actions/workflows/ci.yml)

A full-stack customer-facing storefront with a Go backend and a Next.js SSR front-end,
plus a typed A/B experiment framework (traffic splitting, sticky bucketing, exposure
logging, conversion analytics).

## Stack

- **Backend**: Go 1.25, [chi](https://github.com/go-chi/chi) for HTTP, gRPC for telemetry,
  pgx for Postgres, go-redis for Redis. Goroutines batch exposures and events into Postgres
  via `COPY FROM`.
- **Frontend**: Next.js 14 (Pages Router) + React 18 + TypeScript, server-side rendering via
  `getServerSideProps`. Cookies carry a stable user id so assignments stick across refreshes.
- **Storage**: Postgres for the product catalog, experiment config, exposures, and events.
  Redis caches per-user assignment snapshots.
- **Experiment framework**: typed Go package (`internal/experiments`) with deterministic
  SHA-256 bucketing and independent domains for traffic vs variant decisions.
- **Deployment**: Docker for local dev; ECS + ALB + RDS + ElastiCache for cloud, with
  CloudFront in front of the Next.js origin. See [deploy/aws-notes.md](deploy/aws-notes.md).

## Quick start

Requires Docker and Docker Compose.

```bash
docker compose up --build
```

- Front-end: http://localhost:3000
- Backend REST: http://localhost:8080/api/v1/products
- gRPC telemetry: localhost:9090
- Health check: http://localhost:8080/healthz

The backend runs migrations and seeds ~8 products and 20+ experiments on first boot,
so the storefront has something to render immediately.

## Running without Docker

```bash
# Postgres and Redis are expected on the default dev ports.
cd backend && go run ./cmd/server
cd frontend && npm install && npm run dev
```

Override connection strings via env (`POSTGRES_URL`, `REDIS_ADDR`, `HTTP_ADDR`, `GRPC_ADDR`,
`ALLOWED_ORIGIN`).

## Tests

```bash
make be-test   # go test ./...
make fe-test   # jest
```

Backend tests cover the bucketing function (determinism, traffic-slice accuracy,
weighted variant distribution, salt independence) and the service-level holdout
contract. Frontend tests cover the typed assignment helpers.

## Architecture

```
 Browser
   |  HTML/CSS/JS (SSR)
   v
 Next.js (SSR)  --->  api/proxy/*   --->  Go backend (chi)  --->  Postgres
                                         |  |
                                         |  +--> Redis (assignment cache)
                                         |
                                         +--> goroutines batch exposures / events
                                         |
                                         +--> gRPC (telemetry) :9090
```

1. A browser hits any Next.js page. `getServerSideProps` fires a same-region fetch to the
   Go backend for the product catalog plus the user's experiment assignments.
2. The backend reads the `pf_uid` cookie (or issues one) and computes each active
   experiment's assignment from `sha256(salt || key || user_id || domain)`. The same user
   always lands on the same variant, even with a cold cache.
3. Assignments for treatment variants queue onto a buffered Go channel. A background
   goroutine batches them into `exposures` via `pgx.CopyFrom` every 2 seconds.
4. The page renders with the correct variant baked into the HTML, so there is no post-hydration
   flash when the user toggles between, say, control and treatment copy on the hero.
5. Clicks, add-to-carts, and purchases post back to `/api/v1/events`, which feeds the
   `ConversionByVariant` report query at `/api/v1/experiments/{key}/report`.

## API routes

```
GET  /healthz
GET  /readyz
GET  /api/v1/products
GET  /api/v1/products/{id}
GET  /api/v1/experiments
GET  /api/v1/experiments/{key}/report?event=purchase&since=RFC3339
GET  /api/v1/assignments
POST /api/v1/events
```

gRPC: `pulsefront.telemetry.v1.TelemetryService.RecordExposure` and `RecordEvent`.

## Layout

```
backend/                    # Go module
  cmd/server/main.go        # wiring and graceful shutdown
  internal/config           # env parsing
  internal/db               # pgx pool + embedded migration
  internal/cache            # redis client
  internal/experiments      # typed A/B framework (registry, bucketing, service)
  internal/analytics        # buffered batch writer, conversion query
  internal/catalog          # product reads
  internal/httpapi          # chi router, middleware, handlers
  internal/grpcapi          # telemetry gRPC server + generated pb
  internal/seed             # product and experiment seed data
  proto/telemetry.proto

frontend/                   # Next.js app
  src/pages/                # SSR pages and api proxy
  src/components/           # Layout, ProductCard
  src/lib/                  # API client, experiment helpers
  src/styles/               # globals.css
  __tests__/                # jest unit tests
```
