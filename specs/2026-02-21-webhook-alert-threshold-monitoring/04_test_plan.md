---
doc: 04_test_plan
spec_date: 2026-02-21
slug: webhook-alert-threshold-monitoring
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-status-outbox-delivery
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
  - alert env parsing/defaults/validation.
  - alert lifecycle state machine and cooldown.
  - dedicated alert runtime behavior and failure isolation.
  - dispatcher decoupling from alert evaluation.
  - deployment profile/env scope validation.
- Not covered:
  - external alert channel integration.
  - durable alert state across restart.

## Tests

### Unit

- TC-001: config default and validation behavior

  - Linked requirements: FR-001 / NFR-002
  - Steps:
    - load config with alert env absent.
    - load config with invalid alert env values.
  - Expected:
    - defaults applied; invalid values fail with explicit config errors.

- TC-002: alert monitor lifecycle transition

  - Linked requirements: FR-003 / NFR-005
  - Steps:
    - feed snapshot sequence crossing threshold, staying breached across cooldown windows, then recovering.
  - Expected:
    - event sequence is `triggered -> ongoing -> resolved` with cooldown suppression.

- TC-003: alert worker failure isolation
  - Linked requirements: FR-004 / NFR-002
  - Steps:
    - simulate overview query app error across multiple cycles.
  - Expected:
    - each cycle logs evaluation failure and loop continues.

### Integration

- TC-101: webhook-alert-worker runtime validation

  - Linked requirements: FR-002
  - Steps:
    - run runtime config validator with enabled/disabled cases.
  - Expected:
    - disabled returns `CONFIG_WEBHOOK_ALERT_DISABLED`; enabled passes.

- TC-102: dispatcher alert decoupling

  - Linked requirements: FR-005 / NFR-006
  - Steps:
    - run webhook dispatcher worker tests.
  - Expected:
    - no overview/alert dependency in dispatcher code path.

- TC-103: deployment profile split
  - Linked requirements: FR-006 / NFR-006
  - Steps:
    - run `make -n service-up SERVICE_WEBHOOK_ENABLED=true SERVICE_WEBHOOK_ALERT_ENABLED=true`.
  - Expected:
    - includes `webhook-alert-worker` profile and scale args.

### E2E (if applicable)

- Scenario 1:
  - start stack with `webhook-dispatcher` + `webhook-alert-worker`.
  - verify alert logs appear only in `webhook-alert-worker` logs.

## Edge cases and failure modes

- Case: threshold is `0`.
- Expected behavior: signal disabled and skipped.

- Case: no pending rows (`oldest_pending_age_seconds=nil`).
- Expected behavior: age signal remains healthy.

## NFR verification

- Performance: one overview query per alert poll cycle.
- Reliability: query failure does not terminate loop.
- Security: no payload/secret fields in alert logs.
