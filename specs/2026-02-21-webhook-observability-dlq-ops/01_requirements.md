---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- DLQ: delivery events currently in webhook outbox `delivery_status='failed'`.
- requeue: move failed webhook event back to `pending` for redelivery.

## Out-of-scope behaviors

- OOS1: per-endpoint adaptive throttling policy.
- OOS2: automated alerts integration with third-party alerting platforms.

## Functional requirements

### FR-001 - Outbox overview observability endpoint

- Description: service must expose webhook outbox overview data through application/inbound HTTP layers.
- Acceptance criteria:
  - [x] AC1: `GET /v1/webhook-outbox/overview` returns current `pending_count`, `pending_ready_count`, `retrying_count`, `failed_count`, and `delivered_count`.
  - [x] AC2: response includes `oldest_pending_age_seconds` (nullable when no pending rows).
  - [x] AC3: values are derived from database snapshot at request time (no stale in-memory cache required).
- Notes: endpoint is operational visibility API, not a billing/accounting API.

### FR-002 - Dispatch delivery bucket telemetry

- Description: webhook dispatch cycle output must include delivery bucket counters.
- Acceptance criteria:
  - [x] AC1: dispatch output includes `http_2xx_count`, `http_4xx_count`, `http_5xx_count`, and `network_error_count`.
  - [x] AC2: worker cycle completion logs include these bucket values.
  - [x] AC3: existing success/retry/fail transition behavior remains unchanged.
- Notes: counters are per cycle execution output.

### FR-003 - DLQ listing endpoint

- Description: service must expose a DLQ listing endpoint for failed webhook events.
- Acceptance criteria:
  - [x] AC1: `GET /v1/webhook-outbox/dlq` returns failed events ordered by `updated_at DESC, id DESC`.
  - [x] AC2: endpoint supports `limit` query param with default 50 and valid range `1..200`.
  - [x] AC3: response includes operator-safe fields (`event_id`, `payment_request_id`, `destination_url`, `attempts`, `max_attempts`, `last_error`, `updated_at`, `created_at`) without returning payload body.
- Notes: this endpoint enables manual recovery workflow selection.

### FR-004 - Manual DLQ requeue operation

- Description: service must allow operators to requeue failed webhook events.
- Acceptance criteria:
  - [x] AC1: `POST /v1/webhook-outbox/dlq/{event_id}/requeue` transitions only `failed` event to `pending`.
  - [x] AC2: requeue reset semantics are applied: `attempts=0`, `next_attempt_at=now`, `last_error=NULL`, `lease_owner=NULL`, `lease_until=NULL`, `updated_at=now`.
  - [x] AC3: requeue for non-failed event returns conflict error.
- Notes: this operation does not mutate event payload or destination URL.

### FR-005 - Manual cancel operation

- Description: service must allow operators to cancel webhook event delivery attempts.
- Acceptance criteria:
  - [x] AC1: `POST /v1/webhook-outbox/events/{event_id}/cancel` supports optional JSON body `{ "reason": "..." }`.
  - [x] AC2: cancel transitions `pending` or keeps `failed` as `failed`, clears lease ownership, and sets `last_error` to normalized manual-cancel reason.
  - [x] AC3: cancel for `delivered` event returns conflict error.
- Notes: cancel acts as explicit operator terminalization.

### FR-006 - API contracts and operator docs update

- Description: OpenAPI and README must document new webhook outbox operations.
- Acceptance criteria:
  - [x] AC1: OpenAPI includes overview, DLQ list, requeue, and cancel endpoints.
  - [x] AC2: README includes usage examples and behavior notes for manual compensation workflow.
- Notes: documentation must match implemented validation rules.

## Non-functional requirements

- Performance (NFR-001): overview and DLQ list queries must each use single SQL query path and avoid scanning payload text field in result projection.
- Availability/Reliability (NFR-002): manual operations must be single-row atomic updates with deterministic status guards.
- Security/Privacy (NFR-003): operator endpoints must not return webhook payload body by default.
- Compliance (NFR-004): no additional compliance controls introduced in this iteration.
- Observability (NFR-005): dispatcher logs include delivery bucket counters every cycle.
- Maintainability (NFR-006): feature is implemented via ports/use-cases/adapters with no domain import boundary violations.

## Dependencies and integrations

- External systems: webhook receivers indirectly affected by requeue/cancel operations.
- Internal services: webhook outbox PostgreSQL adapter, server HTTP router/controllers, dispatcher worker logging.
