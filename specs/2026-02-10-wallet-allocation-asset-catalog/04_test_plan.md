---
doc: 04_test_plan
spec_date: 2026-02-10
slug: wallet-allocation-asset-catalog
mode: Full
status: READY
owners:
  - posen
depends_on:
  - 2026-02-09-multi-asset-payment-request-api
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
  - Schema and seed correctness for `wallet_accounts` + `asset_catalog`.
  - Startup validation gate behavior, including address-scheme allow-list checks.
  - Deterministic BTC/EVM derivation in dev/test adapter.
  - Transactional cursor allocation and rollback behavior.
  - Application-level canonicalization responsibility.
  - Concurrency safety for shared allocator scenarios (ETH+USDT).
- Not covered:
  - Production HSM/external wallet provider implementation.
  - Blockchain confirmations and webhook workflows.

## Tests

### Unit

- TC-001: Migration schema constraint verification

  - Linked requirements: FR-001
  - Steps: Assert generated schema contains required constraints (`next_index >= 0`, expiry bounds, EVM `chain_id` rule, `UNIQUE(chain, network, keyset_id)`, token/native metadata checks where feasible) using migration test harness.
  - Expected: Constraint set matches spec; migration rerun remains idempotent.

- TC-002: Startup validator rule matrix

  - Linked requirements: FR-002, FR-001, NFR-005
  - Steps: Feed validator with valid rows and invalid fixtures (missing wallet account, inactive account, out-of-range expiry, EVM missing `chain_id`, invalid `address_scheme`, token row missing token metadata, native row carrying token metadata, keyset metadata `depth != 3`, non-hardened `child_number`, derivation suffix violating `0/{index}` policy, derivation suffix containing hardened segment markers).
  - Expected: Valid set passes; policy-mismatch fixtures return `invalid_configuration` with deterministic detail codes.

- TC-003: BTC derivation determinism fixture test

  - Linked requirements: FR-003, FR-004
  - Steps: Run adapter with fixed BTC non-prod extended public key fixture (`tpub`/`vpub`) and suffix path/index repeatedly.
  - Expected: Address output is deterministic and network-correct for `regtest` and `testnet` cases, including `vpub` -> library-supported normalization when required.

- TC-004: EVM derivation + canonicalization fixture test

  - Linked requirements: FR-004, FR-005
  - Steps: Adapter derives EVM `address_raw`; application canonicalizer produces `address_canonical` (lowercase) and API `payment_instructions.address` (EIP-55).
  - Expected: Adapter and canonicalizer responsibilities are separated and deterministic, with explicit DB/API value mapping.

- TC-005: 64-bit derivation index handling

  - Linked requirements: FR-003, FR-005
  - Steps: Run create-flow unit/integration boundary tests with index values above 32-bit range.
  - Expected: No truncation occurs; index remains consistent with `BIGINT` value.

- TC-006: Dev/test mainnet guard behavior

  - Linked requirements: FR-007, FR-003, NFR-003
  - Steps: Execute derivation request targeting mainnet with guard off/on.
  - Expected: Default path rejects with `mainnet_allocation_blocked` and without cursor mutation; explicit override path allows derivation.

- TC-007: EVM key material format validation

  - Linked requirements: FR-004, FR-003
  - Steps: Start adapter with valid EVM `xpub` and `tpub` configs, then with invalid variants (`ypub`/`zpub`/`vpub`).
  - Expected: EVM accepts `xpub`/`tpub` and normalizes internally; invalid variants fail at startup/config validation.

- TC-008: Derivation-path suffix hardened-segment guard

  - Linked requirements: FR-003, FR-004
  - Steps: Configure wallet account with suffix templates like `0/{index}` (valid), `{index}` (invalid in this feature), and `0'/{index}` or `0h/{index}` (invalid).
  - Expected: Only `0/{index}` passes; non-policy/hardened suffix values are rejected before request serving.

- TC-009: Key normalization fail-closed behavior
  - Linked requirements: FR-002, FR-004
  - Steps: Run adapter/bootstrap with key material that cannot be safely normalized by selected runtime/library.
  - Expected: Service fails startup with typed `invalid_key_material_format`; no silent fallback path is used.

### Integration

- TC-101: Seed mapping correctness including shared allocator

  - Linked requirements: FR-001, FR-005, NFR-004
  - Steps: Apply migrations and query seed rows for BTC/ETH/USDT across configured non-prod networks.
  - Expected: ETH and USDT rows on same EVM network reference same `wallet_account_id`; rows are enabled only when valid.

- TC-102: Transaction rollback protects cursor

  - Linked requirements: FR-005, NFR-002
  - Steps: Inject failure after derivation but before commit, then query `wallet_accounts.next_index` and `payment_requests`.
  - Expected: No payment request persisted and `next_index` unchanged.

- TC-103: Single-create smoke by asset

  - Linked requirements: FR-006, FR-004
  - Steps: Create one request each for BTC, ETH, USDT.
  - Expected: Each response stores/returns valid address metadata and increments cursor exactly once.

- TC-104: Parallel create on same tuple (>=200)

  - Linked requirements: FR-006, FR-005, NFR-002
  - Steps: Fire >=200 concurrent creates for one tuple with unique idempotency keys.
  - Expected: Zero duplicate `(wallet_account_id, derivation_index)` and `(chain, network, address_canonical)` rows.

- TC-105: Mixed ETH+USDT shared allocator concurrency

  - Linked requirements: FR-006, FR-005, NFR-002
  - Steps: Fire mixed concurrent ETH and USDT creates against same EVM network allocator.
  - Expected: Single monotonic index stream without collisions across both assets.

- TC-106: Observability signal coverage

  - Linked requirements: NFR-005, NFR-001
  - Steps: Execute successful and failing allocation paths; inspect logs/metrics.
  - Expected: Required labels and latency metrics are emitted with stable keys.

- TC-107: Canonical-storage vs API-format mapping

  - Linked requirements: FR-005, FR-004
  - Steps: Create EVM request, inspect stored `payment_requests.address_canonical`, then fetch API payload.
  - Expected: DB stores canonical lowercase address; API returns checksummed `payment_instructions.address` for the same value.

- TC-108: Wallet account natural-key uniqueness
  - Linked requirements: FR-001
  - Steps: Attempt to insert duplicate `wallet_accounts` rows with same `(chain, network, keyset_id)`.
  - Expected: DB rejects duplicates deterministically; allocator cannot split across duplicate natural keys.

### E2E (if applicable)

- Scenario 1:

  - Linked requirements: FR-005, FR-006
  - Steps: Through API integration harness, create payment requests while adapter runs in dev/test mode.
  - Expected: End-to-end create flow consumes cursor correctly and returns usable instructions.

- Scenario 2:
  - Linked requirements: FR-007, NFR-003
  - Steps: Run local stack with default config and attempt mainnet-targeted request.
  - Expected: Request fails safely with explicit `mainnet_allocation_blocked` error and no persistence side effects.

## Edge cases and failure modes

- Case: Catalog row enabled but wallet account later marked inactive.
- Expected behavior: Startup validation fails on next boot and service does not start. (FR-002)

- Case: Adapter receives unsupported chain/network pair.
- Expected behavior: Typed `unsupported_allocator_target` error, no cursor advancement. (FR-003, FR-004)

- Case: Concurrent lock contention spikes.
- Expected behavior: Requests either serialize safely or fail with retriable transaction error; no duplicates. (FR-005, NFR-002)

- Case: Mainnet guard disabled intentionally for controlled test.
- Expected behavior: System logs warning with guard state and proceeds according to explicit config. (FR-007, NFR-003)

## NFR verification

- Performance:
  - Measure p95 allocation path latency under 20 RPS; target <= 300 ms (NFR-001).
- Reliability:
  - Run TC-104/TC-105; assert duplicate counters are exactly zero and rollback semantics hold (NFR-002).
- Security:
  - Verify logs contain no private key/seed values and mainnet guard is enforced by default (NFR-003).
- Compliance:
  - Confirm asset seed change records exist for enabled production-like rows (NFR-004).
- Observability:
  - Validate metrics/logs from TC-106 are queryable with required dimensions (NFR-005).
- Maintainability:
  - Run quality gates (`go fmt`, `go vet`, `go list`, `go test`) on impacted modules (NFR-006).
