---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- admin key: server-side configured key used to authorize webhook ops endpoints.
- operator id: caller identity supplied in `X-Principal-ID` for manual mutation auditing.

## Out-of-scope behaviors

- OOS1: dynamic key management integration with external secret managers.
- OOS2: per-endpoint role-based authorization policies.

## Functional requirements

### FR-001 - Admin auth for webhook ops endpoints

- Description: all `/v1/webhook-outbox/*` endpoints must require valid admin authentication.
- Acceptance criteria:
  - [x] AC1: request must provide valid admin key via `Authorization: Bearer <key>`.
  - [x] AC2: request using only legacy `X-ChainTx-Admin-Key` is rejected with `401` structured error envelope.
  - [x] AC3: missing or invalid Bearer token returns `401` with structured error envelope.
  - [x] AC4: when admin keys are not configured, endpoint fails closed (`503`).
- Notes: applies to overview, DLQ list, requeue, and cancel endpoints.

### FR-002 - Operator identity required for manual mutations

- Description: requeue/cancel must require caller identity for audit recording.
- Acceptance criteria:
  - [x] AC1: `POST /v1/webhook-outbox/dlq/{event_id}/requeue` requires non-empty `X-Principal-ID`.
  - [x] AC2: `POST /v1/webhook-outbox/events/{event_id}/cancel` requires non-empty `X-Principal-ID`.
  - [x] AC3: missing `X-Principal-ID` returns `400 invalid_request`.
- Notes: read-only endpoints (overview/list) do not require operator id.

### FR-003 - Persist manual action audit fields in webhook outbox

- Description: successful requeue/cancel operations must persist audit metadata in DB.
- Acceptance criteria:
  - [x] AC1: schema includes nullable columns for manual action (`requeue|cancel`), actor, and action timestamp.
  - [x] AC2: successful requeue stores action=`requeue`, actor from `X-Principal-ID`, action timestamp=`now`.
  - [x] AC3: successful cancel stores action=`cancel`, actor from `X-Principal-ID`, action timestamp=`now`.
- Notes: existing delivery status semantics remain unchanged.

### FR-004 - Docs and runtime config contract update

- Description: operator auth/audit configuration and endpoint contract must be documented.
- Acceptance criteria:
  - [x] AC1: config docs include admin key env format and Bearer auth usage.
  - [x] AC2: OpenAPI declares webhook ops Bearer security scheme and mutation actor header requirement.
  - [x] AC3: Swagger UI supports `Authorize` flow for webhook ops with persisted auth in session.
- Notes: docs must match implementation behavior.

## Non-functional requirements

- Performance (NFR-001): auth check overhead must be negligible (header parsing + constant-time comparisons).
- Availability/Reliability (NFR-002): auth/audit addition must not break webhook dispatch or payment reconciliation flows.
- Security/Privacy (NFR-003): auth comparison uses constant-time check; secrets are never logged.
- Compliance (NFR-004): audit trail captures who/when for manual outbox mutations.
- Observability (NFR-005): auth failures produce structured API error responses.
- Maintainability (NFR-006): implementation stays within adapters/application/infrastructure boundaries.

## Dependencies and integrations

- External systems: webhook receiver test endpoint for runtime verification.
- Internal services: config parser, HTTP webhook outbox controller, PostgreSQL webhook outbox repository.
