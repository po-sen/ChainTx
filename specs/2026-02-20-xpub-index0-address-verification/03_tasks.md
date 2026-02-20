---
doc: 03_tasks
spec_date: 2026-02-20
slug: xpub-index0-address-verification
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-10-wallet-allocation-asset-catalog
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
- Rationale: Scope now includes DB schema/index changes, startup rotation behavior, and compatibility/backfill handling in addition to verifier flow.
- Upstream dependencies (`depends_on`):
  - `2026-02-10-wallet-allocation-asset-catalog`
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`
- If `02_design.md` is skipped (Quick mode):
  - Not applicable; this spec is Full mode.
- If `04_test_plan.md` is skipped:
  - Not skipped in this spec; `04_test_plan.md` is included.

## Milestones

- M1: Hash-based key identity model and migration are defined and implemented.
- M2: Wallet account auto-rotation flow is implemented and idempotent.
- M3: Operator verifier flow validates index-0 address match.
- M4: Tests and docs cover rotation, hash privacy, and verification behavior.
- M5: Deployment preflight gate blocks service startup when keyset verification fails.

## Tasks (ordered)

1. T-001 - Add wallet account key-hash schema support

   - Scope: Introduce wallet account hash columns and active-row uniqueness model required for rotation.
   - Output: SQL migration updating `wallet_accounts` schema/index constraints.
   - Linked requirements: FR-003, FR-004, FR-006, NFR-007
   - Validation:
     - [x] How to verify (manual steps or command): Run migrations on clean and existing DB fixtures.
     - [x] Expected result: `wallet_accounts` includes hash fields; only one active account allowed per `(chain,network,keyset_id)`.
     - [x] Logs/metrics to check (if applicable):

2. T-002 - Implement key-hash computation in startup sync

   - Scope: Compute deterministic `hmac-sha256` from configured key-material string using required env secret, without persisting raw key material.
   - Output: Hash compute function and guardrails in `service_sync_catalog.sh` plus config requirement checks.
   - Linked requirements: FR-003, NFR-002, NFR-003
   - Validation:
     - [x] How to verify (manual steps or command): Run sync path with same key twice, then with changed key and/or secret.
     - [x] Expected result: Same key + same secret yields same hash; changed key or changed secret yields different hash; missing secret fails fast.
     - [x] Logs/metrics to check (if applicable): Rotation log includes hash prefix only.

3. T-003 - Implement startup sync reuse/rotate decision by hash

   - Scope: In catalog sync/bootstrap flow, reuse active account on same hash and rotate to new wallet account id on hash change.
   - Output: Updated sync logic with transactional account rotation + catalog remap.
   - Linked requirements: FR-004, FR-006, NFR-002, NFR-007
   - Validation:
     - [x] How to verify (manual steps or command): Run sync twice with same key, then with changed key.
     - [x] Expected result: Same key => same active `wallet_account_id`; unseen hash => new active `wallet_account_id` with `next_index=0`; reverted-to-known hash => historical row is reactivated with preserved `next_index`.
     - [x] Logs/metrics to check (if applicable): `action=reused|reactivated|rotated` appears with tuple identifiers.

4. T-004 - Enforce no raw xpub persistence in DB write paths

   - Scope: Ensure migrations/scripts/repositories never write raw key material into DB columns or SQL literals.
   - Output: Code path review + tests/guards that only hash is stored.
   - Linked requirements: FR-003, NFR-003
   - Validation:
     - [x] How to verify (manual steps or command): Inspect DB rows and logs after sync using sample keyset.
     - [x] Expected result: DB contains hash fields only; raw xpub is absent.
     - [x] Logs/metrics to check (if applicable): Key values are masked/hash-prefix only.

5. T-005 - Define verification command contract and implement index-0 matcher

   - Scope: Keep the operator verification command with typed input validation and deterministic comparison.
   - Output: Verifier entrypoint + result schema + exit-code contract.
   - Linked requirements: FR-001, FR-002, FR-005, NFR-001, NFR-005, NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): Run verifier with known-good BTC/EVM fixtures and mismatch fixture.
     - [x] Expected result: Match returns exit `0`; mismatch/invalid input returns non-zero with typed reason.
     - [x] Logs/metrics to check (if applicable): Output contains `match`, `derived_address`, `expected_address`.

6. T-006 - Add integration tests for migration, rotation, and compatibility

   - Scope: Cover legacy-row backfill, active-row uniqueness, and rotation idempotency.
   - Output: Integration test cases under bootstrap/persistence packages.
   - Linked requirements: FR-004, FR-006, NFR-002, NFR-007
   - Validation:
     - [x] How to verify (manual steps or command): Run targeted integration suites.
     - [x] Expected result: Tests prove one-active-account invariant and correct rotation behavior.
     - [x] Logs/metrics to check (if applicable): Test assertions include action/reason checks.

7. T-007 - Update README/runbook for operator usage and rotation semantics

   - Scope: Document how wallet account ids rotate by hash, required HMAC env, and how to run verification command.
   - Output: README/runbook examples and troubleshooting notes.
   - Linked requirements: FR-005, FR-006, NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): Follow docs to run one reuse case and one rotate case.
     - [x] Expected result: Operator can predict behavior and confirm from DB/query outputs.
     - [x] Logs/metrics to check (if applicable): Documented log fields match implementation.

8. T-008 - Add startup preflight verifier script and wire into `service-up`

   - Scope: Add a script that iterates keysets, runs `cmd/keysetverify`, and fails on first mismatch/error.
   - Output: `scripts/local-chains/service_verify_keysets.sh` and `Makefile` integration before app startup.
   - Linked requirements: FR-007, NFR-008
   - Validation:
     - [x] How to verify (manual steps or command): Run `make service-up` with one bad expected address.
     - [x] Expected result: `service-up` exits non-zero and app container is not started.
     - [x] Logs/metrics to check (if applicable): Failure log shows tuple + verifier reason.

9. T-009 - Extend keyset JSON contract for startup preflight

   - Scope: Add `expected_index0_address` fields in generated keyset JSON and document required inputs/env.
   - Output: Updated `service_keysets_json.sh`, compose defaults, and README examples.
   - Linked requirements: FR-007, NFR-005, NFR-008
   - Validation:
     - [x] How to verify (manual steps or command): Print generated keysets JSON and inspect fields.
     - [x] Expected result: Each configured keyset contains expected index-0 address used by preflight.
     - [x] Logs/metrics to check (if applicable): Preflight success logs list checked keysets.

10. T-010 - Reactivate historical row for known hash to preserve cursor

- Scope: In hash-change path, detect tuple-matching historical row with same hash and reactivate it instead of creating a new row.
- Output: Updated `service_sync_catalog.sh` decision flow with `reactivated` action and preserved `next_index`.
- Linked requirements: FR-004, NFR-007, NFR-009
- Validation:
  - [x] How to verify (manual steps or command): Change key A->B->A and inspect `wallet_accounts`.
  - [x] Expected result: Final switch back to A reactivates original A row id and retains its prior `next_index`.
  - [x] Logs/metrics to check (if applicable): `action=reactivated` for hash reversion path.

## Traceability (optional)

- FR-001 -> T-005
- FR-002 -> T-005
- FR-003 -> T-001, T-002, T-004
- FR-004 -> T-001, T-003, T-006, T-010
- FR-005 -> T-005, T-007
- FR-006 -> T-001, T-003, T-006, T-007
- FR-007 -> T-008, T-009
- NFR-001 -> T-005
- NFR-002 -> T-002, T-003, T-006
- NFR-003 -> T-004
- NFR-005 -> T-005, T-007
- NFR-006 -> T-002, T-005
- NFR-007 -> T-001, T-003, T-006, T-010
- NFR-009 -> T-010
- NFR-008 -> T-008, T-009

## Rollout and rollback

- Feature flag: none.
- Migration sequencing: apply schema migration before deploying updated sync logic.
- Rollback steps: rollback migration and restore previous sync behavior; keep existing active wallet account mapping frozen.
