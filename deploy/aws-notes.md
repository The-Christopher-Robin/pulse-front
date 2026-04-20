# AWS deployment notes

This repo runs on Docker locally, but the target production shape is ECS Fargate
behind an ALB, with RDS Postgres, ElastiCache Redis, and CloudFront in front of
the Next.js origin. This doc sketches what changes between local and cloud.

## Services

- **Next.js (frontend)**: one ECS service, 2+ tasks, behind the public ALB on :443.
  CloudFront sits in front with a cache behaviour that only caches static assets
  (`_next/static/*`) and always forwards `/`, `/product/*`, and `/experiments`
  because those are server-rendered per user.

- **Go backend**: a second ECS service, 2+ tasks, behind an internal ALB on :8080.
  The Next.js SSR fetches hit this internal ALB over the VPC. Browser-originated
  calls go through `/api/proxy/*` on the Next side so cookies stay same-origin
  and no public CORS window is needed.

- **gRPC telemetry**: same Go task exposes :9090. Internal Network Load Balancer
  (NLB) or ALB with gRPC listener in the VPC. The Go server already speaks
  HTTP/2 via `grpc-go` so gRPC-over-ALB works out of the box.

- **RDS Postgres** in the same VPC, private subnets. `POSTGRES_URL` is wired from
  Secrets Manager. The `001_init.sql` migration is idempotent and runs on boot.

- **ElastiCache Redis** (single node for dev, cluster mode disabled). Latency from
  backend to Redis is single-digit ms in the same AZ.

## Traffic path

```
user -> CloudFront -> public ALB -> Next.js (Fargate)
                                       |
                                       +-- SSR fetch --> internal ALB -> Go backend -> RDS / ElastiCache
                                       |
                                       +-- /api/proxy/* (browser XHR)
```

## Scaling

- ECS service auto-scaling on CPU > 60% and p95 target response time.
- The exposure writer batches via Go channels so the hot request path never blocks
  on Postgres. Batch size and flush interval are env-tunable (`EXPOSURE_BUFFER`,
  `EXPOSURE_FLUSH_INTERVAL`).
- Redis caches per-user assignment snapshots for 5 minutes to keep re-render
  latency flat when marketing launches a new experiment.

## Observability

- JSON access logs on stdout land in CloudWatch Logs via the Fargate awslogs
  driver.
- ALB 5xx target tracking alarms page on rolling 5-minute windows.
- Conversion report endpoint (`/api/v1/experiments/{key}/report`) is the read
  path for the internal dashboard that marketing uses to call an experiment.

## What is deliberately not in this repo

This repo is a one-region, two-service reference. IAM policies, Terraform, blue/green
deploy config, and the CloudFront behaviour catalogue live in a separate infra repo.
