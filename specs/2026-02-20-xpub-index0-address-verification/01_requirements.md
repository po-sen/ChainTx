---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- Verification input: `chain`, `network`, `address_scheme`, `keyset_id`, extended public key, and expected address for index `0`.
- Match result: deterministic comparison outcome between derived address and expected address.
- Typed key error: structured derivation/key parsing error mapped from `walletkeys` error model.
- Key material hash: deterministic digest generated from configured extended public key string + HMAC secret, for equality detection without storing raw xpub.
- Keyset env preferred shape: `PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON` nested by chain/network, for example:
  - `{ "bitcoin": { "regtest": { "keyset_id": "ks_btc_regtest", "extended_public_key": "..." } }, "ethereum": { "sepolia": { "keyset_id": "ks_eth_sepolia", "extended_public_key": "..." } } }`
- HMAC secret env: `PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET` used with `hmac-sha256` for key-material hash derivation.
- Startup preflight verification: deployment step that runs index-0 address checks for configured keysets before starting service containers.

## Out-of-scope behaviors

- OOS1: Any on-chain signing/spending transaction to prove private-key ownership.
- OOS2: Support for hardened derivation or non-standard templates in this feature.
- OOS3: Multi-index batch verification in the first iteration.

## Functional requirements

### FR-001 - Verification input must be validated before derivation

- Description: The verification flow must reject invalid or unsupported inputs before deriving index `0`.
- Acceptance criteria:
  - [x] AC1: Missing required input fields (`chain`, `network`, `address_scheme`, extended public key, expected address) return validation error.
  - [x] AC2: Unsupported chain/address-scheme combinations are rejected with a typed configuration error.
  - [x] AC3: Extended public key parsing failures return `invalid_key_material_format`.
  - [x] AC4: Non account-level extended public keys (`depth != 3` or non-hardened account child number) are rejected as invalid configuration.
  - [x] AC5: Derivation path policy is enforced as `0/{index}` with `index=0`.
- Notes: Reuse existing wallet key parsing/normalization and policy checks; do not duplicate cryptographic validation logic.

### FR-002 - System must compare derived index-0 address against expected address

- Description: For valid inputs, derive address at index `0` and compare with expected address deterministically.
- Acceptance criteria:
  - [x] AC1: Derived address for index `0` is computed with current chain-specific derivation logic.
  - [x] AC2: Comparison returns `match=true` only when derived address equals expected address under chain comparison rules.
  - [x] AC3: Result always includes both expected and derived address values for operator review.
  - [x] AC4: On mismatch, result includes explicit reason (`address_mismatch`) and non-zero process status for CLI mode.
- Notes: Bitcoin address comparison should use normalized lowercase bech32; EVM comparison should be case-insensitive hex match.

### FR-003 - DB persistence must store key material hash only

- Description: Database persistence for wallet-key identity must use hash/fingerprint only and must not store raw xpub.
- Acceptance criteria:
  - [x] AC1: Hash is generated from configured key-material string using `hmac-sha256`.
  - [x] AC2: Raw xpub/tpub/vpub string is never persisted in relational tables.
  - [x] AC3: Persisted hash is available for equality checks during startup/catalog sync and allocation readiness validation.
  - [x] AC4: Hash computation is deterministic for identical configured key-material input.
  - [x] AC5: Hash algorithm metadata persists as `hmac-sha256`.
- Notes: HMAC secret must come from env and must not be hardcoded in source.

### FR-004 - Wallet account identity must auto-rotate on key hash change without reusing addresses

- Description: For each `(chain, network, keyset_id)` context, system must rotate active wallet account on key hash change and preserve cursor continuity when reverting to a previously used hash.
- Acceptance criteria:
  - [x] AC1: If incoming key hash equals latest active wallet account hash for the same `(chain, network, keyset_id)`, existing wallet account is reused.
  - [x] AC2: If incoming key hash differs and no historical row exists for that hash, system creates a new wallet account row with new `wallet_account_id`, `next_index=0`, and marks previous row inactive.
  - [x] AC3: Asset-catalog mapping for the tuple is updated to the new wallet account in the same sync run.
  - [x] AC4: Rotation is idempotent; repeated sync with unchanged hash does not create duplicate wallet accounts.
  - [x] AC5: System emits structured log/metadata indicating whether action was `reused`, `reactivated`, or `rotated`.
  - [x] AC6: If incoming key hash matches a historical inactive row for the tuple, system reactivates that row and preserves its existing `next_index` (no new row).
- Notes: Wallet account id generation should be unique for unseen hashes and avoid collisions.

### FR-005 - Operator must have one reproducible execution flow for ad-hoc verification

- Description: Operators must be able to run one command with provided user values and get a clear pass/fail output.
- Acceptance criteria:
  - [x] AC1: Repository contains a documented command path (script or executable invocation) that accepts user-provided verification inputs without code edits.
  - [x] AC2: Successful match returns exit status `0`; mismatch/validation errors return non-zero.
  - [x] AC3: Output format includes machine-readable status field (`match=true|false` or equivalent) plus human-readable summary.
  - [x] AC4: Execution and output examples are documented for one Bitcoin and one EVM case.
- Notes: This requirement is for operator confidence before using a keyset in funding flows.

### FR-006 - Backfill and startup compatibility for pre-hash rows

- Description: Existing deployments with legacy wallet rows must remain operable after introducing key hash persistence and auto-rotation.
- Acceptance criteria:
  - [x] AC1: Migration/backfill populates hash field for existing rows without requiring raw xpub persistence in DB.
  - [x] AC2: Startup validation fails fast with explicit error if hash is required but cannot be resolved from configured keyset material.
  - [x] AC3: No import-cycle or architecture boundary violations are introduced while adding hash/rotation support.
  - [x] AC4: Configuration parser accepts both legacy keyset JSON formats (`{"id":"xpub"}` and `{"id":{"extended_public_key":"xpub"}}`) and preferred nested chain/network format.
- Notes: Backfill may use configured devtest keyset map as source of truth during startup sync.

### FR-007 - Deployment must auto-verify keysets before app startup

- Description: `service-up` flow must execute index-0 verification for configured keysets before starting app container.
- Acceptance criteria:
  - [x] AC1: Preflight reads `PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON` and verifies each configured keyset entry that defines `expected_index0_address`.
  - [x] AC2: On any mismatch or validation failure, preflight exits non-zero and app startup is aborted.
  - [x] AC3: On success, preflight prints per-keyset pass logs and allows startup to continue.
  - [x] AC4: Preferred nested keyset format supports `expected_index0_address` alongside `keyset_id` and `extended_public_key`.
  - [x] AC5: Failure output includes `chain`, `network`, `keyset_id`, and verifier reason/error code.
- Notes: This is a startup gate; do not silently skip failed checks.

## Non-functional requirements

- Performance (NFR-001): Single verification completes within 1 second on local developer environment.
- Availability/Reliability (NFR-002): Same input and same keyset hash state must always produce identical output, exit code, and rotation decision.
- Security/Privacy (NFR-003): Feature must not require or accept private key material (`xprv`), and must not persist raw xpub to storage.
- Compliance (NFR-004): Not applicable for this local/operator verification utility.
- Observability (NFR-005): Failures must include typed error code and actionable reason (`invalid_key_material_format`, `invalid_configuration`, `address_mismatch`, `wallet_account_rotation_failed`).
- Maintainability (NFR-006): Implementation must reuse `walletkeys` normalization + derivation functions rather than reimplementing BIP32 math.
- Data integrity (NFR-007): For each `(chain, network, keyset_id)` there is at most one active wallet account chosen by current hash.
- Deployment safety (NFR-008): Preflight verification completes before app startup and fails fast on first invalid/mismatch keyset.
- Data integrity (NFR-009): The same `(chain, network, keyset_id, key_material_hash)` must map back to a single cursor lineage (`next_index`) to prevent address index reuse after key reversion.

## Dependencies and integrations

- External systems: none required for deterministic local derivation check.
- Internal services: `internal/infrastructure/walletkeys` derivation primitives; existing error mapping conventions; PostgreSQL wallet account/catalog persistence flow.
