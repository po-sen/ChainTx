---
doc: 02_design
spec_date: 2026-02-07
slug: postgresql-backend-service-compose
mode: Full
status: READY
owners:
  - posen
depends_on:
  - 2026-02-06-golang-service-healthz-swagger
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Technical Design

## High-level approach

- Summary: Add a startup-time persistence initialization use case in the application layer, backed by a PostgreSQL outbound adapter, then orchestrate both app and DB locally via `docker-compose.yml`.
- Key decisions:
  - Keep current single-module layout (`cmd/` + `internal/`) and avoid introducing bounded-context folders.
  - Use one app-level DB configuration input: `DATABASE_URL`.
  - Use `golang-migrate` (`github.com/golang-migrate/migrate/v4`) as the primary migration engine.
  - Add DB capabilities via outbound ports in `internal/application/ports/out` and outbound adapter implementation in `internal/adapters/outbound/persistence/postgresql`.
  - Run DB readiness checks and migrations before HTTP `ListenAndServe` starts.

## System context

- Components:
  - `cmd/server/main.go`: process lifecycle orchestration only.
  - `internal/bootstrap/config.go`: parse runtime env config, including `DATABASE_URL`.
  - `internal/bootstrap/di/container.go`: bind outbound ports to PostgreSQL adapter and build use cases/controllers.
  - `internal/application/ports/in`: inbound port for startup persistence initialization use case.
  - `internal/application/ports/out`: outbound DB capability contracts.
  - `internal/application/use_cases`: orchestration for DB readiness + migration execution.
  - `internal/adapters/outbound/persistence/postgresql`: DB connection, ping, migrations.
  - `docker-compose.yml`: local app + PostgreSQL runtime topology (`postgres:latest`).
- Interfaces:
  - Inbound port example: `InitializePersistenceUseCase.Execute(ctx)`.
  - Outbound port example: `PersistenceBootstrapGateway` with methods for readiness check and migrations.
  - HTTP API remains existing `GET /healthz` in this increment.

## Key flows

- Flow 1: Happy path startup
  - `docker compose up --build` starts `postgres` first.
  - App container starts, loads `DATABASE_URL`, resolves outbound adapter via DI.
  - Startup use case executes DB readiness check and migrations.
  - If successful, HTTP server starts and `GET /healthz` is reachable.
- Flow 2: DB unavailable
  - App startup use case retries readiness with bounded timeout.
  - After timeout, it returns a structured error.
  - Main process exits non-zero; HTTP server does not start.
- Flow 3: Repeated startup
  - Migration step re-runs safely.
  - Already-applied migrations are skipped; no destructive re-apply occurs.

## Diagrams (optional)

- Mermaid sequence / flow:

```mermaid
sequenceDiagram
  participant C as Compose
  participant A as App(main)
  participant U as InitPersistenceUseCase
  participant P as PostgresAdapter
  participant D as PostgreSQL

  C->>D: start postgres
  C->>A: start app
  A->>U: Execute(ctx)
  U->>P: CheckReadiness(ctx)
  P->>D: ping/connect
  U->>P: RunMigrations(ctx)
  P->>D: apply pending migrations
  U-->>A: success
  A->>A: start HTTP server
```

## Data model

- Entities:
  - No new domain entity is required in this increment; scope is infrastructure readiness.
- Schema changes or migrations:
  - Introduce migration directory with ordered SQL files using `golang-migrate` naming (`*.up.sql`, `*.down.sql`).
  - Baseline migration (`000001_bootstrap_metadata`) creates minimal service-owned schema artifact.
  - Concrete baseline migration example:

```sql
-- 000001_bootstrap_metadata.up.sql
CREATE SCHEMA IF NOT EXISTS app;

CREATE TABLE IF NOT EXISTS app.bootstrap_metadata (
  key text PRIMARY KEY,
  value text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO app.bootstrap_metadata (key, value)
VALUES ('bootstrap_version', 'v1')
ON CONFLICT (key) DO NOTHING;
```

```sql
-- 000001_bootstrap_metadata.down.sql
DROP TABLE IF EXISTS app.bootstrap_metadata;
DROP SCHEMA IF EXISTS app;
```

- `golang-migrate` maintains `schema_migrations`; do not create this table manually.
- Consistency and idempotency:
  - Migrations must be idempotent and safe on repeated startup.
  - No cross-service transaction workflow in this increment.

## API or contracts

- Endpoints or events:
  - No new HTTP endpoint is required; existing endpoints remain available once startup initialization succeeds.
- Request/response examples:
  - Env contract:
    - `DATABASE_URL=postgresql://chaintx:chaintx@postgres:5432/chaintx?sslmode=disable`

## Backward compatibility (optional)

- API compatibility:
  - Existing `/healthz` and Swagger paths remain intact.
- Data migration compatibility:
  - Initial migration introduces new DB objects only; no backward-incompatible data transformation.

## Failure modes and resiliency

- Retries/timeouts:
  - Startup readiness retries until success or a total timeout of 30 seconds.
- Backpressure/limits:
  - Connection pool limits are explicitly configured for local environment defaults.
- Degradation strategy:
  - On startup DB failure, fail fast and exit; avoid running partially initialized service.

## Observability

- Logs:
  - Log DB readiness attempts, readiness success/failure, and migration outcomes.
  - Never log raw credential segments from `DATABASE_URL`.
- Metrics:
  - Capture startup timing and migration duration where practical.
- Traces:
  - Not required for this local-first increment.
- Alerts:
  - Not in scope for local compose; rely on logs/exit codes.

## Security

- Authentication/authorization:
  - DB authentication via credentials encoded in `DATABASE_URL`.
- Secrets:
  - Local defaults may exist in compose for dev only; production secrets are out of scope.
- Abuse cases:
  - Prevent accidental credential disclosure via logs.

## Alternatives considered

- Option A: Keep DB logic directly in bootstrap/main.
- Option B: Add outbound ports + application startup use case + outbound adapter.
- Why chosen: Option B preserves clean architecture boundaries and keeps domain/application independent from database drivers.

## Risks

- Risk: Compose startup race can cause transient connection failures.
- Mitigation: Add bounded startup retry with clear timeout and logs.
- Risk: Migration engine/tool choice delay may stall implementation.
- Mitigation: Tool choice is fixed to `golang-migrate`; keep migration interface minimal and isolate tool-specific code in the outbound adapter.
