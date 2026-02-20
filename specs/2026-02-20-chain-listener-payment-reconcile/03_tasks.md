---
doc: 03_tasks
spec_date: 2026-02-20
slug: chain-listener-payment-reconcile
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-keyset-startup-sync-audit-rotation
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
- Rationale: includes runtime worker behavior, DB migration/index updates, and external chain integrations.
- Upstream dependencies (`depends_on`):
  - `2026-02-20-keyset-startup-sync-audit-rotation`
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`

## Milestones

- M1: Spec and config/runtime contracts finalized.
- M2: Reconciliation use case + persistence adapter implemented.
- M3: Chain observers + worker loop integrated and verified.

## Tasks (ordered)

1. T-001 - Add reconciler config model and validation

   - Scope: extend `Config` with enable/poll/batch/observer endpoint fields and parser tests.
   - Output: validated config surface for runtime worker.
   - Linked requirements: FR-001, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/infrastructure/config`
     - [x] Expected result: valid and invalid env scenarios covered.
     - [x] Logs/metrics to check (if applicable): N/A

2. T-002 - Add reconciliation ports/use case and unit tests

   - Scope: create inbound/outbound contracts and implement status transition orchestration.
   - Output: `ReconcilePaymentRequestsUseCase` with deterministic counters and CAS transitions.
   - Linked requirements: FR-002, FR-003, FR-006, NFR-001, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/application/use_cases`
     - [x] Expected result: unit tests cover expiry, detected, confirmed, skipped, idempotent behavior.
     - [x] Logs/metrics to check (if applicable): N/A

3. T-003 - Extend PostgreSQL payment request adapter for reconciliation

   - Scope: query open requests and CAS status updates with metadata merge.
   - Output: adapter methods implementing reconciliation repository port.
   - Linked requirements: FR-002, FR-003, FR-006, NFR-001, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/adapters/outbound/persistence/postgresql/paymentrequest`
     - [x] Expected result: repository logic passes, no create-path regressions.
     - [x] Logs/metrics to check (if applicable): N/A

4. T-004 - Add migration for status/index support

   - Scope: add `000005` migration for status constraint and polling index.
   - Output: migration up/down SQL and startup compatibility.
   - Linked requirements: FR-002, FR-006, NFR-002
   - Validation:
     - [x] How to verify (manual steps or command): run migration via startup/integration tests.
     - [x] Expected result: migration applies cleanly on existing schema.
     - [x] Logs/metrics to check (if applicable): startup migration logs.

5. T-005 - Implement chain observer gateway (BTC Esplora + EVM RPC)

   - Scope: outbound gateway for chain amount observations per request.
   - Output: observer adapter with timeout, parsing, and unsupported-network skip behavior.
   - Linked requirements: FR-004, FR-005, NFR-002, NFR-003, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): adapter unit tests with mocked HTTP responses.
     - [x] Expected result: correct detected/confirmed amount calculations.
     - [x] Logs/metrics to check (if applicable): no sensitive data in logs.

6. T-006 - Wire runtime worker and lifecycle logs

   - Scope: DI and server runtime goroutine for periodic reconciliation.
   - Output: worker starts only when enabled and exits gracefully on shutdown.
   - Linked requirements: FR-007, NFR-004, NFR-001
   - Validation:
     - [x] How to verify (manual steps or command): run app with reconciler enabled and inspect logs.
     - [x] Expected result: periodic cycle summary logs and graceful stop on shutdown.
     - [x] Logs/metrics to check (if applicable): cycle counters and transition logs.

7. T-007 - Local verification and spec closure
   - Scope: end-to-end smoke checks, docs alignment, and spec status promotion.
   - Output: validated behavior evidence for pending->confirmed and expiry transitions.
   - Linked requirements: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, NFR-001, NFR-002, NFR-003, NFR-004, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `SPEC_DIR=specs/2026-02-20-chain-listener-payment-reconcile bash scripts/spec-lint.sh && bash scripts/verify-agents.sh specs/2026-02-20-chain-listener-payment-reconcile && go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: all checks pass; runtime logs show reconciliation cycles when enabled.
     - [x] Logs/metrics to check (if applicable): transition and cycle summaries present.

## Traceability (optional)

- FR-001 -> T-001, T-006, T-007
- FR-002 -> T-002, T-003, T-004, T-007
- FR-003 -> T-002, T-003, T-007
- FR-004 -> T-005, T-007
- FR-005 -> T-005, T-007
- FR-006 -> T-002, T-003, T-004, T-007
- FR-007 -> T-006, T-007
- NFR-001 -> T-002, T-003, T-006, T-007
- NFR-002 -> T-004, T-005, T-007
- NFR-003 -> T-005, T-007
- NFR-004 -> T-006, T-007
- NFR-005 -> T-001, T-002, T-003, T-005, T-007

## Rollout and rollback

- Feature flag: `PAYMENT_REQUEST_RECONCILER_ENABLED`.
- Migration sequencing: apply migration before enabling worker in production-like runtime.
- Rollback steps: disable worker, deploy previous app version, roll back `000005` if required.

## Execution evidence

- Verification date: 2026-02-20
- Commands executed:
  - `go fmt ./...`
  - `go mod tidy`
  - `go list ./...`
  - `go test -short ./...`
  - `go vet ./...`
  - `SPEC_DIR=specs/2026-02-20-chain-listener-payment-reconcile bash scripts/spec-lint.sh`
  - `bash scripts/verify-agents.sh specs/2026-02-20-chain-listener-payment-reconcile`
  - `SERVICE_RECONCILER_ENABLED=true SERVICE_RECONCILER_POLL_INTERVAL_SECONDS=2 SERVICE_EVM_RPC_URLS_JSON='{"local":"http://host.docker.internal:8545"}' make local-up-all`
  - `make local-receive-test`
- Runtime evidence:
  - API check: ETH and USDT requests transitioned to `confirmed`.
  - API check: BTC request remained `pending` when BTC observer endpoint was not configured.
  - App logs include periodic cycle summaries (`scanned/confirmed/detected/expired/skipped/errors`).
- Result: all checks passed.
