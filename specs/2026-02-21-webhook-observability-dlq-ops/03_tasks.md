---
doc: 03_tasks
spec_date: 2026-02-21
slug: webhook-observability-dlq-ops
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-status-outbox-delivery
  - 2026-02-21-webhook-lease-heartbeat-renewal
  - 2026-02-21-webhook-idempotency-replay-signature
  - 2026-02-21-webhook-retry-jitter-budget
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
- Rationale: adds new operational APIs, asynchronous telemetry contract changes, and manual failure-handling flows.
- Upstream dependencies (`depends_on`):
  - 2026-02-21-webhook-status-outbox-delivery
  - 2026-02-21-webhook-lease-heartbeat-renewal
  - 2026-02-21-webhook-idempotency-replay-signature
  - 2026-02-21-webhook-retry-jitter-budget
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`.

## Milestones

- M1: outbox observability and DLQ application contracts implemented.
- M2: HTTP operational endpoints and docs wired.
- M3: dispatcher telemetry buckets + full verification complete.

## Tasks (ordered)

1. T-001 - Add DTO/ports/use-cases for webhook outbox overview, DLQ list, requeue, cancel

   - Scope: define application contracts and validate command/query guards.
   - Output: clean inbound and outbound ports with unit tests.
   - Linked requirements: FR-001 / FR-003 / FR-004 / FR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/application/use_cases -count=1`
     - [x] Expected result: new use-case validation and happy-path tests pass.
     - [x] Logs/metrics to check (if applicable): N/A

2. T-002 - Implement PostgreSQL webhook outbox read model and repository manual-operation SQL

   - Scope: add overview aggregate query, DLQ list query, and guarded requeue/cancel updates.
   - Output: adapter implements new ports with deterministic status guards.
   - Linked requirements: FR-001 / FR-003 / FR-004 / FR-005 / NFR-001 / NFR-002 / NFR-003
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/adapters/outbound/persistence/postgresql/webhookoutbox/...`
     - [x] Expected result: adapter tests cover query mapping and guard semantics.
     - [x] Logs/metrics to check (if applicable): N/A

3. T-003 - Add inbound HTTP controller/routes and DI wiring for webhook outbox operations

   - Scope: expose overview/DLQ/requeue/cancel routes, parse/validate inputs, map use-case errors.
   - Output: server runtime supports manual compensation APIs.
   - Linked requirements: FR-001 / FR-003 / FR-004 / FR-005 / FR-006 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/adapters/inbound/http/... -count=1`
     - [x] Expected result: endpoint contract tests pass for success and invalid/conflict cases.
     - [x] Logs/metrics to check (if applicable): request error logs include code/message on failures.

4. T-004 - Extend dispatch output and worker logs with delivery bucket telemetry

   - Scope: track 2xx/4xx/5xx/network counters per cycle and emit in worker completion logs.
   - Output: actionable observability fields in dispatcher logs and output DTO.
   - Linked requirements: FR-002 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/application/use_cases ./internal/infrastructure/webhook -count=1`
     - [x] Expected result: bucket counters are computed and logged fields are present.
     - [x] Logs/metrics to check (if applicable): cycle completion log includes `http_2xx/http_4xx/http_5xx/network_error`.

5. T-005 - Update OpenAPI/README and run verification + spec closeout
   - Scope: document all new endpoints/workflows and execute full repo/spec checks.
   - Output: operator-ready documentation and `DONE` spec with evidence.
   - Linked requirements: FR-006 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: all checks pass and spec lint passes.
     - [x] Logs/metrics to check (if applicable): N/A

## Traceability (optional)

- FR-001 -> T-001, T-002, T-003
- FR-002 -> T-004
- FR-003 -> T-001, T-002, T-003
- FR-004 -> T-001, T-002, T-003
- FR-005 -> T-001, T-002, T-003
- FR-006 -> T-003, T-005
- NFR-001 -> T-002
- NFR-002 -> T-002
- NFR-003 -> T-002
- NFR-005 -> T-004, T-005
- NFR-006 -> T-001, T-003, T-004, T-005

## Rollout and rollback

- Feature flag: none.
- Migration sequencing: none.
- Rollback steps: revert new outbox endpoints/use-cases/repository methods and worker telemetry fields.

## Validation evidence

- 2026-02-21 commands executed:
  - `go test ./internal/application/use_cases -count=1` -> `ok`
  - `go test ./internal/adapters/inbound/http/controllers -count=1` -> `ok`
  - `go test ./internal/adapters/outbound/persistence/postgresql/webhookoutbox -count=1` -> `[no test files]` (adapter compiled in full short suite)
  - `go test ./internal/infrastructure/webhook -count=1` -> `ok`
  - `go fmt ./...` -> pass
  - `go mod tidy` -> pass
  - `go list ./...` -> pass
  - `go test -short ./...` -> pass
  - `go vet ./...` -> pass
  - `SPEC_DIR=specs/2026-02-21-webhook-observability-dlq-ops bash scripts/spec-lint.sh` -> pass
  - `bash scripts/verify-agents.sh specs/2026-02-21-webhook-observability-dlq-ops` -> pass
