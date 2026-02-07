---
doc: 03_tasks
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

# Task Plan

## Mode decision

- Selected mode: Full
- Rationale: This change introduces a new persistent store (PostgreSQL), schema migration workflow, and local infrastructure integration through Docker Compose.
- Migration tool decision: Use `golang-migrate` as the primary migration engine.
- Upstream dependencies (`depends_on`):
  - 2026-02-06-golang-service-healthz-swagger
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`
- If `02_design.md` is skipped (Quick mode): Not applicable; Full mode required.
- If `04_test_plan.md` is skipped: Not applicable; Full mode requires test plan.

## Milestones

- M1: Persistence bootstrap architecture and PostgreSQL adapter contract defined.
- M2: Local compose orchestration and migration-enabled startup flow implemented.
- M3: Validation, documentation, and operational developer workflow complete.

## Tasks (ordered)

1. T-001 - Add DB configuration and startup wiring

   - Scope: Extend bootstrap config to parse one `DATABASE_URL` input and wire persistence initialization use case in DI.
   - Output: DB config support for `DATABASE_URL` plus DI container updates without business logic in `cmd/`.
   - Linked requirements: FR-001, FR-007, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): `go test ./...`
     - [ ] Expected result: Build/tests pass with new config fields and DI wiring.
     - [ ] Logs/metrics to check (if applicable): Startup logs include only redacted DB target details (no raw URL credentials).

2. T-002 - Define application ports and initialization use case

   - Scope: Add inbound port/use case for startup persistence initialization and outbound port contracts for readiness + migration.
   - Output: Application-layer interfaces/use case with unit tests using mocked outbound ports.
   - Linked requirements: FR-002, FR-003, FR-007, NFR-002, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): `go test ./internal/application/...`
     - [ ] Expected result: Unit tests cover success/failure/timeout paths.
     - [ ] Logs/metrics to check (if applicable): Structured error mapping includes stable code/message.

3. T-003 - Implement PostgreSQL outbound adapter

   - Scope: Implement PostgreSQL adapter for readiness checks and DB connection lifecycle.
   - Output: Outbound adapter package under `internal/adapters/outbound/persistence/postgresql` implementing application ports.
   - Linked requirements: FR-002, FR-003, FR-007, NFR-001, NFR-002, NFR-005
   - Validation:
     - [ ] How to verify (manual steps or command): adapter integration test against local PostgreSQL instance/container.
     - [ ] Expected result: Adapter reports readiness success when DB is available and typed failure when unavailable.
     - [ ] Logs/metrics to check (if applicable): Retry attempts and final readiness result are logged.

4. T-004 - Add SQL migration workflow

   - Scope: Add `golang-migrate`-compatible migration files (`000001_bootstrap_metadata.up.sql`, `000001_bootstrap_metadata.down.sql`) and execute migrations during startup initialization before HTTP server starts.
   - Output: Versioned migration directory + `golang-migrate` runner integration with idempotent behavior; baseline migration creates `app.bootstrap_metadata`.
   - Linked requirements: FR-004, FR-003, NFR-002, NFR-005
   - Validation:
     - [ ] How to verify (manual steps or command): start service twice against same DB, then query `app.bootstrap_metadata` and `schema_migrations`.
     - [ ] Expected result: First run creates schema/table and inserts `bootstrap_version`; second run performs no duplicate inserts or destructive work.
     - [ ] Logs/metrics to check (if applicable): Applied/skipped migration counts are logged.

5. T-005 - Create simple docker compose stack

   - Scope: Add root `docker-compose.yml` for app + PostgreSQL with `postgres:latest`, persistent volume, and startup dependency.
   - Output: Compose file enabling one-command local startup.
   - Linked requirements: FR-005, NFR-001, NFR-003
   - Validation:
     - [ ] How to verify (manual steps or command): `docker compose up --build` then `curl -i http://localhost:8080/healthz`
     - [ ] Expected result: Both containers are healthy/running and HTTP endpoint is reachable.
     - [ ] Logs/metrics to check (if applicable): App logs show successful DB readiness and migration completion.

6. T-006 - Update developer docs and command workflow

   - Scope: Update `README.md` (and optional Make targets) with compose usage, env defaults, and troubleshooting.
   - Output: Clear local runbook for DB-enabled service.
   - Linked requirements: FR-006, NFR-006, NFR-005
   - Validation:
     - [ ] How to verify (manual steps or command): follow documented steps on a clean local environment.
     - [ ] Expected result: User can reproduce startup/shutdown and troubleshoot common DB errors.
     - [ ] Logs/metrics to check (if applicable): Documented error examples match real startup failure output.

7. T-007 - Execute quality gates and traceability review
   - Scope: Run full verification and confirm requirement-to-task/test traceability.
   - Output: Passing validation evidence and updated spec status decision.
   - Linked requirements: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, NFR-001, NFR-002, NFR-003, NFR-005, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): `go fmt ./... && go vet ./... && go list ./... && go test ./...`
     - [ ] Expected result: All checks pass; traceability table remains complete.
     - [ ] Logs/metrics to check (if applicable): No unexpected startup/runtime errors in smoke tests.

## Traceability (optional)

- FR-001 -> T-001, T-007
- FR-002 -> T-002, T-003, T-007
- FR-003 -> T-002, T-003, T-004, T-005, T-007
- FR-004 -> T-004, T-007
- FR-005 -> T-005, T-007
- FR-006 -> T-006, T-007
- FR-007 -> T-001, T-002, T-003, T-007
- NFR-001 -> T-003, T-005, T-007
- NFR-002 -> T-002, T-003, T-004, T-007
- NFR-003 -> T-005, T-007
- NFR-005 -> T-003, T-004, T-006, T-007
- NFR-006 -> T-001, T-002, T-006, T-007

## Rollout and rollback

- Feature flag: Not required; rollout is controlled by deployment of DB-enabled build and compose workflow.
- Migration sequencing: Run migrations before starting HTTP listener; abort startup on migration failure.
- Rollback steps: Revert DB integration changes, stop compose stack, and remove local DB volume only when data reset is acceptable for dev.
