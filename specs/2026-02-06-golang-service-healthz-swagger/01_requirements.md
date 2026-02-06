---
doc: 01_requirements
spec_date: 2026-02-06
slug: golang-service-healthz-swagger
mode: Quick
status: DONE
owners:
  - posen
depends_on: []
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: null
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Requirements

## Glossary (optional)

- Health check: A lightweight endpoint that confirms service process readiness/liveness.
- Swagger UI: Browser-rendered API documentation generated from an OpenAPI document.

## Out-of-scope behaviors

- OOS1: Any POST/PUT/DELETE API.
- OOS2: Persistent storage and migration scripts.
- OOS3: Authentication and RBAC.

## Functional requirements

### FR-001 - Bootstrap runnable Go HTTP service

- Description: Provide a project structure and executable entrypoint for a backend HTTP server.
- Acceptance criteria:
  - [ ] Repository includes `go.mod` and a `main` entrypoint that starts an HTTP server.
  - [ ] Server binds to `PORT` when provided, otherwise defaults to `8080`.
  - [ ] Server logs startup and shuts down gracefully on termination signal.
- Notes: Keep implementation straightforward and easy to extend.

### FR-002 - Implement `GET /healthz`

- Description: Expose a health endpoint for probes and local verification.
- Acceptance criteria:
  - [ ] `GET /healthz` responds with HTTP `200`.
  - [ ] Response body is JSON containing at least `{ "status": "ok" }`.
  - [ ] Non-GET method to `/healthz` is rejected by routing behavior (not `200`).
- Notes: No dependency checks required in this phase.

### FR-003 - Expose Swagger docs at `/swagger`

- Description: Provide interactive API documentation route backed by OpenAPI spec.
- Acceptance criteria:
  - [ ] `GET /swagger` redirects to `/swagger/index.html` (or equivalent docs entrypoint).
  - [ ] `GET /swagger/index.html` serves Swagger UI.
  - [ ] OpenAPI document uses `openapi: 3.0.3`.
  - [ ] OpenAPI document contains `/healthz` endpoint definition with `200` response.
- Notes: OpenAPI `3.0.3` can be static file-based for deterministic bootstrapping.

### FR-004 - Provide local developer automation via `Makefile`

- Description: Add make targets for common local workflows.
- Acceptance criteria:
  - [ ] `make run` starts the API server locally.
  - [ ] `make test` executes unit/integration tests.
  - [ ] `make lint` runs baseline static checks supported by repository tooling.
- Notes: Commands must fail fast with non-zero exit codes on error.

### FR-005 - Document local usage

- Description: Update top-level docs with run/test/docs usage.
- Acceptance criteria:
  - [ ] `README.md` explains how to run service and call `/healthz`.
  - [ ] `README.md` includes Swagger docs URL.
- Notes: Keep instructions copy-paste friendly.

## Non-functional requirements

- Performance (NFR-001): `GET /healthz` local p95 latency <= `50ms` under single-request load.
- Availability/Reliability (NFR-002): Server startup failure exits with non-zero code and clear log message.
- Security/Privacy (NFR-003): Endpoint responses must not expose environment values, tokens, or stack traces.
- Compliance (NFR-004): Use only OSS dependencies with permissive licenses already common in Go ecosystem.
- Observability (NFR-005): Log at least startup, shutdown, and request-level errors with timestamp.
- Maintainability (NFR-006): New routes can be added without modifying health handler logic (separation of concerns).

## Dependencies and integrations

- External systems: None.
- Internal services: None.
- Toolchain/runtime: Go >= `1.22`, GNU Make.
- Libraries: `github.com/swaggo/http-swagger/v2` for Swagger UI serving.
