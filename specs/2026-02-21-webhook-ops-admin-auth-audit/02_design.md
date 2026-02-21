---
doc: 02_design
spec_date: 2026-02-21
slug: webhook-ops-admin-auth-audit
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-observability-dlq-ops
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Technical Design

## High-level approach

- Summary: add admin key config + controller auth guard for webhook outbox endpoints and add DB audit columns written by manual mutation SQL paths.
- Key decisions:
  - Admin key config format: JSON string array env for key rotation support.
  - Auth header support: Bearer-only (`Authorization: Bearer <key>`); no custom header fallback.
  - Audit persistence: extend outbox table with `manual_last_action`, `manual_last_actor`, `manual_last_at`.
  - OpenAPI auth contract: webhook ops endpoints use `securitySchemes` (`http` + `bearer`) so Swagger `Authorize` auto-wires the header.

## System context

- Components:
  - `internal/infrastructure/config`: parse admin keys env.
  - `internal/adapters/inbound/http/controllers/webhook_outbox_controller.go`: enforce auth and actor header validation.
  - `internal/application/dto` + `use_cases`: include operator id in requeue/cancel commands.
  - `internal/adapters/outbound/persistence/postgresql/webhookoutbox`: write audit fields during requeue/cancel.
  - `internal/.../migrations`: add audit columns.
- Interfaces:
  - `WebhookOutboxRepository.RequeueFailedByEventID/CancelByEventID` extended with operator id.

## Key flows

- Flow 1: authenticated overview/list request
  - controller extracts Bearer token from `Authorization`
  - controller authenticates token against configured admin keys
  - authorized request proceeds to read use case
- Flow 2: authenticated requeue/cancel request
  - controller authenticates admin key
  - controller validates non-empty `X-Principal-ID`
  - use case invokes repository mutation with operator id
  - repository updates delivery fields + audit fields atomically

## Data model

- Entities: unchanged.
- Schema changes or migrations:
  - add `manual_last_action text`
  - add `manual_last_actor text`
  - add `manual_last_at timestamptz`
  - add check constraint for action enum if column non-null
- Consistency and idempotency: existing status guards remain; audit fields only update on successful mutation.

## API or contracts

- Endpoints or events: existing webhook outbox endpoints remain, now with auth requirement.
- Request/response examples:
  - `Authorization: Bearer <admin-key>`
  - `X-Principal-ID: ops-user-001` for requeue/cancel.
  - OpenAPI `WebhookOpsBearerAuth` is required for `/v1/webhook-outbox/*`.

## Failure modes and resiliency

- Retries/timeouts: unchanged dispatcher behavior.
- Backpressure/limits: unchanged.
- Degradation strategy: missing key config returns `503`; invalid key returns `401`.

## Observability

- Logs: controller request error log entries for auth/validation failures (without leaking keys).
- Metrics: no new metrics in this iteration.

## Security

- Authentication/authorization: API key auth for webhook outbox endpoints.
- Secrets: admin keys loaded from env only, never logged.
- Abuse cases: unauthenticated mutation requests rejected before use-case execution.

## Alternatives considered

- Option A: startup fail if no admin keys configured.
- Option B: endpoint-level fail-closed (`503`) when keys missing.
- Why chosen: avoids blocking unrelated runtime startup while still securing ops endpoints.
- Option C: keep legacy custom header fallback (`X-ChainTx-Admin-Key`).
- Why not chosen: single Bearer contract is simpler and avoids operator confusion.

## Risks

- Risk: key leakage in operator usage.
- Mitigation: support key rotation list and recommend secret manager-backed env injection.
