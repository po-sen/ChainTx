---
doc: 02_design
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

# Technical Design

## High-level approach

- Summary: add webhook outbox read/operation use cases and HTTP endpoints in server runtime, and extend dispatch output with delivery bucket telemetry.
- Key decisions:
  - Treat existing `failed` rows as DLQ; no new delivery status introduced.
  - Keep observability snapshot query database-derived and stateless.
  - Manual actions are explicit application use cases backed by status-guarded SQL updates.

## System context

- Components:
  - `internal/application/dto`: new overview/DLQ DTOs and manual command DTOs.
  - `internal/application/ports/in`: new inbound use cases for overview/list/requeue/cancel.
  - `internal/application/ports/out`: new read-model port and repository operation extensions.
  - `internal/application/use_cases`: implementations for new use cases.
  - `internal/adapters/outbound/persistence/postgresql/webhookoutbox`: SQL implementations for read and update operations.
  - `internal/adapters/inbound/http/controllers` and `router`: new HTTP handlers and routes.
  - `internal/infrastructure/di`: wire new use cases into server.
  - `internal/infrastructure/webhook`: log delivery bucket counters from dispatch output.
- Interfaces:
  - `WebhookOutboxReadModel` (overview + DLQ list)
  - `WebhookOutboxRepository` extended with requeue/cancel operations.

## Key flows

- Flow 1: observability overview

  - HTTP controller validates request and invokes overview use case.
  - Read model executes aggregate SQL and returns snapshot DTO.
  - Controller returns 200 JSON snapshot.

- Flow 2: DLQ list

  - HTTP controller parses `limit` query.
  - Use case validates range and delegates to read model.
  - Read model returns failed rows projected to safe fields only.

- Flow 3: manual requeue

  - Controller extracts `event_id` path parameter.
  - Use case validates event id and calls repository requeue with `now`.
  - Repository updates row with `WHERE delivery_status='failed'`; zero rows => conflict.

- Flow 4: manual cancel

  - Controller extracts `event_id` and optional reason.
  - Use case normalizes reason and calls repository cancel with `now`.
  - Repository updates row with `WHERE delivery_status IN ('pending','failed')`; delivered or missing => conflict/not found mapping.

- Flow 5: dispatch telemetry
  - Dispatch use case increments per-attempt bucket counters by send outcome.
  - Worker completion log includes bucket counters.

## Diagrams (optional)

```mermaid
flowchart TD
  A[GET /v1/webhook-outbox/overview] --> B[OverviewUseCase]
  B --> C[WebhookOutboxReadModel]
  C --> D[(PostgreSQL)]
  D --> C --> B --> A

  E[POST /v1/webhook-outbox/dlq/{event_id}/requeue] --> F[RequeueUseCase]
  F --> G[WebhookOutboxRepository]
  G --> D
```

## Data model

- Entities: unchanged.
- Schema changes or migrations: none.
- Consistency and idempotency:
  - Requeue/cancel rely on guarded update predicates on current `delivery_status`.
  - Operations are idempotent at SQL-level by status guards (repeat call after transition yields no-op/conflict).

## API or contracts

- Endpoints or events:
  - `GET /v1/webhook-outbox/overview`
  - `GET /v1/webhook-outbox/dlq?limit=`
  - `POST /v1/webhook-outbox/dlq/{event_id}/requeue`
  - `POST /v1/webhook-outbox/events/{event_id}/cancel`
- Request/response examples:
  - cancel request body: `{ "reason": "operator_cancelled" }`
  - requeue response: `{ "event_id": "evt_xxx", "delivery_status": "pending" }`

## Backward compatibility (optional)

- API compatibility: additive HTTP routes only.
- Data migration compatibility: no data migration required.

## Failure modes and resiliency

- Retries/timeouts: unchanged dispatcher retry mechanics.
- Backpressure/limits: DLQ list query limit guard prevents unbounded responses.
- Degradation strategy: if outbox query/update fails, structured internal error surfaces to caller.

## Observability

- Logs: webhook dispatch cycle completion includes delivery bucket counters.
- Metrics: outbox overview endpoint provides machine-readable backlog snapshot for scraping/polling.
- Traces: not introduced.
- Alerts: external alerting can poll overview and threshold on `pending_ready_count`, `oldest_pending_age_seconds`, `failed_count`.

## Security

- Authentication/authorization: unchanged from existing service model.
- Secrets: no new secret values introduced.
- Abuse cases: manual operation endpoints validate path/body to reduce malformed update attempts.

## Alternatives considered

- Option A: introduce new `cancelled` status and schema migration.
- Option B: keep current statuses and use `failed + manual_cancelled` reason marker.
- Why chosen: minimal migration risk and faster rollout aligned with current constraints.

## Risks

- Risk: no auth layer means operational endpoints are open wherever API is exposed.
- Mitigation: document deployment expectation to protect service network perimeter and add auth in future iteration.
