---
doc: 00_problem
spec_date: 2026-02-22
slug: payment-request-settlements-api
mode: Quick
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-reorg-recovery-finality-policy
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: null
  tasks: 03_tasks.md
  test_plan: null
---

# Problem & Goals

## Context

- Background: reconciliation already stores per-transaction evidence in `app.payment_request_settlements`.
- Users or stakeholders: operators and integrators who need to inspect how a payment request reached current status.
- Why now: current public API only exposes `/v1/payment-requests/{id}`, so settlement details require direct DB access.

## Constraints (optional)

- Technical constraints: keep strict existing Clean Architecture boundaries and no new migration for this scope.
- Timeline/cost constraints: implement as a small additive read API in current iteration.
- Compliance/security constraints: endpoint must not leak unrelated payment request data.

## Problem statement

- Current pain: there is no API to fetch settlement evidence rows for a payment request.
- Current pain: troubleshooting partial payments, reorg impact, or canonical/non-canonical evidence needs manual SQL.
- Evidence or examples: user requested a “simple #1” feature to expose settlement inspection via API.

## Goals

- G1: add a stable read API to list settlement evidence for one payment request.
- G2: return deterministic ordering and explicit evidence fields needed for reconciliation debugging.
- G3: preserve current service behavior for all existing endpoints.

## Non-goals (out of scope)

- NG1: changing reconciliation write strategy or settlement schema.
- NG2: adding filtering/pagination in this iteration.
- NG3: exposing raw blockchain provider payloads.

## Assumptions

- A1: callers want all currently stored evidence rows for a request in one response.
- A2: when payment request exists but has no evidence yet, API returns `200` with empty list.

## Open questions

- Q1: none for this scope.

## Success metrics

- Metric: endpoint correctness.
- Target: valid request ID returns `200` and settlement list; unknown request returns `404`.
- Metric: regression safety.
- Target: existing payment request create/get and reconciliation flows remain unchanged.
