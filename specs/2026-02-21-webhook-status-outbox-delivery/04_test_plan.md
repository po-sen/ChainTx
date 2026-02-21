---
doc: 04_test_plan
spec_date: 2026-02-21
slug: webhook-status-outbox-delivery
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
  - 2026-02-21-min-confirmations-configurable
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
  - required request-level `webhook_url` validation.
  - webhook URL allowlist config parsing/enforcement.
  - outbox enqueue with destination snapshot.
  - dispatcher delivery/retry/fail using event-bound destination URL.
  - HMAC-SHA256 signature generation.
- Not covered:
  - partner endpoint SLA guarantees.
  - webhook replay tooling/UI.

## Tests

### Unit

- TC-001:
  - Linked requirements: FR-003 / NFR-003
  - Steps: execute signing helper with fixed secret/timestamp/body.
  - Expected: deterministic signature matches expected digest.
- TC-002:
  - Linked requirements: FR-006 / FR-007 / NFR-003
  - Steps: run create-payment-request use case validation with missing `webhook_url`, invalid URL, and non-allowlisted host.
  - Expected: explicit validation errors are returned.
- TC-003:
  - Linked requirements: FR-002 / NFR-002 / NFR-001
  - Steps: run dispatcher use case with fake repository + fake gateway for success/retry/terminal-fail using per-event URLs.
  - Expected: counters and repository state updates match expected transitions.
- TC-004:
  - Linked requirements: FR-007 / NFR-005
  - Steps: parse allowlist env with valid/invalid JSON and edge host patterns.
  - Expected: invalid format fails with explicit config error; valid format parsed correctly.

### Integration

- TC-101:
  - Linked requirements: FR-001 / FR-006 / FR-007 / NFR-002
  - Steps: persist payment request with `webhook_url`, trigger transition enqueue, inspect outbox row.
  - Expected: outbox row contains destination snapshot matching persisted payment request URL.
- TC-102:
  - Linked requirements: FR-002 / FR-004 / FR-005 / NFR-002
  - Steps: dispatcher cycle claims rows and sends to destination URL captured in outbox.
  - Expected: `2xx` -> delivered; failure -> retry schedule; max attempts -> failed.
- TC-103:
  - Linked requirements: FR-006 / FR-007 / NFR-003
  - Steps: API integration test for `POST /v1/payment-requests` with non-allowlisted `webhook_url`.
  - Expected: request rejected with validation error and no DB write.

### E2E (if applicable)

- Scenario 1:
  - Create payment request with valid allowlisted `webhook_url`, perform on-chain payment, verify webhook received and signature valid.
- Scenario 2:
  - Create payment request with non-allowlisted `webhook_url`, verify creation rejected immediately.

## Edge cases and failure modes

- Case: missing `webhook_url` on create.
- Expected behavior: reject request with validation error.
- Case: allowlist malformed.
- Expected behavior: startup fails with explicit config error.
- Case: allowlist not configured.
- Expected behavior: startup uses default local-safe allowlist values.
- Case: destination endpoint timeout/refusal.
- Expected behavior: outbox attempts increment and retry scheduled.
- Case: max attempts reached.
- Expected behavior: outbox row marked terminal `failed`.

## NFR verification

- Performance:
  - dispatch cycle remains bounded by batch and timeout settings.
- Reliability:
  - pending events survive restart and retries continue.
- Security:
  - HMAC secret not logged; destination restriction enforced by allowlist + scheme rules.
