---
doc: 00_problem
spec_date: 2026-02-20
slug: detected-threshold-configurable
mode: Quick
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
  - 2026-02-20-chainobserver-solid-refactor
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: null
  tasks: 03_tasks.md
  test_plan: null
---

# Problem & Goals

## Context

- Background: currently `detected` is effectively BTC-only and EVM flows jump from `pending` to `confirmed` once expected amount is met.
- Users or stakeholders: product and operations require all supported chains/assets to report partial-funding progress consistently.
- Why now: user explicitly requested all chains support `detected` and threshold must be configurable.

## Constraints (optional)

- Technical constraints: preserve existing Clean Architecture boundaries and current port contracts.
- Technical constraints: avoid schema changes for this iteration.

## Problem statement

- Current pain: ETH/USDT partial payment cannot transition to `detected`.
- Current pain: amount thresholds are hard-coded and not operator-configurable.

## Goals

- G1: enable `detected` transitions for BTC/ETH/USDT consistently.
- G2: make detected/confirmed amount thresholds configurable via env.
- G3: keep default behavior safe and backward-compatible for existing deployments.

## Non-goals (out of scope)

- NG1: adding new payment statuses (e.g., `underpaid`) in this iteration.
- NG2: changing lease/claim architecture.

## Assumptions

- A1: threshold configuration can be modeled as basis points (bps) over `expected_amount_minor`.
- A2: default confirmed threshold remains 100% of expected amount.

## Open questions

- Q1: resolved for this iteration: thresholds apply globally across chains/assets (not per-asset yet).

## Success metrics

- Metric: status coverage
- Target: partial ETH/USDT payment results in `detected` before `confirmed`.
- Metric: operability
- Target: thresholds adjustable through config with validation and docs.
