---
doc: 00_problem
spec_date: 2026-02-21
slug: webhook-idempotency-replay-signature
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-status-outbox-delivery
  - 2026-02-21-webhook-lease-heartbeat-renewal
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Problem & Goals

## Context

- Background: webhook dispatcher already supports outbox delivery, retry, and lease-heartbeat.
- Users or stakeholders: merchant webhook receivers and service operators.
- Why now: current webhook header contract has `event_id` and HMAC signature, but replay defense and idempotency contract are not explicit enough for robust downstream operation.

## Constraints (optional)

- Technical constraints: preserve current Clean Architecture boundaries; keep dispatch flow and outbox schema stable when possible.
- Timeline/cost constraints: incremental hardening, no full webhook subsystem rewrite.
- Compliance/security constraints: no secret leakage in logs; HMAC stays `hmac-sha256`.

## Problem statement

- Current pain: downstream systems may not implement consistent dedupe semantics for webhook retries, and existing signature content does not provide a strong anti-replay contract.
- Evidence or examples: retries are expected in at-least-once delivery, but without a clear receiver contract duplicates may create side effects; replayed requests cannot be bounded by a signed nonce contract.

## Goals

- G1: provide explicit downstream idempotency contract centered on stable event identity.
- G2: strengthen webhook signing contract with replay-resistance primitives (signed timestamp + signed nonce + signed event identity).
- G3: keep delivery semantics as at-least-once while making receiver implementation deterministic and auditable.
- G4: preserve existing deployment/runtime model and avoid DB migration for this scope.

## Non-goals (out of scope)

- NG1: exactly-once delivery guarantee.
- NG2: server-side nonce storage in ChainTx for receiver replay checks.
- NG3: retry jitter/budget changes (handled in a separate feature).

## Assumptions

- A1: receivers can store dedupe keys and short-lived nonce caches.
- A2: webhook receivers can roll forward to a versioned signature contract without requiring ChainTx-side schema changes.

## Open questions

- Q1: compatibility strategy selected for rollout: keep legacy signature header while adding a new versioned anti-replay signature header.
- Q2: receiver replay window default recommendation is documented (for example 300 seconds), but enforced by receiver side.

## Success metrics

- Metric: idempotency interoperability.
- Target: every retry carries a stable idempotency key (`event_id`) and deterministic receiver guidance.
- Metric: replay hardening coverage.
- Target: each outbound webhook includes signed timestamp, signed nonce, and versioned signature metadata.
- Metric: compatibility.
- Target: existing integrations can migrate without immediate outage via dual-header transition strategy.
