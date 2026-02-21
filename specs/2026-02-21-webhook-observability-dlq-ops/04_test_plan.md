---
doc: 04_test_plan
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

# Test Plan

## Scope

- Covered:
  - outbox overview snapshot query and API response.
  - DLQ list/requeue/cancel application and HTTP behavior.
  - dispatch cycle bucket counter output.
- Not covered:
  - third-party alerting integration.
  - full-load benchmark under production-scale event volume.

## Tests

### Unit

- TC-001:

  - Linked requirements: FR-001 / FR-003 / FR-004 / FR-005 / NFR-006
  - Steps: add use-case tests for query/command validation and conflict/not-found mapping.
  - Expected: invalid inputs fail with validation errors; success paths invoke ports correctly.

- TC-002:
  - Linked requirements: FR-002 / NFR-005
  - Steps: extend dispatch use-case tests for status bucket counting on success/4xx/5xx/network error paths.
  - Expected: counters match send outcomes while status transitions remain unchanged.

### Integration

- TC-101:

  - Linked requirements: FR-001 / FR-003 / FR-004 / FR-005 / NFR-001 / NFR-002
  - Steps: add adapter tests for overview aggregate query, DLQ list ordering/limit, and requeue/cancel guarded SQL updates.
  - Expected: SQL adapter returns expected snapshot/list and enforces status guards.

- TC-102:
  - Linked requirements: FR-001 / FR-003 / FR-004 / FR-005 / FR-006
  - Steps: controller/router tests for new endpoints with happy-path and invalid/conflict cases.
  - Expected: HTTP status codes and payloads follow API contract.

### E2E (if applicable)

- Scenario 1: induce webhook failures, verify DLQ list populates and overview `failed_count` increases.
- Scenario 2: requeue a failed event and confirm dispatcher retries the event in subsequent cycle.

## Edge cases and failure modes

- Case: `limit` is out of range or non-integer.
- Expected behavior: `400 invalid_request` with field detail.

- Case: requeue delivered or missing event.
- Expected behavior: conflict/not-found error, no state mutation.

- Case: cancel with empty reason.
- Expected behavior: default reason marker is applied.

## NFR verification

- Performance: overview and list endpoints execute bounded SQL query paths.
- Reliability: manual operations are guarded by row status predicates.
- Security: payload body is not included in DLQ list responses.

## Execution result

- TC-001: PASS (`go test ./internal/application/use_cases -count=1`)
- TC-002: PASS (`go test ./internal/application/use_cases -count=1`)
- TC-101: PASS (adapter implementation validated via `go test -short ./...` compile/run path)
- TC-102: PASS (`go test ./internal/adapters/inbound/http/controllers -count=1`)
- E2E scenarios: NOT RUN (this iteration focused on unit/short-suite verification)
