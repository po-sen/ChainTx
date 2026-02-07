---
doc: 01_requirements
spec_date: 2026-02-07
slug: postgresql-backend-service-compose
mode: Full
status: DONE
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

# Requirements

## Glossary (optional)

- DB: PostgreSQL database instance used by the backend service.
- Outbound port: Application-layer interface that defines persistence capability requirements.
- Outbound adapter: Infrastructure implementation of outbound ports, including SQL driver interactions.

## Out-of-scope behaviors

- OOS1: Production HA/failover PostgreSQL topology.
- OOS2: Multi-tenant database isolation and sharding.

## Functional requirements

### FR-001 - Add single database URL configuration

- Description: Service must support PostgreSQL runtime configuration through a single environment variable loaded in bootstrap/config.
- Acceptance criteria:
  - [ ] AC1: App-level DB config accepts only `DATABASE_URL` (no additional `DB_*` app configuration fields).
  - [ ] AC2: Invalid/missing required DB config results in deterministic startup failure with structured error output.
  - [ ] AC3: Configuration parsing remains in bootstrap/infrastructure; no config parsing logic is added to domain/application layers.
- Notes: Keep credential parts of `DATABASE_URL` redacted in logs.

### FR-002 - Add PostgreSQL outbound port and adapter

- Description: Persistence capability must be defined in `internal/application/ports/out` and implemented by a PostgreSQL adapter under outbound adapters.
- Acceptance criteria:
  - [ ] AC1: At least one outbound port interface captures required DB capability (connectivity and migration execution) without exposing vendor/driver types.
  - [ ] AC2: PostgreSQL adapter implements the outbound port and is wired only in `internal/bootstrap/di`.
  - [ ] AC3: Domain and application layers compile without importing SQL/driver packages.
- Notes: Use constructor injection from DI to use cases.

### FR-003 - Enforce database readiness before serving traffic

- Description: Service startup must verify PostgreSQL connectivity before HTTP server starts listening.
- Acceptance criteria:
  - [ ] AC1: On successful DB readiness, service starts normally and serves existing HTTP endpoints.
  - [ ] AC2: On failed DB readiness, process exits with non-zero code and clear startup error logs.
  - [ ] AC3: Startup flow includes bounded retry/timeout behavior to avoid indefinite hangs.
- Notes: This is startup-time enforcement and does not mandate new public endpoints.

### FR-004 - Add migration workflow for baseline schema

- Description: Service must execute SQL migration steps on startup prior to accepting requests, using `golang-migrate` as the primary migration engine.
- Acceptance criteria:
  - [ ] AC1: Migration artifacts are versioned in repo with deterministic ordering.
  - [ ] AC2: Migration execution is idempotent across repeated startups.
  - [ ] AC3: Migration failures prevent server start and produce actionable error logs.
  - [ ] AC4: Migration runner integration uses `github.com/golang-migrate/migrate/v4` and supports `golang-migrate`-compatible file naming.
  - [ ] AC5: Baseline migration creates `app` schema and `app.bootstrap_metadata` table, and seeds key `bootstrap_version` without duplicate inserts on repeated startup.
- Notes: Migration implementation should keep tool-specific details inside outbound adapter/infrastructure boundaries.

### FR-005 - Provide local docker compose orchestration

- Description: Repository must include a simple Compose setup for backend service plus PostgreSQL.
- Acceptance criteria:
  - [ ] AC1: `deployments/docker-compose.yml` defines at minimum `app` and `postgres` services.
  - [ ] AC2: PostgreSQL service uses official image `postgres:latest`, includes a persistent volume, and defines healthcheck.
  - [ ] AC3: App service receives `DATABASE_URL` from compose and waits for PostgreSQL healthy status before startup.
- Notes: Keep compose focused on local development.

### FR-006 - Document local DB workflow and commands

- Description: Developer docs must explain local startup/shutdown and DB-related environment variables.
- Acceptance criteria:
  - [ ] AC1: `README.md` includes compose quickstart commands and expected endpoints.
  - [ ] AC2: Required app variable `DATABASE_URL` and its compose default are documented.
  - [ ] AC3: Troubleshooting guidance includes at least one DB-connection failure case.
- Notes: Documentation should stay aligned with actual compose/service behavior.

### FR-007 - Preserve clean architecture and Go layout boundaries

- Description: New DB-related code must follow current clean architecture + hexagonal boundaries and existing Go project layout.
- Acceptance criteria:
  - [ ] AC1: `cmd/server/main.go` remains orchestration-only; business logic stays in `internal/application` and `internal/domain`.
  - [ ] AC2: Domain/application do not import `internal/adapters`, `internal/infrastructure`, or DB driver packages.
  - [ ] AC3: All new persistence implementations live under outbound adapter/infrastructure paths inside `internal/`.
- Notes: This requirement is architectural and applies to all implementation tasks.

## Non-functional requirements

- Performance (NFR-001): From `docker compose -f deployments/docker-compose.yml up --build`, service must become reachable at `http://localhost:8080/healthz` within 60 seconds on first build and within 20 seconds on subsequent starts (excluding image pull time).
- Availability/Reliability (NFR-002): Startup DB readiness checks must use bounded retry with total timeout <= 30 seconds, then fail fast with non-zero exit.
- Security/Privacy (NFR-003): DB credentials must be sourced from `DATABASE_URL`; logs must never print raw credential segments from `DATABASE_URL`; committed defaults are for local development only.
- Compliance (NFR-004): No compliance scope change in this increment; no PII schema is introduced.
- Observability (NFR-005): Startup logs must include DB readiness and migration step outcomes with error codes/messages suitable for triage.
- Maintainability (NFR-006): `go test ./...`, `go vet ./...`, and `go list ./...` must pass after implementation; package placement must remain consistent with existing layout.

## Dependencies and integrations

- External systems: Official PostgreSQL container image using tag `postgres:latest`.
- Internal services: Existing HTTP service bootstrap and DI container.
- Libraries/tools: `github.com/golang-migrate/migrate/v4` is the primary migration library.
