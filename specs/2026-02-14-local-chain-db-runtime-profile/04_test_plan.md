---
doc: 04_test_plan
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

# Test Plan

## Scope

- Covered:
  - DB tuple coexistence (`sepolia` + `local`) for ETH/USDT.
  - DB-first chain-id/token metadata propagation.
  - Wallet account cursor split across local/sepolia.
  - Startup DB upsert idempotency from artifacts.
  - Gateway derivation chain-id behavior via input contract.
- Not covered:
  - Mainnet production provider.
  - TRON/SOL rail behavior.

## Tests

### Unit

- TC-001: DeriveAddressInput chain-id propagation

  - Linked requirements: FR-005, NFR-001
  - Steps: Unit test use case + gateway with input chain id values `31337` and `11155111`.
  - Expected: Derived output carries same non-mainnet chain id; mainnet remains `1`.

- TC-002: Validation rejects missing local keyset mapping
  - Linked requirements: FR-003, NFR-005
  - Steps: Startup validation with `wa_eth_local_001` mapped keyset missing.
  - Expected: `invalid_configuration` before service traffic starts.

### Integration

- TC-101: Migration/upsert creates local and sepolia tuples

  - Linked requirements: FR-002, FR-003, FR-004
  - Steps: Migrate clean DB, query `app.wallet_accounts` and `app.asset_catalog`.
  - Expected: all required local/sepolia rows exist with correct wallet/account mappings.

- TC-102: Local tuple metadata correctness

  - Linked requirements: FR-001, FR-002
  - Steps: Query `asset_catalog` for `ethereum/local/ETH|USDT`.
  - Expected: `chain_id=31337`; USDT row has `token_decimals=6` and non-empty `token_contract`.

- TC-103: Sepolia tuple untouched by local upsert

  - Linked requirements: FR-002, FR-004
  - Steps: Run local artifact upsert then query sepolia rows.
  - Expected: sepolia chain id remains `11155111`; sepolia contract fields unchanged.

- TC-104: Local payment request flow uses DB metadata

  - Linked requirements: FR-001, FR-006
  - Steps: Create request with `chain=ethereum`, `network=local`, `asset=ETH` and `USDT`.
  - Expected: response chain_id/token fields match local DB row values.

- TC-105: Sepolia payment request flow remains compatible

  - Linked requirements: FR-006, NFR-003
  - Steps: Create request with `network=sepolia` for ETH/USDT.
  - Expected: response metadata matches sepolia DB rows; no regression.

- TC-106: Cursor isolation between local and sepolia wallet accounts
  - Linked requirements: FR-003
  - Steps: Alternate create requests between `network=local` and `network=sepolia`.
  - Expected: each wallet account `next_index` increments independently.

### E2E

- TC-201: Full local startup and receive test

  - Linked requirements: FR-004, FR-006, FR-007, FR-008
  - Steps: `make local-up-all` then run manual local ETH/USDT receive flow from README.
  - Expected: local chain receives transfers and service instructions match DB-local tuple.

- TC-202: Repeated startup idempotency
  - Linked requirements: FR-007, NFR-002
  - Steps: run `make local-down && make local-up-all` for 3 cycles.
  - Expected: no duplicate tuple rows, no startup failure from repeated upserts.

## Edge cases and failure modes

- Case: `eth.json` missing `usdt_contract_address`.
- Expected: local DB upsert fails with clear error and does not write partial local USDT row.

- Case: local row exists but `wallet_account_id` points to inactive wallet.
- Expected: startup validation fails before API serves requests.

- Case: local request sent before local tuple upsert.
- Expected: API returns `unsupported_network` for `ethereum/local`.

## NFR verification

- Determinism (NFR-001): repeated local request derives stable chain metadata from catalog rows.
- Reliability (NFR-002): repeated startup cycles keep idempotent DB state.
- Backward compatibility (NFR-003): sepolia rows and responses remain unchanged.
- Maintainability (NFR-004): chain-id ownership located in DB rows, not transient env flags.
- Testability (NFR-005): unit + integration + smoke scenarios cover tuple model and cursor split.

## Execution results

- Status: Pending implementation.
- Planned commands:
  - `go test ./internal/infrastructure/config ./internal/adapters/outbound/wallet/devtest ./internal/application/use_cases`
  - migration integration tests
  - `make local-up-all`
  - local manual receive smoke for `network=local`
