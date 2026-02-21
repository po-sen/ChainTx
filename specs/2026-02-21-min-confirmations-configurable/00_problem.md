---
doc: 00_problem
spec_date: 2026-02-21
slug: min-confirmations-configurable
mode: Quick
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
  - 2026-02-20-chainobserver-solid-refactor
  - 2026-02-20-detected-threshold-configurable
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: null
  tasks: 03_tasks.md
  test_plan: null
---

# Problem & Goals

## Context

- Background: current reconciler status progression is amount-threshold based. It does not let operators configure chain confirmation depth for `confirmed` transitions.
- Users or stakeholders: product/operations need to control settlement confidence (for example 1 vs 6 BTC confirmations).
- Why now: user explicitly requested configurable block-confirmation gating and asked where BTC confirmation count is configured.

## Constraints (optional)

- Technical constraints: preserve existing Clean Architecture boundaries and adapter contracts.
- Technical constraints: no DB schema migration in this iteration.

## Problem statement

- Current pain: there is no explicit config for minimum confirmations per chain family.
- Current pain: `confirmed` can be reached too early for risk-sensitive operation if deeper confirmations are required.

## Goals

- G1: provide operator-configurable minimum confirmations for BTC and EVM rails.
- G2: apply minimum confirmations to `confirmed` decision while keeping `detected` as earlier signal.
- G3: keep defaults backward-compatible (equivalent to current behavior).

## Non-goals (out of scope)

- NG1: per-asset/per-network confirmation policy matrix in this iteration.
- NG2: introduce new statuses (`underpaid`, `reorged`) in this iteration.

## Assumptions

- A1: one global BTC minimum confirmation setting is acceptable for all BTC networks currently supported.
- A2: one global EVM minimum confirmation setting is acceptable for ETH/USDT on current EVM networks.

## Open questions

- Q1: resolved for this iteration: `detected` remains early signal (BTC includes mempool; EVM uses latest state), `confirmed` is gated by configured minimum confirmations.

## Success metrics

- Metric: operability
- Target: operators can set confirmation depth through env without code changes.
- Metric: behavior correctness
- Target: partial/full payment transitions follow configured depth rules across BTC/ETH/USDT.
