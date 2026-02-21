---
doc: 00_problem
spec_date: 2026-02-21
slug: webhook-status-outbox-delivery
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
  - 2026-02-21-min-confirmations-configurable
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Problem & Goals

## Context

- Background: payment request status transitions already run in reconciler, but the system does not push those transitions to external systems.
- Users or stakeholders: merchants/operations integrating ChainTx with ERP/order systems need machine-readable status callbacks.
- Why now: user approved moving to webhook as the next feature after chain listener/reconciler readiness.

## Constraints (optional)

- Technical constraints: maintain Clean Architecture boundaries and current reconciler horizontal scaling model.
- Technical constraints: delivery must survive process restarts and transient webhook endpoint failures.
- Compliance/security constraints: webhook authenticity must be verifiable via HMAC-SHA256 signature.

## Problem statement

- Current pain: external systems must poll `GET /v1/payment-requests/{id}` to detect `pending/detected/confirmed/expired` changes.
- Current pain: no retry-capable notification mechanism for transient downstream outage.

## Goals

- G1: emit webhook events when payment request status changes.
- G2: deliver events asynchronously and reliably with retry/backoff.
- G3: support horizontal scaling of dispatcher workers without duplicate delivery claims.
- G4: sign webhook payloads with HMAC-SHA256 so receivers can verify integrity/authenticity.
- G5: run webhook dispatcher as a dedicated runtime/container independent of API/reconciler process lifecycle.
- G6: keep reconciler and webhook-dispatcher runtime wiring decoupled while preserving outbox producer/consumer flow.
- G7: require caller-provided `webhook_url` per payment request and dispatch only to allowlisted destinations.

## Non-goals (out of scope)

- NG1: per-merchant webhook subscription management APIs.
- NG2: exactly-once delivery guarantee across downstream systems.
- NG3: callback transformations/templates per partner.

## Assumptions

- A1: at-least-once semantics with idempotent consumer handling is acceptable.
- A2: event source for this iteration is reconciler-driven status transition path.

## Open questions

- Q1: resolved for this iteration: status transition to `detected`, `confirmed`, and `expired` will enqueue events; no event for no-op scans.
- Q2: resolved for this iteration: retries stop after configurable max attempts and event is marked terminal failure.

## Success metrics

- Metric: delivery reliability
- Target: transient downstream failures are retried automatically with no event loss on process restart.
- Metric: integrity/security
- Target: every outbound webhook includes verifiable HMAC-SHA256 signature and timestamp header.
- Metric: runtime topology clarity
- Target: webhook dispatch runs only in dedicated `webhook-dispatcher` runtime; app/reconciler no longer start webhook worker.
- Metric: destination control
- Target: payment request creation fails when `webhook_url` is missing or outside configured allowlist.
