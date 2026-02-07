---
doc: 04_test_plan
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

# Test Plan

## Scope

- Covered:
  - DB configuration parsing and validation paths.
  - Application startup initialization orchestration (readiness + migrations).
  - PostgreSQL outbound adapter behavior.
  - Local `docker compose` startup flow for app + PostgreSQL.
- Not covered:
  - Production deployment behavior and cloud-managed PostgreSQL settings.
  - High-volume load/perf testing beyond basic startup/health checks.

## Tests

### Unit

- TC-001: DB config validation

  - Linked requirements: FR-001, NFR-003, NFR-006
  - Steps: Execute config-loading unit tests for valid `DATABASE_URL`, missing value, and malformed URL value.
  - Expected: Valid `DATABASE_URL` is accepted; missing/malformed value returns typed error and prevents startup.

- TC-002: Startup initialization use case orchestration
  - Linked requirements: FR-002, FR-003, NFR-002, NFR-006
  - Steps: Test use case with mocked outbound port to simulate readiness success, transient failure then success, and timeout.
  - Expected: Use case retries within configured bounds, succeeds when DB becomes ready, and returns deterministic timeout/failure errors otherwise.

### Integration

- TC-101: PostgreSQL adapter readiness success

  - Linked requirements: FR-002, FR-003, NFR-001, NFR-005
  - Steps: Run adapter integration test against a running PostgreSQL instance (containerized local DB) with valid credentials.
  - Expected: Adapter establishes connection/ping and reports readiness success.

- TC-102: PostgreSQL adapter readiness failure

  - Linked requirements: FR-003, NFR-002, NFR-005
  - Steps: Run adapter test with invalid credentials/host encoded in `DATABASE_URL` and capture retry + timeout behavior.
  - Expected: Adapter/use case returns failure within <= 30 seconds with clear error code/message.

- TC-103: Migration idempotency
  - Linked requirements: FR-004, NFR-002, NFR-005
  - Steps: Execute startup initialization twice against same DB, then query `app.bootstrap_metadata` and inspect `golang-migrate` bookkeeping state.
  - Expected: `app.bootstrap_metadata` contains exactly one `bootstrap_version` row; second run applies no new migration and `schema_migrations` shows no duplicate apply.

### E2E (if applicable)

- TC-201: Compose happy-path local startup

  - Linked requirements: FR-005, FR-006, NFR-001, NFR-003
  - Steps: Run `docker compose up --build`; wait for app logs; call `GET /healthz`.
  - Expected: Both services run, app starts after DB readiness, and `GET /healthz` responds successfully.

- TC-202: Compose DB failure path
  - Linked requirements: FR-003, FR-005, NFR-002, NFR-005
  - Steps: Break `DATABASE_URL` (for example invalid password), start app service, observe exit behavior and logs.
  - Expected: App exits non-zero without serving HTTP, logs indicate DB initialization failure cause.

## Edge cases and failure modes

- Case: PostgreSQL starts slowly but becomes ready before timeout.
- Expected behavior: App retries and eventually starts successfully without manual restart.

- Case: Migration SQL syntax/runtime error.
- Expected behavior: Startup stops before HTTP server begins listening; error is logged clearly.

- Case: DB password contains special characters.
- Expected behavior: Connection string handling remains correct and credentials are not leaked in logs.

## NFR verification

- Performance:
  - Verify startup timing targets from FR/NFR (`<=60s` first compose build path; `<=20s` warm restart excluding image pulls).
- Reliability:
  - Verify DB readiness timeout and fail-fast behavior (`<=30s` bounded startup wait).
- Security:
  - Verify logs redact secrets and credentials from `DATABASE_URL`.
