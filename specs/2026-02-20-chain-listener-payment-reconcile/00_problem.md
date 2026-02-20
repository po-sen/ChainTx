---
doc: 00_problem
spec_date: 2026-02-20
slug: chain-listener-payment-reconcile
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-keyset-startup-sync-audit-rotation
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Problem & Goals

## Context

- Background: Payment requests are currently created with status `pending`, but no runtime process updates status from on-chain activity.
- Background: Local smoke tests prove chain transfers, yet service state does not reconcile those transfers into request lifecycle states.
- Users or stakeholders: payment integrators, operators, and QA who need service-side status to reflect chain settlement progression.
- Why now: User explicitly approved starting the chain-listening feature.

## Constraints (optional)

- Technical constraints: Preserve Clean Architecture boundaries (`application` orchestration, `outbound` adapters for chain/persistence IO).
- Technical constraints: Keep default startup stable; reconciler must be configurable and safely disabled.
- Technical constraints: Existing schema/data must remain compatible.

## Problem statement

- Current pain: API returns `pending` forever even after funds arrive on chain.
- Current pain: No automated expiry transition for timed-out requests.
- Current pain: No periodic reconciliation worker exists in service runtime.

## Goals

- G1: Add periodic chain reconciliation worker for open payment requests.
- G2: Implement lifecycle transitions: `pending -> detected -> confirmed` and `pending/detected -> expired`.
- G3: Support first-phase chain observation for BTC (Esplora API) and EVM (JSON-RPC balance checks).
- G4: Keep reconciler idempotent and safe under repeated polling.

## Non-goals (out of scope)

- NG1: Reorg rollback handling and finalized confirmation depth per asset/provider.
- NG2: Websocket/event-stream listeners (phase-1 uses polling only).
- NG3: On-chain proof API responses in this iteration.

## Assumptions

- A1: Addresses generated per request are unique and controlled by service wallet derivation.
- A2: BTC observation can rely on Esplora `/address/{address}` chain/mempool aggregate stats.
- A3: EVM observation can use `eth_getBalance` for native asset and `balanceOf` for ERC20.

## Open questions

- Q1: Resolved for phase-1: default `confirmed` criterion is observed amount meeting expected amount in latest confirmed chain state (no deep finality model yet).
- Q2: Resolved for phase-1: reconciler is off by default and enabled by config.

## Success metrics

- Metric: Lifecycle convergence
- Target: On local chain smoke, paid requests leave `pending` and reach `confirmed`.
- Metric: Expiry correctness
- Target: Expired unpaid requests transition to `expired` within one polling cycle.
- Metric: Runtime safety
- Target: Reconciler can be enabled/disabled without startup failure regression.
