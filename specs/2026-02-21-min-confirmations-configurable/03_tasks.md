---
doc: 03_tasks
spec_date: 2026-02-21
slug: min-confirmations-configurable
mode: Quick
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
  - 2026-02-20-chainobserver-solid-refactor
  - 2026-02-20-detected-threshold-configurable
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: null
  tasks: 03_tasks.md
  test_plan: null
---

# Task Plan

## Mode decision

- Selected mode: Quick
- Rationale: changes are confined to config parsing, observer adapter logic, and docs; no schema migration or new integration.
- Upstream dependencies (`depends_on`):
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
  - 2026-02-20-chainobserver-solid-refactor
  - 2026-02-20-detected-threshold-configurable
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`.
- If `02_design.md` is skipped (Quick mode):
  - Why it is safe to skip: architecture and port contracts remain unchanged.
  - What would trigger switching to Full mode: per-network policy matrix, reorg handling pipeline, or schema changes.
- If `04_test_plan.md` is skipped:
  - Where validation is specified (must be in each task): explicit validation checklist under each task below.

## Milestones

- M1: confirmation-depth config surfaces implemented and validated.
- M2: BTC/EVM observer logic enforces minimum confirmations while keeping detected signal.
- M3: docs/deploy and verification evidence complete.

## Tasks (ordered)

1. T-001 - Add minimum-confirmations config and validation

   - Scope: extend config model/env parsing for BTC and EVM minimum confirmations; propagate through DI.
   - Output: new config fields/env constants, validation errors, and tests.
   - Linked requirements: FR-001 / NFR-002
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/infrastructure/config/...`
     - [x] Expected result: defaults are `1`, valid custom values load, non-positive/invalid values fail with config error.
     - [x] Logs/metrics to check (if applicable): startup error message points to invalid env key.

2. T-002 - Apply minimum-confirmations logic in BTC/EVM observers

   - Scope: update BTC observer confirmed-amount source for min confirmations; update EVM observer confirmed block-tag selection; keep detected semantics.
   - Output: observer logic + adapter tests for depth-gated confirmation behavior.
   - Linked requirements: FR-002 / FR-003 / NFR-001 / NFR-002
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/adapters/outbound/chainobserver/devtest/... ./internal/application/use_cases/...`
     - [x] Expected result: with min confirmations > 1, requests are `detected` before `confirmed` until required depth is reached.
     - [x] Logs/metrics to check (if applicable): observation metadata includes effective min confirmation settings.

3. T-003 - Expose envs in deployment/docs and run end-to-end checks
   - Scope: wire new envs in compose/Makefile/README; run repo verification and local smoke checks.
   - Output: updated operator documentation and execution evidence.
   - Linked requirements: FR-004 / FR-002 / FR-003 / NFR-002
   - Validation:
     - [x] How to verify (manual steps or command): `go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: verification passes and service boots with default values.
     - [x] Logs/metrics to check (if applicable): local chain run shows depth-gated `confirmed` transitions for BTC/ETH/USDT.

## Traceability (optional)

- FR-001 -> T-001
- FR-002 -> T-002, T-003
- FR-003 -> T-002, T-003
- FR-004 -> T-003

## Validation evidence

- Unit/integration checks:
  - `go test -short ./internal/infrastructure/config/...` passed.
  - `go test -short ./internal/adapters/outbound/chainobserver/devtest/...` passed.
  - `go test -short ./...` passed.
  - `go vet ./...` passed.
- Local smoke checks (live chains + live service):
  - `bash scripts/local-chains/smoke_service_receive_all.sh` passed and produced `deployments/local-chains/artifacts/service-receive-local-all.json`.
  - Reconciler configured with depth gates:
    - `PAYMENT_REQUEST_RECONCILER_BTC_MIN_CONFIRMATIONS=2`
    - `PAYMENT_REQUEST_RECONCILER_EVM_MIN_CONFIRMATIONS=3`
  - Observed status progression:
    - initial after payment broadcast/mining: BTC/ETH/USDT all `detected`
    - after extra mining (`BTC +1 block`, `EVM +2 blocks`) and next reconcile cycle: BTC/ETH/USDT all `confirmed`
- NFR-001 -> T-002
- NFR-002 -> T-001, T-003
- NFR-005 -> T-002
- NFR-006 -> T-001, T-002

## Rollout and rollback

- Feature flag: none; behavior controlled by env defaults.
- Migration sequencing: no DB migration.
- Rollback steps: set min confirmations to `1` and redeploy previous image if needed.
