---
doc: 00_problem
spec_date: 2026-02-12
slug: local-btc-eth-usdt-chain-sim
mode: Full
status: DONE
owners:
  - posen
depends_on: []
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Problem & Goals

## Context

- Background: ChainTx currently supports BTC (`bitcoin/regtest`, `bitcoin/testnet`), ETH (`ethereum/sepolia`), and USDT-ERC20 (`ethereum/sepolia`) for payment request generation, but local runtime only ships app + PostgreSQL compose by default.
- Background: For this local simulation spec, we intentionally lock scope to deterministic local networks: BTC `regtest` and local EVM devnet (`chain_id=31337`) used to simulate sepolia-style flows.
- Users or stakeholders: Backend engineers, QA engineers, and integration testers who need reproducible local chain simulation to validate service behavior.
- Why now: Team needs a deterministic local environment to validate service startup and payment flows without relying on public testnets or external infra.

## Constraints (optional)

- Technical constraints: Keep this work isolated from business/domain logic; focus on local infra, scripts, and developer workflow.
- Technical constraints: BTC, ETH, and USDT rails must be started via separate Docker Compose files.
- Technical constraints: Provide Makefile orchestration with resource-aware profiles (`local-up` minimal, `local-up-all` optional full stack).
- Technical constraints: Rails must be network-isolated by default; cross-rail calls use explicit host-level RPC contracts (not shared compose network aliases).
- Technical constraints: New rails (for example `usdt-tron`) must be additive and must not force runtime-contract changes in existing rails.
- Technical constraints: BTC receiver key export must follow descriptor-based account-level policy (BIP84 external chain derivation).
- Compliance/security constraints: Local-only keys/funds; do not use real mainnet credentials.
- Timeline/cost constraints: First iteration must prioritize reliable startup and reproducible smoke flows over production-grade node hardening.

## Problem statement

- Current pain: There is no chain simulation stack in repo to exercise BTC/EVM transfer behavior locally.
- Current pain: BTC test wallet setup (payer vs receiver-xpub wallet) is manual and error-prone.
- Current pain: Without fixed local chain constants (chain id, token decimals, network contract), scripts and smoke checks drift and become flaky.
- Evidence or examples: Existing `deployments/service/docker-compose.yml` starts app + PostgreSQL only; no BTC node, EVM node, USDT contract deployment workflow, or Make targets for chain stacks.

## Goals

- G1: Provide local chain simulation stacks for BTC, ETH, and USDT usage in ChainTx.
- G2: Ensure each rail is controlled by a separate Docker Compose file, with independent up/down lifecycle.
- G3: Bootstrap two BTC wallets automatically: one funded payer wallet and one receiver wallet used to export descriptor/xpub artifacts for receiving flow tests.
- G4: Start ChainTx service in local mode and enable quick smoke verification through Makefile commands.
- G5: Produce deterministic artifacts (RPC endpoints, wallet metadata, xpub, contract address) for test reuse.
- G6: Default startup path must avoid high host pressure by starting only minimal required stacks, with full-stack startup as optional command.
- G7: Keep rail architecture open for extension (new rail add-on) and closed for modification (existing rail contracts unchanged).

## Non-goals (out of scope)

- NG1: No production deployment, key management, or node hardening.
- NG2: No new business API/domain behavior changes in ChainTx core use cases.
- NG3: No blockchain indexing, webhook, or confirmation monitoring pipeline in this iteration.
- NG4: No local BTC `testnet` simulation in this first local-sim iteration.
- NG5: No non-Ethereum USDT rails (for example TRON/Omni) unless explicitly added in future spec.

## Assumptions

- A1: Local BTC simulation baseline is `regtest` only.
- A2: Local EVM simulation baseline uses `chain_id=31337`.
- A3: Local USDT ERC20 contract baseline uses `token_decimals=6`.
- A4: USDT local stack can depend on ETH local RPC endpoint while still being managed by its own compose file.
- A5: Cross-rail dependency contract uses host-level RPC endpoint configuration (for example `ETH_RPC_URL`) instead of shared compose network DNS.
- A6: Team accepts deterministic test keys/artifacts stored in local-only files ignored by git.
- A7: Docker Engine + Docker Compose plugin are available in developer machines.

## Open questions

- Q1: None for this iteration; baseline local constants are fixed in this spec to reduce implementation drift.

## Success metrics

- Metric: Startup ergonomics (default profile)
- Target: `make local-up` starts minimal profile (service + BTC stack) within 3 minutes on a clean machine with images pre-pulled.
- Metric: Startup ergonomics (optional full profile)
- Target: `make local-up-all` starts service + BTC + ETH + USDT within 6 minutes on a clean machine with images pre-pulled.
- Metric: BTC wallet bootstrap
- Target: One command creates payer wallet and receiver descriptor/xpub wallet data, funds payer on regtest, and writes artifact output without manual CLI steps.
- Metric: Stack isolation
- Target: `make chain-up-btc`, `make chain-up-eth`, and `make chain-up-usdt` can run independently and can be stopped independently.
- Metric: Local validation coverage
- Target: `scripts/local-chains/smoke_local.sh` validates default profile; `scripts/local-chains/smoke_local_all.sh` validates full-profile BTC/ETH/USDT checks with deterministic pass/fail summary.
