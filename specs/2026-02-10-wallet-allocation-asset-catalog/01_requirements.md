---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- Wallet account: Allocation cursor entity that owns derivation state (`next_index`) for one keyset on one chain/network.
- Asset catalog: Server-controlled mapping from `(chain, network, asset)` to metadata and `wallet_account_id`.
- Shared allocator: Multiple assets mapped to the same `wallet_account_id` (for example ETH + USDT on Ethereum).
- Canonical address: Deterministic normalized address used for persistence uniqueness checks.
- Derivation material: Public-only derivation inputs (`keyset_id`, xpub, derivation path template) used to derive deposit addresses.

## Out-of-scope behaviors

- OOS1: Final production wallet provider (HSM/external wallet service) implementation.
- OOS2: On-chain event ingestion and payment status progression.
- OOS3: New consumer-facing endpoint design work.

## Functional requirements

### FR-001 - Persist wallet accounts and asset catalog with integrity constraints

- Description: Database must define `wallet_accounts` and `asset_catalog` with schema checks that enforce allocation correctness.
- Acceptance criteria:
  - [ ] AC1: Migration creates `wallet_accounts` with at least `id`, `chain`, `network`, `keyset_id`, `derivation_path_template`, `next_index BIGINT`, `is_active`, timestamps, `next_index >= 0` check, and natural-key uniqueness `UNIQUE(chain, network, keyset_id)`.
  - [ ] AC2: Migration creates `asset_catalog` with at least `chain`, `network`, `asset`, `wallet_account_id`, `address_scheme`, `default_expires_in_seconds`, `enabled`, token metadata columns (`token_standard`, `token_contract`, `token_decimals`), optional `chain_id`, and timestamps.
  - [ ] AC3: `asset_catalog.default_expires_in_seconds` is constrained to `[60, 2592000]`.
  - [ ] AC4: Enabled catalog rows must reference an active wallet account; EVM rows must include non-null `chain_id`.
  - [ ] AC5: Seed data includes BTC, ETH, and USDT rows for approved networks, and ETH/USDT rows on same EVM network reference the same `wallet_account_id`.
  - [ ] AC6: Token/native metadata invariants are enforced: token assets require non-null `token_standard`, `token_contract`, and `token_decimals`; native assets require those fields to be null.
  - [ ] AC7: Token/native row typing is determined by catalog metadata, not asset symbol branching: `token_standard IS NOT NULL` => token row; `token_standard IS NULL` => native row.
- Notes: Seed reruns must be idempotent.

### FR-002 - Enforce startup catalog-wallet validation gate

- Description: Service bootstrap must validate enabled asset mappings before serving requests.
- Acceptance criteria:
  - [ ] AC1: Startup loads enabled `asset_catalog` rows and verifies referenced wallet account exists and is active.
  - [ ] AC2: Startup verifies allocator compatibility rules (chain/network consistency, EVM `chain_id` presence, and token/native metadata invariants).
  - [ ] AC3: Address-scheme allow-list is enforced from one centralized validation ruleset (bootstrap/config) with current values: Bitcoin -> `bip84_p2wpkh`; EVM -> `evm_bip44`.
  - [ ] AC4: Startup enforces keyset depth policy by chain:
    - Bitcoin keyset must be account-level BIP84 (`m/84'/coin_type'/account'` completed before xpub export)
    - EVM keyset must be account-level BIP44 (`m/44'/60'/account'` completed before xpub export)
    - Parsed extended public key metadata must satisfy `depth == 3` and hardened `child_number` (`child_number >= 0x80000000`)
    - `depth != 3` (including common change-level depth `4`) is rejected as configuration policy mismatch
  - [ ] AC5: Startup enforces `derivation_path_template` policy as exact suffix `0/{index}` for this feature.
  - [ ] AC6: On key parsing/normalization incompatibility, startup fails closed with typed error `invalid_key_material_format`; no silent fallback is allowed.
  - [ ] AC7: On any validation failure, service exits before accepting traffic and emits structured configuration errors.
  - [ ] AC8: Startup validation can run repeatedly without mutating state.
- Notes:
  - Validation errors must map to deterministic codes for operations troubleshooting.
  - Use `invalid_key_material_format` for parse/normalization failures.
  - Use `invalid_configuration` for policy mismatches (depth/suffix/allow-list/token-invariant violations).

### FR-003 - Provide wallet allocation gateway contract and runtime mode selection

- Description: Allocation must be abstracted behind outbound port with selectable implementations.
- Acceptance criteria:
  - [ ] AC1: Outbound port accepts derivation-ready inputs `(chain, network, keyset_id, derivation_path_template, derivation_index, address_scheme)` plus context and returns derived address metadata.
  - [ ] AC2: `derivation_index` in the outbound port is non-negative 64-bit (`int64`) and must not be narrowed to 32-bit.
  - [ ] AC3: `derivation_path_template` is interpreted as suffix relative to `keyset_id` extended public key depth, must not contain hardened segments (`'` or `h`), and must be exactly `0/{index}` in this feature.
  - [ ] AC4: `keyset_id` refers to account-level extended public key material (not change-level or address-index-level xpub).
  - [ ] AC5: `dev/test` adapter implementation is available and selectable by configuration.
  - [ ] AC6: `prod` adapter placeholder is wired and returns explicit not-configured/not-implemented signal when selected without real implementation.
  - [ ] AC7: Startup validates required adapter configuration for selected mode.
- Notes:
  - Adapter must not perform DB lookups by `wallet_account_id`.
  - DB entities remain in repository/application layers.

### FR-004 - Derive deterministic BTC and EVM addresses in dev/test

- Description: Dev/test adapter must derive real addresses from controlled public derivation material.
- Acceptance criteria:
  - [ ] AC1: BTC dev/test derivation supports `regtest` and `testnet`, uses configured extended public keys, and emits network-correct addresses.
  - [ ] AC2: EVM dev/test derivation supports `sepolia` and emits deterministic 20-byte EVM addresses.
  - [ ] AC3: Same `(keyset_id, derivation_path_template, derivation_index)` input always returns same address output.
  - [ ] AC4: Adapter returns typed errors for unsupported network, invalid key material, or derivation failure.
  - [ ] AC5: Allowed key material formats are explicitly documented and validated at startup:
    - BTC non-prod accepts `tpub` and `vpub`; startup normalization converts to library-supported internal form (including `vpub` <-> `tpub` version-byte normalization when needed) while enforcing `address_scheme=bip84_p2wpkh`
    - EVM accepts BIP32 secp256k1 extended public key serialization with `xpub` or `tpub` prefixes and normalizes to one internal representation; `ypub`/`zpub`/`vpub` are rejected
  - [ ] AC6: Adapter returns raw derived address; application layer is responsible for canonical storage format and response formatting (including EIP-55 for EVM response).
  - [ ] AC7: If safe key normalization is not supported by the selected library/runtime, startup must fail with typed error `invalid_key_material_format` rather than downgrading behavior.
- Notes: Implementation must be recoverable by operator-controlled keys.

### FR-005 - Consume allocator cursor atomically inside create transaction

- Description: Payment request create flow must consume index and persist request atomically.
- Acceptance criteria:
  - [ ] AC1: Flow locks `wallet_accounts` row `FOR UPDATE`, reads `next_index` (`BIGINT`), derives address, persists payment request, and increments `next_index` in one transaction.
  - [ ] AC2: Any failure before commit rolls back payment request write and cursor increment.
  - [ ] AC3: Uniqueness constraints on `(wallet_account_id, derivation_index)` and `(chain, network, address_canonical)` are enforced.
  - [ ] AC4: Shared allocator behavior (ETH+USDT same wallet account) consumes a single monotonic cursor safely.
  - [ ] AC5: Address mapping is explicit: DB persists canonical value in `payment_requests.address_canonical`; API returns formatted `payment_instructions.address` (EVM EIP-55, Bitcoin normalized per scheme).
- Notes: Idempotency integration must prevent replay from consuming extra index.

### FR-006 - Verify allocation correctness with smoke and concurrency tests

- Description: Feature must include executable tests proving no duplicate allocation artifacts.
- Acceptance criteria:
  - [ ] AC1: Smoke tests create BTC, ETH, and USDT requests and receive syntactically valid addresses.
  - [ ] AC2: Concurrency test with 50-200 parallel creates for same tuple produces zero duplicate index/address.
  - [ ] AC3: Mixed ETH+USDT concurrent test on shared allocator also produces zero duplicates.
  - [ ] AC4: Test suite asserts `next_index` advancement count matches successful create count.
- Notes: Tests may run against local PostgreSQL test DB.

### FR-007 - Guard dev/test environments against accidental mainnet allocation

- Description: Dev/test adapter must provide explicit safety controls for mainnet derivation.
- Acceptance criteria:
  - [ ] AC1: By default, dev/test mode rejects mainnet allocation requests unless explicit override is enabled.
  - [ ] AC2: Rejection returns machine-readable code `mainnet_allocation_blocked` and does not consume cursor index.
  - [ ] AC3: Mainnet override is controlled by `PAYMENT_REQUEST_DEVTEST_ALLOW_MAINNET=true`, and startup logs include an explicit warning when enabled.
  - [ ] AC4: Configuration docs clearly state override behavior and risks.
- Notes: This is a safety requirement to prevent accidental real-fund routing during local testing.

## Non-functional requirements

- Performance (NFR-001): In local integration profile, p95 allocation path latency (cursor lock through commit) is <= 300 ms at 20 RPS.
- Availability/Reliability (NFR-002): Under >=200 concurrent create operations, duplicate index/address count remains `0`; transaction rollback correctness is 100% for induced failures.
- Security/Privacy (NFR-003): No private key, seed phrase, or secret-bearing config value is logged; only public derivation material is allowed in dev/test adapter configuration.
- Compliance (NFR-004): Asset seed/token metadata changes require traceable change record before enabling rows in production-like environments.
- Observability (NFR-005): Logs and metrics expose allocation mode, chain/network/asset, success/failure reason, retry count, and latency histogram.
- Maintainability (NFR-006): Implementation preserves architecture boundaries and passes `go fmt ./...`, `go vet ./...`, `go list ./...`, and targeted `go test` suites.

## Dependencies and integrations

- External systems: Address derivation libraries (BIP32/BIP84 for BTC, secp256k1/EVM utilities) and, later, production wallet provider.
- Internal services: PostgreSQL repositories, payment-request create use case, bootstrap configuration loader.
