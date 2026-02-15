---
doc: 01_requirements
spec_date: 2026-02-14
slug: local-chain-db-runtime-profile
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-10-wallet-allocation-asset-catalog
  - 2026-02-12-local-btc-eth-usdt-chain-sim
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Requirements

## Glossary (optional)

- Local EVM profile: `chain=ethereum`, `network=local`, `chain_id=31337` rows in `asset_catalog`.
- Public test profile: `chain=ethereum`, `network=sepolia`, `chain_id=11155111` rows in `asset_catalog`.
- Shared EVM allocator: ETH and USDT on the same network map to one `wallet_account_id`.

## Out-of-scope behaviors

- OOS1: New chain families (TRON, SOL) implementation.
- OOS2: Production key custody changes.
- OOS3: Public API shape changes.

## Functional requirements

### FR-001 - DB must be source-of-truth for EVM chain metadata

- Description: EVM `chain_id` and token metadata used by responses must come from `asset_catalog` rows.
- Acceptance criteria:
  - [ ] AC1: `payment_instructions.chain_id` for ETH/USDT equals `asset_catalog.chain_id` of resolved tuple.
  - [ ] AC2: `payment_instructions.token_contract` and `token_decimals` for USDT equal `asset_catalog` values.
  - [ ] AC3: Local flow must not require `PAYMENT_REQUEST_DEVTEST_ETH_SEPOLIA_CHAIN_ID` to produce correct chain id.

### FR-002 - Local and sepolia must coexist as separate DB tuples

- Description: DB must represent both public test and local EVM variants simultaneously.
- Acceptance criteria:
  - [ ] AC1: `asset_catalog` has enabled rows for:
    - `ethereum/sepolia/ETH`
    - `ethereum/sepolia/USDT`
    - `ethereum/local/ETH`
    - `ethereum/local/USDT`
  - [ ] AC2: `ethereum/sepolia/*` rows keep `chain_id=11155111`.
  - [ ] AC3: `ethereum/local/*` rows keep `chain_id=31337`.
  - [ ] AC4: `ethereum/local/USDT` row keeps `token_decimals=6`.

### FR-003 - Wallet accounts must be split per EVM network variant

- Description: Local and sepolia EVM variants must have independent allocation cursors.
- Acceptance criteria:
  - [ ] AC1: `wallet_accounts` contains separate accounts for `ethereum/sepolia` and `ethereum/local`.
  - [ ] AC2: ETH+USDT on `sepolia` share one sepolia wallet account.
  - [ ] AC3: ETH+USDT on `local` share one local wallet account.
  - [ ] AC4: Cursor increments on one network do not affect the other network.

### FR-004 - Service startup must upsert local catalog rows from artifacts

- Description: Service startup workflow must persist local runtime values into DB rows.
- Acceptance criteria:
  - [ ] AC1: Startup script validates `eth.json` required fields (including embedded USDT metadata fields) before SQL updates.
  - [ ] AC2: Startup script upserts local wallet/account/catalog rows idempotently.
  - [ ] AC3: Startup script updates local USDT `token_contract` from current artifact.
  - [ ] AC4: Startup script does not overwrite sepolia rows.

### FR-005 - Wallet derivation contract must accept DB-resolved chain id

- Description: Wallet gateway must derive EVM addresses without hardcoding non-mainnet chain id.
- Acceptance criteria:
  - [ ] AC1: `DeriveAddressInput` supports optional expected `ChainID` from DB tuple.
  - [ ] AC2: For EVM non-mainnet networks, gateway output `ChainID` matches input `ChainID`.
  - [ ] AC3: For EVM mainnet, gateway output remains `1`.
  - [ ] AC4: If EVM input needs chain id and value is missing/invalid, flow fails with typed configuration error.

### FR-006 - API/network contract must include local EVM tuple

- Description: Operator must be able to request local EVM addresses explicitly.
- Acceptance criteria:
  - [ ] AC1: `POST /v1/payment-requests` accepts `chain=ethereum`, `network=local`, `asset=ETH|USDT` when local rows are enabled.
  - [ ] AC2: `GET /v1/assets` lists enabled `ethereum/local/*` rows with correct metadata.
  - [ ] AC3: Existing `ethereum/sepolia/*` requests continue to work unchanged.

### FR-007 - Make/local workflow must remain minimal and deterministic

- Description: Local startup remains simple while applying DB-first semantics.
- Acceptance criteria:
  - [ ] AC1: Lifecycle targets stay within `service-up/down`, `chain-up/down-*`, `local-up`, `local-up-all`, `local-down`.
  - [ ] AC2: `local-up-all` starts required chain stacks (BTC/ETH) and USDT deploy step before service DB upsert runs.
  - [ ] AC3: Re-running startup is idempotent (no duplicate tuple rows, stable upsert behavior).

### FR-008 - Documentation must reflect DB-first local profile model

- Description: Runbook must explain tuple-level behavior and manual verification steps.
- Acceptance criteria:
  - [ ] AC1: README explains `sepolia` vs `local` tuple behavior and chain-id source.
  - [ ] AC2: README examples for local ETH/USDT use `network=local`.
  - [ ] AC3: README troubleshooting covers stale artifacts and local catalog upsert rerun.

## Non-functional requirements

- NFR-001 (Determinism): For fixed DB rows and keysets, derived addresses and metadata are deterministic.
- NFR-002 (Reliability): DB upsert script is idempotent and safe across repeated `service-up` runs.
- NFR-003 (Backward compatibility): Existing sepolia tuple behavior remains unchanged.
- NFR-004 (Maintainability): Runtime metadata logic is centralized in DB rows, not distributed across ad-hoc env overrides.
- NFR-005 (Testability): Unit/integration coverage validates tuple selection, chain-id propagation, and cursor isolation.

## Dependencies and integrations

- Dependency specs:
  - `2026-02-10-wallet-allocation-asset-catalog`
  - `2026-02-12-local-btc-eth-usdt-chain-sim`
- Runtime integrations: local artifact scripts, PostgreSQL migrations/seed-upsert, service startup bootstrap.
