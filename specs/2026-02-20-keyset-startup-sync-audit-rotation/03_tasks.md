---
doc: 03_tasks
spec_date: 2026-02-20
slug: keyset-startup-sync-audit-rotation
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-xpub-index0-address-verification
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
- Rationale: This change includes startup orchestration behavior, DB schema migration, security-sensitive config, and non-trivial failure handling.
- Upstream dependencies (`depends_on`):
  - `2026-02-20-xpub-index0-address-verification`
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`

## Milestones

- M1: Spec + config/port design finalized and linted to READY.
- M2: Startup sync/preflight implementation + audit migration delivered.
- M3: Tests/docs verification completed and spec promoted to DONE.

## Tasks (ordered)

1. T-001 - Extend config contract for legacy secret and preflight entries

   - Scope: update config parsing/types/tests for active+legacy HMAC secret and nested preflight entries.
   - Output: `Config` exposes parsed legacy secret list and structured keyset preflight entries.
   - Linked requirements: FR-001, FR-002, NFR-002, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/infrastructure/config`
     - [x] Expected result: tests cover valid/invalid legacy secret env and nested preflight parsing.
     - [x] Logs/metrics to check (if applicable): N/A

2. T-002 - Add startup sync step to application port/use case

   - Scope: extend persistence bootstrap gateway port and initialize-persistence orchestration order.
   - Output: use case executes readiness -> migrations -> sync/preflight -> catalog validation.
   - Linked requirements: FR-006, NFR-001, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/application/use_cases`
     - [x] Expected result: use-case tests assert new sync invocation and failure propagation.
     - [x] Logs/metrics to check (if applicable): N/A

3. T-003 - Implement adapter-level preflight and wallet-account sync with legacy fallback

   - Scope: bootstrap gateway SQL + decision logic for `reused/reactivated/rotated` and active-hash rewrite.
   - Output: deterministic sync implementation with fail-fast app errors.
   - Linked requirements: FR-002, FR-003, FR-004, FR-006, NFR-001, NFR-003, NFR-004, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/adapters/outbound/persistence/postgresql/bootstrap`
     - [x] Expected result: branch coverage for action selection and mismatch failures.
     - [x] Logs/metrics to check (if applicable): startup logs include action summary and hash prefix.

4. T-004 - Add migration and write audit events

   - Scope: add new migration for `wallet_account_sync_events` and integrate event inserts in sync flow.
   - Output: queryable audit table with indexes and insert path for each sync action.
   - Linked requirements: FR-005, NFR-003, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): run integration/local startup then query `app.wallet_account_sync_events`.
     - [x] Expected result: one event per processed keyset with valid action/source/hash fields.
     - [x] Logs/metrics to check (if applicable): startup log and DB row action alignment.

5. T-005 - Update docs, compose/env, and local script compatibility

   - Scope: README/env examples include legacy secret config and app-level startup check behavior; script compatibility with legacy secret env.
   - Output: consistent operator guidance and local behavior.
   - Linked requirements: FR-001, FR-002, FR-003, FR-004, NFR-002, NFR-003
   - Validation:
     - [x] How to verify (manual steps or command): `make service-up` with valid/invalid keysets and rotated secrets.
     - [x] Expected result: invalid preflight blocks app startup; legacy secret rotation keeps wallet_account_id stable.
     - [x] Logs/metrics to check (if applicable): startup failure logs and sync action logs.

6. T-006 - End-to-end verification and spec closure
   - Scope: run repo verification workflow and update spec status to DONE.
   - Output: passing checks and traceable evidence in spec docs.
   - Linked requirements: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, NFR-001, NFR-002, NFR-003, NFR-004, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `SPEC_DIR=specs/2026-02-20-keyset-startup-sync-audit-rotation bash scripts/spec-lint.sh && bash scripts/verify-agents.sh specs/2026-02-20-keyset-startup-sync-audit-rotation && go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: all commands pass.
     - [x] Logs/metrics to check (if applicable): no startup validation regressions in smoke logs.

## Traceability (optional)

- FR-001 -> T-001, T-005, T-006
- FR-002 -> T-001, T-003, T-005, T-006
- FR-003 -> T-003, T-005, T-006
- FR-004 -> T-003, T-005, T-006
- FR-005 -> T-004, T-006
- FR-006 -> T-002, T-003, T-006
- NFR-001 -> T-002, T-003, T-006
- NFR-002 -> T-001, T-005, T-006
- NFR-003 -> T-003, T-004, T-005, T-006
- NFR-004 -> T-003, T-006
- NFR-005 -> T-001, T-002, T-003, T-004, T-006

## Rollout and rollback

- Feature flag: none.
- Migration sequencing: apply migration before startup sync executes (already guaranteed by startup order).
- Rollback steps: revert app build + run migration down for `000004` if required; restore prior env contract.
