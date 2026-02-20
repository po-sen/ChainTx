---
doc: 04_test_plan
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

# Test Plan

## Scope

- Covered:
  - Input validation and typed failure mapping for verification flow.
  - Deterministic index `0` derivation comparison for Bitcoin and EVM paths.
  - Key-material hash computation and persistence behavior.
  - Wallet account reuse/rotation/reactivation by hash change with cursor continuity.
  - Exit-code and output contract for operator command.
  - Startup preflight gate behavior before app container startup.
- Not covered:
  - On-chain spend/sign ownership proof.
  - Batch verification for multiple derivation indexes.

## Tests

### Unit

- TC-001: Reject malformed extended public key
  - Linked requirements: FR-001 / NFR-005, NFR-006
  - Steps: Execute verifier with invalid serialized key (for example `not-a-valid-xpub`) and valid-looking expected address.
  - Expected: Response returns `invalid_key_material_format`, `match=false`, non-zero exit code.
- TC-002: Reject non account-level extended public key
  - Linked requirements: FR-001 / NFR-005, NFR-006
  - Steps: Execute verifier with key fixture that parses but violates account-level policy (`depth != 3` or non-hardened account child number).
  - Expected: Response returns `invalid_configuration`, `match=false`, non-zero exit code.
- TC-003: Match known-good Bitcoin index-0 address
  - Linked requirements: FR-002 / NFR-001, NFR-002
  - Steps: Execute verifier with known-good BTC key fixture and expected `index=0` address.
  - Expected: Response `match=true`, expected equals derived, exit code `0`.
- TC-004: Match known-good Ethereum index-0 address
  - Linked requirements: FR-002 / NFR-001, NFR-002
  - Steps: Execute verifier with known-good EVM key fixture and expected `index=0` address.
  - Expected: Response `match=true`, expected equals derived, exit code `0`.
- TC-005: Hash determinism for identical configured input
  - Linked requirements: FR-003 / NFR-002, NFR-003
  - Steps: Compute hash from the same configured key-material string twice with the same `PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET`; then change key string or secret and recompute.
  - Expected: Same input + same secret yields identical hash; changed key string or changed secret yields different hash.
- TC-006: Keyset env parser backward compatibility
  - Linked requirements: FR-006 / NFR-002
  - Steps: Load config with legacy format (`{"ks":"xpub"}`), object legacy format (`{"ks":{"extended_public_key":"xpub"}}`), and nested preferred format (`{"ethereum":{"local":{"keyset_id":"ks","extended_public_key":"xpub"}}}`).
  - Expected: All produce identical in-memory keyset map.

### Integration

- TC-101: Command contract end-to-end with user-provided inputs
  - Linked requirements: FR-005 / NFR-002, NFR-005
  - Steps: Run documented command with runtime input values and capture stdout/stderr + exit status.
  - Expected: Output contains clear status field and human-readable summary; exit status follows match/fail contract.
- TC-102: Address mismatch reporting path
  - Linked requirements: FR-002, FR-005 / NFR-005
  - Steps: Run verifier with valid xpub and intentionally wrong expected address.
  - Expected: Output returns `address_mismatch`, includes expected + derived address, non-zero exit status.
- TC-103: Reuse existing wallet account when hash unchanged
  - Linked requirements: FR-004 / NFR-002, NFR-007
  - Steps: Run startup sync twice with identical key material for same `(chain, network, keyset_id)`.
  - Expected: Same active `wallet_account_id` remains; no extra row created.
- TC-104: Rotate wallet account when hash changes
  - Linked requirements: FR-004 / NFR-002, NFR-007
  - Steps: Run sync with key A, then update keyset to key B and rerun.
  - Expected: New active `wallet_account_id` is created, old account becomes inactive, new account starts at `next_index=0`.
- TC-110: Reactivate historical wallet account when hash reverts
  - Linked requirements: FR-004 / NFR-007, NFR-009
  - Steps: Run sync with key A, then key B, then key A again for the same tuple; capture `wallet_accounts` rows and `next_index`.
  - Expected: Third run reactivates original key-A wallet row (same id as first run) and preserves its previous `next_index`.
- TC-105: DB does not store raw xpub
  - Linked requirements: FR-003 / NFR-003
  - Steps: After sync, inspect `wallet_accounts` and related tables/columns.
  - Expected: Only hash metadata is persisted; raw xpub/tpub/vpub string is absent.
- TC-106: Legacy compatibility/backfill
  - Linked requirements: FR-006 / NFR-007
  - Steps: Start from pre-migration fixture and run migration + startup sync.
  - Expected: Active rows gain valid hash and remain allocatable without manual DB edits.
- TC-107: Missing HMAC secret guard
  - Linked requirements: FR-003, FR-006 / NFR-003
  - Steps: Execute hash/rotation flow without `PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET`.
  - Expected: Startup/sync fails fast with explicit configuration error.
- TC-108: Preflight blocks startup on mismatch
  - Linked requirements: FR-007 / NFR-008
  - Steps: Set one keyset `expected_index0_address` to an incorrect value, run `make service-up`.
  - Expected: Preflight exits non-zero before app startup; service-up fails.
- TC-109: Preflight allows startup when all keysets match
  - Linked requirements: FR-007 / NFR-008
  - Steps: Configure valid expected index-0 addresses for all keysets, run `make service-up`.
  - Expected: Preflight passes for each keyset and startup continues to app + catalog sync.

### E2E (if applicable)

- Scenario 1: Operator receives xpub + expected index-0 address from wallet owner, runs verifier command, and gets deterministic pass/fail.
- Scenario 2: Operator rotates keyset xpub; startup sync creates new wallet account for unseen hash and reactivates historical wallet account for known hash without raw xpub persistence.

## Edge cases and failure modes

- Case: xpub is valid format but chain/scheme pairing is wrong (for example BTC key with EVM scheme).
- Expected behavior: verifier fails with typed configuration error before final comparison.

- Case: EVM expected address has different hex case but same address bytes.
- Expected behavior: comparison treats case-insensitive hex as match.

- Case: Expected BTC bech32 address has uppercase letters.
- Expected behavior: verifier normalizes and compares in canonical lowercase form.

- Case: Active row has missing hash but keyset material exists in config.
- Expected behavior: startup sync backfills hash on the active row and proceeds without creating duplicate active accounts.

- Case: Keyset hash reverts to value used before.
- Expected behavior: system reactivates historical matching-hash row instead of inserting a new row, avoiding index reuse.

- Case: Keyset entry exists but `expected_index0_address` is missing.
- Expected behavior: preflight treats entry as invalid and fails startup with explicit tuple metadata.

## NFR verification

- Performance: single check should finish under 1 second on local machine.
- Reliability: repeated runs with identical input/hash state must return identical match and rotation outcomes.
- Security: command interface must not accept private key material; DB persists only hash for key identity.
- Deployment safety: startup preflight must run before app container startup and fail fast on mismatch.
