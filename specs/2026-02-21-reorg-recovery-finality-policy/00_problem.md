---
doc: 00_problem
spec_date: 2026-02-21
slug: reorg-recovery-finality-policy
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-detected-threshold-configurable
  - 2026-02-21-min-confirmations-configurable
  - 2026-02-20-reconciler-horizontal-scaling
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Problem & Goals

## Context

- Background: current reconciliation is amount-threshold based and can mark a request
  `confirmed` without durable tx-level settlement evidence.
- Users or stakeholders: operators, payment integrators, risk owners, and downstream webhook consumers.
- Why now: user explicitly asked to design reorg handling, including
  `reorged -> confirmed` recovery and a clear observation horizon.

## Constraints (optional)

- Technical constraints: keep Clean Architecture boundaries and current runtime
  split (`server`, `reconciler`, `webhook-dispatcher`, `webhook-alert-worker`).
- Technical constraints: preserve existing API compatibility and additive webhook payload changes only.
- Timeline/cost constraints: incremental delivery over current polling reconciler design.
- Compliance/security constraints: do not store secrets/private keys in new data structures.

## Problem statement

- Current pain: chain reorg can invalidate previously observed confirmations;
  current lifecycle does not encode this reversal explicitly.
- Current pain: there is no explicit policy for how long a `confirmed` payment
  should remain under active reorg monitoring.
- Current pain: settlement evidence persistence currently writes rows every poll
  even when evidence is unchanged, causing avoidable write amplification.
- Current pain: observer adapters can still return aggregate/snapshot evidence
  that cannot deterministically enumerate split installments item-by-item.
- Evidence or examples: current design documents already list reorg rollback as out of scope in phase-1 reconciliation.

## Goals

- G1: support reversible lifecycle for canonical-chain changes (`confirmed -> reorged -> confirmed`).
- G2: move reconciliation decisioning to tx-level settlement evidence instead of balance-only heuristics.
- G2.1: guarantee split-payment visibility where each installment appears as an individual settlement evidence item.
- G3: define explicit operator-controlled finality and observation-window policy.
- G4: provide deterministic status transition reason codes for webhook consumers.
- G5: validate behavior with forced local-chain reorg tests for BTC and EVM rails.

## Non-goals (out of scope)

- NG1: replacing current polling architecture with websocket/indexer streaming.
- NG2: introducing fiat/accounting settlement logic outside chain-confirmation lifecycle.
- NG3: implementing downstream business compensation workflow for reorg losses.

## Assumptions

- A1: dev/local chain environments can simulate reorg scenarios
  (for example invalidating tip and rebuilding alternate chain).
- A2: webhook consumers can tolerate additive fields in status-change payload.
- A3: per-chain-family policy (BTC/EVM) is acceptable for this iteration; per-asset override is deferred.

## Open questions

- Q1: none for this iteration.

## Success metrics

- Metric: reorg rollback correctness.
- Target: in forced-reorg tests, previously `confirmed` requests transition to
  `reorged` within 2 reconciler cycles.
- Metric: recovery correctness.
- Target: when canonical settlement reappears and thresholds are met, `reorged`
  requests return to `confirmed` within 2 cycles.
- Metric: operator control.
- Target: confirmation/finality/observe-window settings are configurable with
  startup validation and documented defaults.
- Metric: settlement write efficiency.
- Target: no-op reconcile cycles (unchanged evidence) produce zero settlement
  row writes for unchanged evidence refs.
- Metric: split-payment observability.
- Target: when a request is paid by N installments,
  `payment_request_settlements` shows N canonical evidence items
  (before any orphaning) with stable chain-native `evidence_ref` values.
