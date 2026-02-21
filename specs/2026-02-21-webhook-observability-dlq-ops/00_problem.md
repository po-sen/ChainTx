---
doc: 00_problem
spec_date: 2026-02-21
slug: webhook-observability-dlq-ops
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-status-outbox-delivery
  - 2026-02-21-webhook-lease-heartbeat-renewal
  - 2026-02-21-webhook-idempotency-replay-signature
  - 2026-02-21-webhook-retry-jitter-budget
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Problem & Goals

## Context

- Background: webhook outbox delivery is running with lease, retry jitter, retry budget, and idempotency signature headers.
- Users or stakeholders: operations teams and service integrators who need to monitor delivery health and recover failed deliveries.
- Why now: user requested to implement the next two priorities: observability baseline and DLQ manual compensation.

## Constraints (optional)

- Technical constraints: preserve current Clean Architecture boundaries and existing webhook outbox schema unless migration is strictly necessary.
- Timeline/cost constraints: incremental implementation within current service and dispatcher runtimes.
- Compliance/security constraints: avoid exposing webhook payload body in operator endpoints.

## Problem statement

- Current pain: there is no dedicated API surface to inspect webhook outbox health (pending backlog, oldest pending age, DLQ size).
- Current pain: failed webhook events are terminal (`failed`) but there is no manual `requeue` or `cancel` workflow through application ports.
- Evidence or examples: operators currently depend on direct SQL inspection and ad-hoc SQL updates for failed delivery recovery.

## Goals

- G1: add webhook outbox observability read endpoints for backlog and DLQ visibility.
- G2: add manual DLQ compensation operations (`requeue`, `cancel`) through application use cases and HTTP adapters.
- G3: extend dispatch cycle telemetry with delivery result buckets (2xx/4xx/5xx/network error) for actionable logs.
- G4: keep implementation inside application/adapters/infrastructure layers with strict dependency direction.

## Non-goals (out of scope)

- NG1: automatic retry-budget tuning or adaptive circuit breaker.
- NG2: introducing a new webhook delivery status beyond existing `pending/delivered/failed` in this iteration.
- NG3: introducing external metrics systems (Prometheus/OpenTelemetry exporters) in this iteration.

## Assumptions

- A1: existing `failed` status is treated as DLQ for operator workflows.
- A2: manual cancel is represented as setting/keeping `delivery_status='failed'` with normalized manual-cancel error reason.

## Open questions

- Q1: none for this iteration.

## Success metrics

- Metric: operator visibility.
- Target: `GET /v1/webhook-outbox/overview` and `GET /v1/webhook-outbox/dlq` return consistent outbox snapshots.
- Metric: manual recovery capability.
- Target: operators can requeue failed events and cancel events without direct SQL.
- Metric: delivery diagnostics.
- Target: dispatcher cycle logs include 2xx/4xx/5xx/network delivery counters.
