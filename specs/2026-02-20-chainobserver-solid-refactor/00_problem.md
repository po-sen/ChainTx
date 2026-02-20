---
doc: 00_problem
spec_date: 2026-02-20
slug: chainobserver-solid-refactor
mode: Quick
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: null
  tasks: 03_tasks.md
  test_plan: null
---

# Problem & Goals

## Context

- Background: `internal/adapters/outbound/chainobserver/devtest/gateway.go` currently mixes gateway routing, BTC HTTP parsing, EVM RPC protocol, amount parsing, and ABI encoding in a single file.
- Users or stakeholders: maintainers who requested stricter SOLID alignment and easier future extension.
- Why now: user explicitly requested SOLID refactor while preserving existing behavior.

## Constraints (optional)

- Technical constraints: no domain/application contract break for existing `PaymentChainObserverGateway` port.
- Technical constraints: preserve current runtime behavior and test expectations.

## Problem statement

- Current pain: single large file has multiple reasons to change and weak separation of responsibilities.
- Current pain: adding a new chain/provider requires editing a monolithic adapter file.

## Goals

- G1: split chain observer implementation into focused components (coordinator, BTC observer, EVM observer, RPC client/utilities).
- G2: keep functional behavior equivalent for BTC/EVM amount detection/confirmation.
- G3: keep tests passing and improve code readability/maintainability.

## Non-goals (out of scope)

- NG1: changing business status transition rules.
- NG2: adding new chain/network support in this refactor.

## Assumptions

- A1: existing integration points (config/env/use case) remain unchanged.
- A2: current observer unit tests represent required behavior baseline.

## Open questions

- Q1: resolved in this refactor: public adapter entrypoint remains `NewGateway(Config)`.

## Success metrics

- Metric: maintainability
- Target: chain observer code is split into focused files with clear responsibilities.
- Metric: compatibility
- Target: `go test -short ./...` passes without behavioral regressions.
