---
doc: 04_test_plan
spec_date: 2026-02-21
slug: webhook-idempotency-replay-signature
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-status-outbox-delivery
  - 2026-02-21-webhook-lease-heartbeat-renewal
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
  - Delivery-attempt metadata propagation from use case to gateway input.
  - Outbound webhook header contract for idempotency and anti-replay v1 signature.
  - README receiver contract update checks.
- Not covered:
  - Receiver-side database/cache implementation for nonce replay storage.
  - Retry jitter/budget behavior.

## Tests

### Unit

- TC-001:
  - Linked requirements: FR-001 / FR-003 / NFR-005
  - Steps: execute dispatch use-case tests and assert `DeliveryAttempt` passed to gateway as `attempts + 1`.
  - Expected: first attempt sends `1`; retries send incremented values without changing event id.
- TC-002:
  - Linked requirements: FR-002 / FR-003 / NFR-001 / NFR-003
  - Steps: execute webhook gateway tests; assert headers `X-ChainTx-Nonce`, `X-ChainTx-Signature-Version`, `X-ChainTx-Signature-V1`, and legacy `X-ChainTx-Signature`.
  - Expected: v1 signature matches canonical payload and legacy signature remains present.

### Integration

- TC-101:
  - Linked requirements: FR-001 / FR-002 / FR-003 / NFR-002
  - Steps: run dispatcher flow against local test receiver (httptest) and inspect one successful request headers.
  - Expected: idempotency and replay headers are present in real request path.

### E2E (if applicable)

- Scenario 1: local webhook receiver processes one event then receives retry with same `Idempotency-Key`; receiver keeps idempotent outcome.
- Scenario 2: replayed raw request with same nonce is rejected by receiver-side nonce policy (documented procedure).

## Edge cases and failure modes

- Case: empty/invalid delivery attempt passed to gateway.
- Expected behavior: gateway coerces to minimum attempt `1` in header output.
- Case: empty event id.
- Expected behavior: existing request validation path still returns structured error.

## NFR verification

- Performance: no noticeable overhead in existing short test suite runtime.
- Reliability: new headers exist on every send path (success + non-2xx tests).
- Security: no secret leakage; signature contract includes timestamp + nonce + event identity.

## Execution result

- TC-001: PASS (`TestDispatchWebhookEventsUseCasePassesDeliveryAttempt`)
- TC-002: PASS (`TestSendWebhookEventSuccess`, `TestSendWebhookEventUsesDefaultAttemptHeader`, `TestSendWebhookEventRequiresEventID`, `TestSendWebhookEventRequiresEventType`)
- TC-101: PASS (httptest-based gateway integration path covered by `TestSendWebhookEventSuccess`)
- E2E scenarios: NOT RUN (documented runbook guidance only in this iteration).
