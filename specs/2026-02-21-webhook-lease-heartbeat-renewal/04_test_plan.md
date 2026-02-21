---
doc: 04_test_plan
spec_date: 2026-02-21
slug: webhook-lease-heartbeat-renewal
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-status-outbox-delivery
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
  - lease renewal SQL guard behavior.
  - heartbeat lifecycle around webhook send attempts.
  - compatibility of delivered/retry/failed transitions with heartbeat enabled.
- Not covered:
  - downstream consumer idempotency storage implementation.
  - high-volume performance benchmark suite.

## Tests

### Unit

- TC-001:
  - Linked requirements: FR-001 / FR-003 / NFR-001
  - Steps: test heartbeat interval derivation and validation against lease duration.
  - Expected: interval is positive, less than lease, and deterministic.
- TC-002:
  - Linked requirements: FR-001 / FR-005 / NFR-005
  - Steps: simulate long-running send in dispatch use case and assert renewal callback is invoked while send is blocking.
  - Expected: renewal happens at least once before send completes; heartbeat stops after final mark.

### Integration

- TC-101:
  - Linked requirements: FR-002 / FR-004 / NFR-006
  - Steps: repository integration for `RenewLease` with matching and mismatching owner/state.
  - Expected: only owned pending row is renewed; mismatches return false.
- TC-102:
  - Linked requirements: FR-001 / FR-004 / NFR-002
  - Steps: multi-worker simulation where one worker heartbeats during long send and second worker tries claim.
  - Expected: second worker cannot claim before lease expires when heartbeat is active.

### E2E (if applicable)

- Scenario 1: run local dispatcher + slow/unreachable endpoint and verify event reaches expected terminal state without concurrent duplicate claim while heartbeat active.
- Scenario 2: stop worker mid-attempt and verify stale lease becomes reclaimable after expiry.

## Edge cases and failure modes

- Case: renewal DB error during send.
- Expected behavior: renewal failure is logged with context; existing send/mark path still executes.
- Case: ownership lost before renewal tick.
- Expected behavior: renewal returns false and loop stops for that event.

## NFR verification

- Performance: renewal write frequency remains bounded by derived interval.
- Reliability: active long send attempts keep lease alive; crash recovery still works via lease expiry.
- Security: logs include identifiers and error codes only; no secret or payload leakage.

## Execution result

- TC-001: PASS (`TestDispatchWebhookEventsUseCaseRejectsLeaseTooSmallForHeartbeat`)
- TC-002: PASS (`TestDispatchWebhookEventsUseCaseRenewsLeaseDuringSlowSend`)
- Additional unit coverage:
  - PASS (`TestDispatchWebhookEventsUseCaseReturnsRenewLeaseError`)
  - PASS (`TestDispatchWebhookEventsUseCaseReturnsLeaseLostWhenRenewReturnsFalse`)
- TC-101: PARTIAL (no dedicated repository test harness in this iteration; SQL guard validated by adapter implementation review and package build).
- TC-102: PARTIAL (no multi-worker integration harness in this iteration; heartbeat behavior covered at use-case level).
