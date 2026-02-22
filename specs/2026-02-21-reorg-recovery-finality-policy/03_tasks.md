---
doc: 03_tasks
spec_date: 2026-02-21
slug: reorg-recovery-finality-policy
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-detected-threshold-configurable
  - 2026-02-21-min-confirmations-configurable
  - 2026-02-20-reconciler-horizontal-scaling
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
- Rationale: requires DB migration, adapter contract expansion, async
  reconciler failure handling, and E2E reorg validation.
- Upstream dependencies (`depends_on`):
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-detected-threshold-configurable
  - 2026-02-21-min-confirmations-configurable
  - 2026-02-20-reconciler-horizontal-scaling
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`.
- If `02_design.md` is skipped (Quick mode): not applicable.
- If `04_test_plan.md` is skipped: not applicable.

## Milestones

- M1: schema + contracts for reversible reorg lifecycle.
- M2: reconciler decision engine + adapter evidence collection.
- M3: end-to-end reorg validation and operational documentation.

## Tasks (ordered)

1. T-001 - Status model and migration scaffolding

   - Scope: add `reorged` status support in domain value object and SQL constraint migration.
   - Output: updated status parser tests + migration scripts for status constraint.
   - Linked requirements: FR-001, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): run migration tests and `go test ./internal/domain/... ./internal/adapters/outbound/persistence/postgresql/...`.
     - [ ] Expected result: parser accepts `reorged`; migration is idempotent and constraints are valid.
     - [ ] Logs/metrics to check (if applicable): migration logs contain no constraint errors.

2. T-002 - Settlement evidence persistence

   - Scope: introduce `payment_request_settlements` schema and outbound
     repository methods for diff-based writes/query canonical totals.
   - Output: new migration + repository integration tests for change-only write/orphan handling.
   - Linked requirements: FR-002, NFR-002, NFR-006, NFR-007
   - Validation:
     - [ ] How to verify (manual steps or command): run DB integration tests for settlement repository operations.
     - [ ] Expected result: unchanged evidence causes no settlement row writes; canonical/orphan toggles behave deterministically.
     - [ ] Logs/metrics to check (if applicable): query/update logs include stable error codes on forced failures.

3. T-003 - Chain observer evidence contract

   - Scope: extend observer DTO/port to return installment-level tx evidence and
     canonical hints for BTC/ETH/ERC20.
   - Output: updated port contracts and adapter tests for BTC outpoints,
     ETH tx hashes, and ERC20 tx-hash/log-index evidence extraction.
   - Linked requirements: FR-002, FR-003, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): `go test ./internal/adapters/outbound/chainobserver/devtest/... ./internal/application/...`.
     - [ ] Expected result: observer outputs include per-installment evidence
           refs, confirmations, and canonical flags for supported rails without
           aggregate snapshot evidence refs.
     - [ ] Logs/metrics to check (if applicable): unsupported rails still return deterministic `Supported=false` path.

4. T-004 - Reconciler decision engine for rollback/recovery

   - Scope: implement business/finality thresholds, observe-window policy, and anti-flap stability counters.
   - Output: use-case logic updates and unit tests for `confirmed -> reorged -> confirmed` sequences.
   - Linked requirements: FR-001, FR-003, FR-004, FR-005, NFR-001, NFR-002
   - Validation:
     - [ ] How to verify (manual steps or command): run reconcile use-case tests covering promote/demote/orphan/expiry branches.
     - [ ] Expected result: transitions match policy; cycles remain idempotent under repeated inputs.
     - [ ] Logs/metrics to check (if applicable): cycle summary emits reorg-related counters.

5. T-005 - Config surface and startup validation

   - Scope: add env parsing/validation for business/finality confirmations, observe window, and stability cycles.
   - Output: config loader changes, defaults, and invalid-input tests.
   - Linked requirements: FR-003, FR-004, FR-005, NFR-002
   - Validation:
     - [ ] How to verify (manual steps or command): run config tests with valid/invalid env matrices.
     - [ ] Expected result: invalid values fail startup deterministically with typed config errors.
     - [ ] Logs/metrics to check (if applicable): startup error logs include env key and validation reason.

6. T-006 - Webhook payload extension

   - Scope: enrich `payment_request.status_changed` payload with transition reason and finality/evidence summary.
   - Output: payload builder updates and webhook dispatcher integration tests.
   - Linked requirements: FR-006, NFR-005
   - Validation:
     - [ ] How to verify (manual steps or command): run webhook outbox/dispatcher tests for transition payload snapshots.
     - [ ] Expected result: additive fields present; existing fields unchanged.
     - [ ] Logs/metrics to check (if applicable): dispatch logs continue to show delivery buckets with no schema errors.

7. T-007 - Reorg E2E smoke coverage (BTC + EVM)

   - Scope: add/extend local-chain smoke scripts to force reorg and verify
     rollback/recovery plus webhook emission, including split-installment
     visibility in settlement rows.
   - Output: deterministic smoke commands and captured verification steps in docs.
   - Linked requirements: FR-001, FR-002, FR-006, NFR-002, NFR-005, NFR-008
   - Validation:
     - [ ] How to verify (manual steps or command): run end-to-end smoke flow with forced reorg on BTC and EVM.
     - [ ] Expected result: status transitions include
           `confirmed -> reorged -> confirmed`; webhook payloads carry transition
           reason; split payments generate one settlement row per installment.
     - [ ] Logs/metrics to check (if applicable): reconciler cycle logs show rollback/recovery counters.

8. T-008 - Documentation and verification gate
   - Scope: update README/env documentation and run repository verification workflow.
   - Output: docs updated for new env semantics and runbook notes for observe window.
   - Linked requirements: FR-003, FR-004, FR-006, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): `go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`.
     - [ ] Expected result: all checks pass; docs describe reorg lifecycle and config defaults.
     - [ ] Logs/metrics to check (if applicable): no import-cycle or lint regression.

## Traceability (optional)

- FR-001 -> T-001, T-004, T-007
- FR-002 -> T-002, T-003, T-007
- FR-003 -> T-003, T-004, T-005, T-008
- FR-004 -> T-004, T-005, T-008
- FR-005 -> T-004, T-005
- FR-006 -> T-006, T-007, T-008
- NFR-008 -> T-003, T-007
- NFR-001 -> T-004
- NFR-002 -> T-002, T-004, T-005, T-007
- NFR-005 -> T-006, T-007
- NFR-006 -> T-001, T-002, T-003, T-008
- NFR-007 -> T-002

## Validation evidence

- `go test ./internal/adapters/outbound/chainobserver/devtest/...`
- `go test ./internal/application/use_cases/...`
- `go fmt ./...`
- `go mod tidy`
- `go list ./...`
- `go test -short ./...`
- `go vet ./...`
- `go test ./...`
- `SPEC_DIR="specs/2026-02-21-reorg-recovery-finality-policy" bash scripts/spec-lint.sh`
- `bash scripts/verify-agents.sh specs/2026-02-21-reorg-recovery-finality-policy`

## Rollout and rollback

- Feature flag: reuse reconciler enablement plus new config validation gates;
  rollout with conservative defaults.
- Migration sequencing:
  - apply status/evidence migrations first
  - deploy reconciler logic
  - enable new policy env overrides gradually
- Rollback steps:
  - disable reconciler if severe regression
  - roll back app binary
  - keep additive evidence table in place (no destructive rollback required for immediate service recovery).
