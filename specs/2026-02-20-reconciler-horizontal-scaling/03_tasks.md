---
doc: 03_tasks
spec_date: 2026-02-20
slug: reconciler-horizontal-scaling
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
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
- Rationale: includes DB migration, runtime role split, and concurrency semantics changes.
- Upstream dependencies (`depends_on`):
  - `2026-02-20-chain-listener-payment-reconcile`
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`

## Milestones

- M1: claim/lease contract and schema ready.
- M2: use case + worker + standalone runtime implemented.
- M3: compose/make scaling workflow and validation complete.

## Tasks (ordered)

1. T-001 - Add lease columns and claim index migration
   - Scope: add schema support for reconciler lease ownership and claim optimization.
   - Output: migration files for adding/removing lease columns and index.
   - Linked requirements: FR-002, FR-003, NFR-001, NFR-003
   - Validation:
     - [x] How to verify (manual steps or command): run startup migration and inspect columns/index in DB.
     - [x] Expected result: migration applies and rollbacks cleanly.
     - [x] Logs/metrics to check (if applicable): migration startup logs.
2. T-002 - Refactor reconciliation repository to claim-first model
   - Scope: replace list-open query with atomic claim query and lease-aware transition update.
   - Output: updated repository port and PostgreSQL adapter implementation.
   - Linked requirements: FR-002, FR-003, NFR-001, NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/application/use_cases ./internal/adapters/outbound/persistence/postgresql/paymentrequest`
     - [x] Expected result: claim and transition tests pass; no compile breaks in dependents.
     - [x] Logs/metrics to check (if applicable): N/A
3. T-003 - Extend command/config for lease duration and worker identity
   - Scope: add lease settings into config and reconcile command payload.
   - Output: validated config fields and worker command wiring.
   - Linked requirements: FR-004, NFR-003, NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/infrastructure/config ./internal/infrastructure/reconciler`
     - [x] Expected result: invalid lease config fails; worker command includes worker ID and lease duration.
     - [x] Logs/metrics to check (if applicable): worker startup log includes worker ID.
4. T-004 - Add standalone reconciler runtime binary
   - Scope: create `cmd/reconciler` entrypoint and Docker image support.
   - Output: runnable reconciler-only process with graceful shutdown.
   - Linked requirements: FR-001, FR-006, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./...` and run `go run ./cmd/reconciler` with local env.
     - [x] Expected result: process starts without HTTP listener, runs cycles, exits cleanly on signal.
     - [x] Logs/metrics to check (if applicable): start/stop and cycle logs.
5. T-005 - Add dedicated reconciler compose service and scaling knobs
   - Scope: update compose and make workflow for separate worker service and replica scaling.
   - Output: one-command startup path for horizontal replicas.
   - Linked requirements: FR-005, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `make service-up SERVICE_RECONCILER_ENABLED=true SERVICE_RECONCILER_REPLICAS=2 ...`
     - [x] Expected result: two reconciler containers run; app stays without embedded worker by default.
     - [x] Logs/metrics to check (if applicable): per-worker cycle logs show distinct worker IDs.
6. T-006 - Run multi-replica smoke validation and finalize spec
   - Scope: run validation commands and runtime checks, then promote docs to DONE.
   - Output: test evidence captured in spec task/test docs.
   - Linked requirements: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, NFR-001, NFR-002, NFR-003, NFR-004, NFR-005, NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `SPEC_DIR=specs/2026-02-20-reconciler-horizontal-scaling bash scripts/spec-lint.sh && bash scripts/verify-agents.sh specs/2026-02-20-reconciler-horizontal-scaling && go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: all checks pass and runtime evidence confirms parallel-safe claiming.
     - [x] Logs/metrics to check (if applicable): no duplicate claim ownership logs for same request within lease window.

## Traceability (optional)

- FR-001 -> T-004, T-006
- FR-002 -> T-001, T-002, T-006
- FR-003 -> T-001, T-002, T-006
- FR-004 -> T-003, T-006
- FR-005 -> T-005, T-006
- FR-006 -> T-004, T-006
- NFR-001 -> T-001, T-002, T-006
- NFR-002 -> T-006
- NFR-003 -> T-001, T-003, T-006
- NFR-004 -> T-006
- NFR-005 -> T-004, T-005, T-006
- NFR-006 -> T-002, T-003, T-006

## Rollout and rollback

- Feature flag: `PAYMENT_REQUEST_RECONCILER_ENABLED` (for reconciler role process).
- Migration sequencing: deploy migration first, then start reconciler replicas.
- Rollback steps: scale reconciler to 0/disable profile, deploy previous binaries, rollback migration if required.

## Execution evidence

- Verification date: 2026-02-20
- Commands executed:
  - `go fmt ./...`
  - `go mod tidy`
  - `go list ./...`
  - `go test -short ./...`
  - `go vet ./...`
  - `SPEC_DIR=specs/2026-02-20-reconciler-horizontal-scaling bash scripts/spec-lint.sh`
  - `bash scripts/verify-agents.sh specs/2026-02-20-reconciler-horizontal-scaling`
  - `SERVICE_RECONCILER_ENABLED=true SERVICE_RECONCILER_REPLICAS=2 SERVICE_RECONCILER_POLL_INTERVAL_SECONDS=2 SERVICE_RECONCILER_LEASE_SECONDS=15 SERVICE_EVM_RPC_URLS_JSON='{"local":"http://host.docker.internal:8545"}' make local-up-all`
  - `make local-receive-test`
- Runtime evidence:
  - compose shows dedicated `reconciler` service with 2 running replicas.
  - worker logs show distinct `worker_id` values per replica.
  - cycle logs show claim counters (`claimed=...`) and no concurrent duplicate transition errors.
  - smoke flow confirms ETH/USDT requests reach `confirmed` under multi-replica reconciler runtime.
- Result: all checks passed.
