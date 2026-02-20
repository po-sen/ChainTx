---
doc: 03_tasks
spec_date: 2026-02-20
slug: detected-threshold-configurable
mode: Quick
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
  - 2026-02-20-chainobserver-solid-refactor
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
- Rationale: change is confined to existing config loading, observer logic, and docs; no schema migration or new integration.
- Upstream dependencies (`depends_on`):
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
  - 2026-02-20-chainobserver-solid-refactor
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`
- If `02_design.md` is skipped (Quick mode):
  - Why it is safe to skip: no new runtime topology, data model, or cross-service contract.
  - What would trigger switching to Full mode: per-chain threshold policy, new statuses, or asynchronous notification pipeline changes.
- If `04_test_plan.md` is skipped:
  - Where validation is specified (must be in each task): explicit validation checklist under each `T-XXX` below.

## Milestones

- M1: config and observer threshold support implemented with unit coverage.
- M2: runtime smoke flow proves ETH/USDT partial payment transitions to `detected` then `confirmed`.

## Tasks (ordered)

1. T-001 - Add configurable threshold config surface
   - Scope: extend config model/env parsing/validation and wiring for detected+confirmed thresholds in bps.
   - Output: updated config structs, defaults, validation tests, and DI propagation.
   - Linked requirements: FR-003 / NFR-002 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/infrastructure/config/...`
     - [x] Expected result: tests cover valid defaults and invalid boundary values; invalid config returns error.
     - [x] Logs/metrics to check (if applicable): startup error includes env key and expected range.
2. T-002 - Apply thresholds consistently in BTC/ETH/USDT observer logic
   - Scope: unify threshold evaluation in chain observer so EVM can emit detected and BTC uses configured values.
   - Output: observer logic update + adapter tests for partial and full payments.
   - Linked requirements: FR-001 / FR-002 / NFR-001 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/adapters/outbound/chainobserver/devtest/... ./internal/application/use_cases/...`
     - [x] Expected result: BTC+ETH+USDT partial -> `Detected=true`, full -> `Confirmed=true`.
     - [x] Logs/metrics to check (if applicable): observation metadata includes observed amount and threshold fields.
3. T-003 - Update deploy/docs and run end-to-end verification
   - Scope: expose envs in compose/README, run verification workflow and local payment flow checks.
   - Output: docs/config updates plus validation evidence.
   - Linked requirements: FR-004 / FR-001 / FR-002 / NFR-002
   - Validation:
     - [x] How to verify (manual steps or command): `go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: full verification passes and service starts with defaults.
     - [x] Logs/metrics to check (if applicable): reconciler transitions `pending -> detected -> confirmed` for partial then completed payment.

## Validation evidence

- `go test -short ./internal/infrastructure/config/...` passed.
- `go test -short ./internal/adapters/outbound/chainobserver/devtest/...` passed.
- `go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...` passed.
- Runtime smoke (with `PAYMENT_REQUEST_RECONCILER_DETECTED_THRESHOLD_BPS=8000`, `PAYMENT_REQUEST_RECONCILER_CONFIRMED_THRESHOLD_BPS=10000`) passed:
  - BTC `pr_a4a081d025df551a8ee803d0`: partial -> `detected`, full -> `confirmed`
  - ETH `pr_6e9ba60d2f1ae78d5eb16d9c`: partial -> `detected`, full -> `confirmed`
  - USDT `pr_1da9df3b4af7a0e1bc60fd72`: partial -> `detected`, full -> `confirmed`

## Traceability (optional)

- FR-001 -> T-002, T-003
- FR-002 -> T-002, T-003
- FR-003 -> T-001
- FR-004 -> T-003
- NFR-001 -> T-002
- NFR-002 -> T-001, T-003
- NFR-005 -> T-002
- NFR-006 -> T-001, T-002

## Rollout and rollback

- Feature flag: none; controlled by threshold env defaults.
- Migration sequencing: no DB migration.
- Rollback steps: set thresholds back to `10000/10000` and redeploy previous image if needed.
