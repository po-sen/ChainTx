---
doc: 04_test_plan
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

# Test Plan

## Scope

- Covered:
  - webhook ops admin auth behavior (401/503/success).
  - manual mutation operator-id validation and audit persistence wiring.
  - migration compatibility and full go verification.
  - local payment receive + webhook delivery runtime proof.
- Not covered:
  - external secret manager integration.

## Tests

### Unit

- TC-001:

  - Linked requirements: FR-001 / NFR-003 / NFR-005
  - Steps: controller auth tests for missing/invalid/valid Bearer token, legacy header-only request, and missing config.
  - Expected: 401 for invalid auth and legacy-header-only requests, 503 for unconfigured keys, success when valid Bearer token.

- TC-002:
  - Linked requirements: FR-002 / FR-003 / NFR-004
  - Steps: use-case tests ensure missing operator id is rejected and mutation commands carry operator id.
  - Expected: mutation use cases return validation errors when operator id missing.

### Integration

- TC-101:

  - Linked requirements: FR-003 / NFR-002 / NFR-004
  - Steps: execute requeue/cancel flow and verify outbox row audit columns updated.
  - Expected: `manual_last_action/manual_last_actor/manual_last_at` set on successful mutation.

- TC-102:
  - Linked requirements: FR-004 / NFR-006
  - Steps: verify OpenAPI/README reflect Bearer-only auth and Swagger security scheme contract.
  - Expected: docs are consistent with endpoint behavior.

### E2E (if applicable)

- Scenario 1: local receive smoke for BTC/ETH/USDT passes after feature changes.
- Scenario 2: webhook receiver receives signed events; unauthorized webhook-outbox ops call is rejected.

## Edge cases and failure modes

- Case: `Authorization` header malformed.
- Expected behavior: 401 unauthorized.

- Case: valid admin key but missing `X-Principal-ID` on mutation.
- Expected behavior: 400 invalid request.

## NFR verification

- Performance: auth check is O(number of keys), key list expected small.
- Reliability: existing receive/reconcile/webhook flows remain passing.
- Security: keys not logged, unauthenticated ops are rejected.
