---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- settlement evidence: one tracked on-chain observation row stored in `app.payment_request_settlements`.

## Out-of-scope behaviors

- OOS1: add settlement mutation APIs.
- OOS2: add pagination/query filters.

## Functional requirements

### FR-001 - Settlement list endpoint

- Description: service must expose settlement evidence for one payment request ID.
- Acceptance criteria:
  - [x] AC1: `GET /v1/payment-requests/{id}/settlements` is available.
  - [x] AC2: unknown `id` returns `404 payment_request_not_found`.
  - [x] AC3: known `id` returns `200` with JSON envelope containing `payment_request_id` and `settlements` array.
- Notes: endpoint is read-only and does not change request status.

### FR-002 - Settlement response contract

- Description: each returned settlement item must provide fields required for debugging reconciliation.
- Acceptance criteria:
  - [x] AC1: each item includes `evidence_ref`, `confirmations`, `is_canonical`, `block_height`, `block_hash`, `metadata`, `first_seen_at`, `last_seen_at`, `updated_at`.
  - [x] AC2: result ordering is deterministic (`first_seen_at` ascending, then `evidence_ref` ascending).
  - [x] AC3: when no rows exist yet for a known payment request, response contains empty `settlements` array.
- Notes: response can include existing `amount_minor` to preserve currently stored visibility.

### FR-003 - Architecture boundary compliance

- Description: implementation must follow existing layered architecture.
- Acceptance criteria:
  - [x] AC1: inbound HTTP adapter only parses path and maps to application query DTO.
  - [x] AC2: application use case validates input and delegates data fetch to outbound read model port.
  - [x] AC3: PostgreSQL read model implements query without leaking SQL concerns into application/domain.
- Notes: no domain rules are introduced in transport or persistence adapters.

## Non-functional requirements

- Performance (NFR-001): single request query should execute with one round-trip and stable ordering suitable for operator debugging.
- Availability/Reliability (NFR-002): new endpoint must not affect existing create/get/reconcile API behavior.
- Security/Privacy (NFR-003): endpoint must scope results strictly by requested payment request ID.
- Compliance (NFR-004): N/A for this incremental read feature.
- Observability (NFR-005): not-found and validation errors must use existing structured error envelope.
- Maintainability (NFR-006): code placement remains under current clean architecture and Go layout conventions.

## Dependencies and integrations

- External systems: none.
- Internal services: payment request read model, HTTP router/controller, OpenAPI spec.
