---
doc: 00_problem
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

# Problem & Goals

## Context

- Background: ChainTx already uses DB-first allocation primitives (`wallet_accounts`, `asset_catalog`) for routing metadata and cursor control.
- Background: Local-chain integration currently still relies on runtime env override for Ethereum sepolia chain id, which is not the desired long-term source-of-truth.
- Users or stakeholders: Backend engineers, QA, and integration developers who need local and public test-chain support without runtime drift.
- Why now: We need a DB-sourced model where chain id and token metadata are determined by catalog rows, and local/public EVM variants are represented explicitly.

## Constraints (optional)

- Technical constraints: Keep Clean Architecture + Hexagonal boundaries.
- Technical constraints: Preserve separate compose stacks per rail (BTC, ETH, USDT, service).
- Technical constraints: Do not modify `specs/2026-02-07-postgresql-backend-service-compose`.
- Technical constraints: Keep Make lifecycle surface minimal (`up/down` family).
- Data constraints: Reuse existing `wallet_accounts` and `asset_catalog` tables; avoid unnecessary new tables unless required.

## Problem statement

- Current pain: Ethereum chain id behavior can drift because local behavior is partly env-driven instead of catalog-driven.
- Current pain: Supporting both local EVM and sepolia semantics is unclear when metadata is not fully represented in DB rows.
- Current pain: The desired model is "write into DB and treat as distinct wallet accounts"; current spec package was documenting implemented changes, not this target architecture.

## Goals

- G1: Make DB `asset_catalog.chain_id` + token metadata the source-of-truth for EVM response fields.
- G2: Represent local EVM and sepolia as separate DB tuples with separate `wallet_account_id` cursors.
- G3: Keep ETH and USDT sharing one wallet account per network (sepolia shares one, local shares one).
- G4: Remove local flow dependence on `PAYMENT_REQUEST_DEVTEST_ETH_SEPOLIA_CHAIN_ID` as primary control path.
- G5: Keep local startup deterministic: start chains, then upsert local DB rows from artifacts.

## Non-goals (out of scope)

- NG1: No production wallet-provider redesign.
- NG2: No new public API endpoint.
- NG3: No TRON/SOL rail implementation in this spec.
- NG4: No changes to BTC allocation model beyond existing regtest/testnet rows.

## Assumptions

- A1: Local EVM variant is represented with `network=local` (not overloading `network=sepolia`).
- A2: `ethereum/local` uses `chain_id=31337` and USDT `token_decimals=6`.
- A3: `ethereum/sepolia` remains `chain_id=11155111`.
- A4: Service request payload can already pass arbitrary normalized `network` strings (for example `local`).

## Open questions

- Q1: None for this revision; network naming is fixed as `local` for DB tuple clarity.

## Success metrics

- Metric: DB source-of-truth correctness
- Target: ETH/USDT `chain_id` in payment instructions always equals `asset_catalog.chain_id` for the requested tuple.
- Metric: Parallel variant support
- Target: Service can create requests for both `ethereum/sepolia/*` and `ethereum/local/*` without toggling env chain-id.
- Metric: Cursor isolation
- Target: `wa_eth_sepolia_*` and `wa_eth_local_*` maintain independent `next_index` progress.
- Metric: Operability
- Target: `make local-up-all` + local DB upsert enables manual ETH/USDT local receive tests using `network=local`.
