---
doc: 04_test_plan
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

# Test Plan

## Scope

- Covered:
  - Config parse/validation for active + legacy HMAC secret env.
  - App startup preflight and sync failure behavior.
  - Wallet-account decision branches (`reused/reactivated/rotated`) and legacy-hash upgrade.
  - Audit event persistence shape and indexes.
- Not covered:
  - New external API surface (none planned in this scope).

## Tests

### Unit

- TC-001: Config parses valid legacy secret JSON array.

  - Linked requirements: FR-001, NFR-005
  - Steps: set env with valid `PAYMENT_REQUEST_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON` and call `LoadConfig`.
  - Expected: config loads; legacy secret list populated and trimmed.

- TC-002: Config rejects invalid legacy secret JSON.

  - Linked requirements: FR-001, NFR-005
  - Steps: set invalid JSON value and call `LoadConfig`.
  - Expected: config error with explicit invalid legacy-secret code.

- TC-003: Startup use case executes new sync step.

  - Linked requirements: FR-006, NFR-001, NFR-005
  - Steps: run initialize-persistence use-case unit tests with fake gateway tracking method calls.
  - Expected: call order includes sync between migration and catalog validation.

- TC-004: Decision engine branches for reused/reactivated/rotated.

  - Linked requirements: FR-003, FR-004, NFR-001, NFR-005
  - Steps: run bootstrap unit tests that feed row/hash combinations.
  - Expected: expected action and selected wallet account ID for each branch.

- TC-005: Preflight mismatch fails.
  - Linked requirements: FR-002, FR-006, NFR-002, NFR-005
  - Steps: run bootstrap unit with mismatched expected index-0 address.
  - Expected: startup error with mismatch code.

### Integration

- TC-101: Startup sync writes audit events.

  - Linked requirements: FR-005, NFR-003
  - Steps: run bootstrap integration path against test DB, then query `app.wallet_account_sync_events`.
  - Expected: rows exist for processed keysets; required columns populated and indexed query succeeds.

- TC-102: Legacy secret rotation continuity.
  - Linked requirements: FR-003, FR-004, NFR-001
  - Steps: run startup with secret A, then secret B + legacy=[A] using same key material.
  - Expected: wallet_account_id unchanged; hash updated to secret-B digest; action not `rotated`.

### E2E (if applicable)

- TC-201: App startup blocks on invalid keyset preflight.

  - Linked requirements: FR-002, FR-006
  - Steps: set wrong `expected_index0_address`, start service.
  - Expected: startup exits non-zero before serving HTTP.

- TC-202: Local receive flow still passes after default config.
  - Linked requirements: FR-003, FR-006, NFR-004
  - Steps: restore defaults, run local stack smoke.
  - Expected: receive smoke passes and startup logs show sync actions.

## Edge cases and failure modes

- Case: Active row exists with null hash from historical state.
- Expected behavior: treated as `unhashed` reuse, hash backfilled with active digest.

- Case: Legacy secret list contains duplicates/blank entries.
- Expected behavior: blanks ignored, duplicates de-duplicated, startup continues.

- Case: Config contains keyset without nested preflight fields.
- Expected behavior: startup fails in devtest with explicit preflight/config error.

## NFR verification

- Performance: measure startup added time with 4 default keysets; ensure well under 5 seconds locally.
- Reliability: repeated startup with unchanged config keeps same active wallet account IDs.
- Security: verify DB has only hash digest and audit metadata, no raw xpub or raw secret values.
