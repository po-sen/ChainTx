---
doc: 02_design
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

# Technical Design

## High-level approach

- Summary: Shift local EVM routing to DB-first tuple modeling (`ethereum/local/*`) while keeping public test tuple (`ethereum/sepolia/*`).
- Key decisions:
  - Use existing schema shape (`wallet_accounts`, `asset_catalog`) with new/upserted rows; no new tables required.
  - Distinguish local vs sepolia via `network` field, not runtime env chain-id override.
  - Keep one shared wallet account per EVM network for ETH+USDT.
  - Pass catalog `chain_id` into wallet derivation input to remove hardcoded non-mainnet chain-id behavior.

## System context

- Components:
  - DB migrations/seed upsert:
    - add `wallet_accounts` row for `ethereum/local` (for example `wa_eth_local_001`, keyset `ks_eth_local`)
    - upsert `asset_catalog` rows for `ethereum/local/ETH` and `ethereum/local/USDT`
  - Local scripts:
    - `service_sync_catalog.sh` (or renamed equivalent) upserts local rows from artifacts
  - Application:
    - `DeriveAddressInput` extended to include expected chain id
    - devtest wallet gateway uses input chain id for EVM non-mainnet
  - Make workflow:
    - `local-up-all` keeps chain-first then service-up sequence

## Key flows

- Flow 1: Local full startup

  1. Start `btc`, `eth` chains and run `usdt` deploy step on ETH chain.
  2. Generate `btc.json`, `eth.json` artifacts (`eth.json` contains embedded USDT metadata).
  3. Start service stack.
  4. Run local catalog upsert script:

  - validate artifacts
  - upsert `wallet_accounts` local row
  - upsert `asset_catalog` local ETH/USDT rows (`chain_id=31337`)
  - update local USDT contract from artifact

- Flow 2: Create payment request for local ETH/USDT

  1. Request arrives with `chain=ethereum`, `network=local`, `asset=ETH|USDT`.
  2. Use case resolves local tuple from `asset_catalog`.
  3. Repository locks mapped local wallet account and consumes local cursor.
  4. Use case passes resolved `ChainID` to wallet gateway derive input.
  5. Response returns DB tuple metadata (`chain_id`, token metadata).

- Flow 3: Create payment request for sepolia ETH/USDT
  1. Same flow as local, but tuple is `network=sepolia`.
  2. Uses sepolia wallet cursor and sepolia chain metadata.

## Data model

- Existing schema reused.
- New seeded/upserted tuples:
  - `wallet_accounts`
    - `wa_eth_sepolia_001` -> `chain=ethereum`, `network=sepolia`, `keyset_id=ks_eth_sepolia`
    - `wa_eth_local_001` -> `chain=ethereum`, `network=local`, `keyset_id=ks_eth_local`
  - `asset_catalog`
    - `ethereum/sepolia/ETH` -> wallet `wa_eth_sepolia_001`, `chain_id=11155111`
    - `ethereum/sepolia/USDT` -> wallet `wa_eth_sepolia_001`, sepolia token contract, `token_decimals=6`, `chain_id=11155111`
    - `ethereum/local/ETH` -> wallet `wa_eth_local_001`, `chain_id=31337`
    - `ethereum/local/USDT` -> wallet `wa_eth_local_001`, local token contract, `token_decimals=6`, `chain_id=31337`

## API or contracts

- Public API request shape unchanged.
- Effective behavior change:
  - local EVM usage becomes explicit via `network=local`.
- Port contract update:
  - `internal/application/ports/out.DeriveAddressInput` adds `ChainID *int64`.
- Compatibility:
  - existing `network=sepolia` behavior is preserved.

## Backward compatibility (optional)

- BTC and sepolia tuples remain backward compatible.
- `PAYMENT_REQUEST_DEVTEST_ETH_SEPOLIA_CHAIN_ID` fallback path is removed; EVM chain id is DB-driven.

## Failure modes and resiliency

- Missing local DB tuples:
  - symptom: `unsupported_network` / `unsupported_asset`
  - remediation: rerun service DB upsert script
- Invalid artifact fields during upsert:
  - symptom: startup sync exits non-zero
  - remediation: regenerate artifacts via `chain-up-*` and rerun service-up
- Keyset missing for `ks_eth_local`:
  - symptom: startup validation `invalid_configuration`
  - remediation: ensure devtest keyset map includes local keyset id

## Observability

- Startup log emits local tuple upsert summary with affected rows.
- Validation log includes tuple identifiers on failure (`chain`, `network`, `asset`, `wallet_account_id`).
- Allocation log already contains `chain/network/asset/wallet_account_id/derivation_index` and can verify cursor split.

## Security

- No private key persistence added.
- DB still stores only public catalog metadata and keyset identifiers.
- Mainnet guard behavior in devtest remains unchanged.

## Alternatives considered

- Option A: Keep `network=sepolia` for local by env chain-id override.
  - Rejected: still env-driven and cannot cleanly coexist with public sepolia metadata.
- Option B: Add new profile column/table.
  - Deferred: more invasive schema/query changes than needed for current goal.

## Risks

- Risk: Consumer confusion between `network=sepolia` and `network=local`.
- Mitigation: README and `/v1/assets` examples must explicitly show both tuples.

- Risk: Local USDT contract rotates after reset and DB row becomes stale.
- Mitigation: service startup upsert always refreshes local USDT contract from artifact.
