---
doc: 03_tasks
spec_date: 2026-02-20
slug: chainobserver-solid-refactor
mode: Quick
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
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
- Rationale: code-structure refactor only; no schema/API contract change.
- Upstream dependencies (`depends_on`):
  - `2026-02-20-chain-listener-payment-reconcile`
  - `2026-02-20-reconciler-horizontal-scaling`
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`
- If `02_design.md` is skipped (Quick mode):
  - Why it is safe to skip: responsibilities are refactored within one adapter package with unchanged external port.
  - What would trigger switching to Full mode: any runtime contract/schema change.
- If `04_test_plan.md` is skipped:
  - Where validation is specified (must be in each task): per-task validation below.

## Milestones

- M1: split observer responsibilities into focused files/components.
- M2: keep behavior parity and pass full short test suite.

## Tasks (ordered)

1. T-001 - Split chain observer coordinator and provider-specific observers

   - Scope: extract BTC/EVM observation logic out of monolithic gateway file.
   - Output: coordinator file + dedicated BTC/EVM observer files.
   - Linked requirements: FR-001, NFR-001
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/adapters/outbound/chainobserver/devtest`
     - [x] Expected result: adapter tests pass with same behavior.
     - [x] Logs/metrics to check (if applicable): N/A

2. T-002 - Extract reusable RPC helper and shared parsing utilities

   - Scope: move JSON-RPC transport concerns and parsing helpers to focused components.
   - Output: rpc helper file(s) used by EVM observer.
   - Linked requirements: FR-002, NFR-001, NFR-002
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/adapters/outbound/chainobserver/devtest`
     - [x] Expected result: EVM tests remain green.
     - [x] Logs/metrics to check (if applicable): N/A

3. T-003 - Regression verification and spec closure
   - Scope: run repo verification chain and finalize spec status.
   - Output: green verification evidence.
   - Linked requirements: FR-003, NFR-002, NFR-003
   - Validation:
     - [x] How to verify (manual steps or command): `SPEC_DIR=specs/2026-02-20-chainobserver-solid-refactor bash scripts/spec-lint.sh && go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: all checks pass and behavior remains compatible.
     - [x] Logs/metrics to check (if applicable): N/A

## Traceability (optional)

- FR-001 -> T-001
- FR-002 -> T-002
- FR-003 -> T-003
- NFR-001 -> T-001, T-002
- NFR-002 -> T-002, T-003
- NFR-003 -> T-003

## Rollout and rollback

- Feature flag: none.
- Migration sequencing: none.
- Rollback steps: revert adapter refactor commit if regression found.

## Execution evidence

- Verification date: 2026-02-20
- Commands executed:
  - `SPEC_DIR=specs/2026-02-20-chainobserver-solid-refactor bash scripts/spec-lint.sh`
  - `go fmt ./...`
  - `go mod tidy`
  - `go list ./...`
  - `go test -short ./...`
  - `go vet ./...`
- Result: all commands passed.
