---
doc: 00_problem
spec_date: 2026-02-21
slug: webhook-lease-heartbeat-renewal
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-status-outbox-delivery
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Problem & Goals

## Context

- Background: webhook dispatcher currently claims outbox events with `lease_until` and processes them outside transaction.
- Users or stakeholders: operations and merchants relying on webhook correctness and stable downstream traffic.
- Why now: lease-based claim is in production baseline; next hardening step is to prevent duplicate processing during long-running delivery attempts.

## Constraints (optional)

- Technical constraints: keep current Clean Architecture boundaries (application ports + outbound repository/gateway adapters).
- Timeline/cost constraints: incremental change only; avoid redesigning outbox table model.
- Compliance/security constraints: no secret exposure; preserve existing HMAC signing flow.

## Problem statement

- Current pain: if a single delivery attempt lasts longer than configured lease, another worker can reclaim the same event while first worker is still active.
- Evidence or examples: with short lease and slow/blocked endpoint, the same event can be processed concurrently by different workers before first attempt finishes.

## Goals

- G1: add heartbeat-based lease renewal during webhook delivery execution.
- G2: prevent active in-flight events from being reclaimed until worker stops heartbeating or finishes.
- G3: keep crash recovery behavior intact (stale claims become reclaimable after lease timeout).
- G4: preserve current retry/backoff/failed semantics and observability.

## Non-goals (out of scope)

- NG1: exactly-once delivery semantics.
- NG2: replacing outbox model with queue middleware.
- NG3: replay UI/API for failed events.

## Assumptions

- A1: dispatcher remains at-least-once delivery system; downstream idempotency by `event_id` is still required.
- A2: repository updates can use compare-and-lease-owner checks for safe renewal.

## Open questions

- Q1: heartbeat interval strategy (fixed vs derived from lease duration) is decided in design.
- Q2: whether to emit separate lease-renew metrics is deferred; cycle logs are mandatory.

## Success metrics

- Metric: concurrent duplicate processing window.
- Target: while worker heartbeats normally, the same event is not reclaimable by other workers before send attempt completes.
- Metric: resilience.
- Target: if worker crashes, event remains reclaimable after lease expiry without manual action.
- Metric: compatibility.
- Target: existing `pending -> delivered/failed` transitions and retry counters remain unchanged.
