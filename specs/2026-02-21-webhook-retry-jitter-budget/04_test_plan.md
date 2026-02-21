---
doc: 04_test_plan
spec_date: 2026-02-21
slug: webhook-retry-jitter-budget
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-lease-heartbeat-renewal
  - 2026-02-21-webhook-idempotency-replay-signature
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
  - config parse/validation for retry jitter and retry budget.
  - use-case retry/failed branch behavior with runtime budget.
  - jittered backoff bounds and disabled behavior.
  - README config documentation update.
- Not covered:
  - live multi-replica load benchmark.
  - adaptive rate limiting.

## Tests

### Unit

- TC-001:
  - Linked requirements: FR-001 / FR-002 / FR-004 / NFR-006
  - Steps: add config tests for valid/invalid jitter and retry budget envs.
  - Expected: valid values load; out-of-range or negative values return config errors.
- TC-002:
  - Linked requirements: FR-002 / FR-003 / NFR-005
  - Steps: use-case tests where runtime budget triggers `MarkFailed` before row `max_attempts`.
  - Expected: event transitions to `failed` when reaching effective budget threshold.
- TC-003:
  - Linked requirements: FR-001 / FR-003 / NFR-001 / NFR-002
  - Steps: use-case/jitter helper tests verify jitter disabled path equals base backoff and jitter enabled path stays within expected bounds.
  - Expected: jittered backoff is bounded and positive.

### Integration

- TC-101:
  - Linked requirements: FR-004 / NFR-006
  - Steps: webhook worker run-cycle test or wiring test ensures config fields are propagated to dispatch command.
  - Expected: command contains configured jitter and budget values.

### E2E (if applicable)

- Scenario 1: endpoint forced to fail; observe varied retry schedule (not fixed exact exponential) with jitter enabled.
- Scenario 2: set small retry budget and verify earlier terminal `failed` transition.

## Edge cases and failure modes

- Case: jitter bps = `0`.
- Expected behavior: exact legacy backoff behavior.
- Case: retry budget = `0`.
- Expected behavior: use persisted `max_attempts` only.

## NFR verification

- Performance: jitter math/hash adds negligible overhead.
- Reliability: retry bursts are less synchronized with jitter.
- Security: no secret-sensitive data added to logs.

## Execution result

- TC-001: PASS (`TestLoadConfigParsesWebhookConfig`, `TestLoadConfigRejectsInvalidWebhookRetryJitterBPS`, `TestLoadConfigRejectsInvalidWebhookRetryBudget`)
- TC-002: PASS (`TestDispatchWebhookEventsUseCaseFailsWhenRetryBudgetReached`)
- TC-003: PASS (`TestWebhookRetryBackoffWithJitterDisabledMatchesBase`, `TestWebhookRetryBackoffWithJitterWithinBounds`)
- TC-101: PASS (`TestWorkerRunsCycleWithRetryConfig`)
- E2E scenarios: NOT RUN (unit/integration scope in this iteration).
