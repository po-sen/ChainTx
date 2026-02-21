---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- webhook-alert-worker: dedicated runtime process for outbox threshold alert evaluation.
- alert signal: one monitored field (`failed_count`, `pending_ready_count`, `oldest_pending_age_seconds`).

## Out-of-scope behaviors

- OOS1: third-party notification channels.
- OOS2: durable alert state across process restart.

## Functional requirements

### FR-001 - Alert env configuration contract

- Description: service supports validated alert configuration via env.
- Acceptance criteria:
  - [x] AC1: support `PAYMENT_REQUEST_WEBHOOK_ALERT_ENABLED`, `PAYMENT_REQUEST_WEBHOOK_ALERT_COOLDOWN_SECONDS`, `PAYMENT_REQUEST_WEBHOOK_ALERT_FAILED_COUNT_THRESHOLD`, `PAYMENT_REQUEST_WEBHOOK_ALERT_PENDING_READY_THRESHOLD`, `PAYMENT_REQUEST_WEBHOOK_ALERT_OLDEST_PENDING_AGE_SECONDS`.
  - [x] AC2: defaults are applied when absent (`enabled=false`, `cooldown=300s`, thresholds `0` => disabled signal).
  - [x] AC3: invalid values fail startup with explicit config error code/message (invalid bool, non-positive cooldown, negative threshold, enabled without any threshold > 0).
- Notes: threshold `0` disables that signal.

### FR-002 - Dedicated alert runtime

- Description: alert evaluation runs in separate `webhook-alert-worker` runtime.
- Acceptance criteria:
  - [x] AC1: add `cmd/webhook-alert-worker` entrypoint with persistence init and runtime config validation.
  - [x] AC2: DI provides dedicated `BuildWebhookAlertWorker` container.
  - [x] AC3: alert runtime exits when alert feature is disabled.
- Notes: runtime uses existing outbox overview use case.

### FR-003 - Alert lifecycle behavior

- Description: alert runtime evaluates thresholds each poll cycle and emits lifecycle logs.
- Acceptance criteria:
  - [x] AC1: threshold breach logs `webhook alert triggered`.
  - [x] AC2: sustained breach logs `webhook alert ongoing` at most once per cooldown window.
  - [x] AC3: recovery logs `webhook alert resolved`.
- Notes: per-signal lifecycle state is in-memory.

### FR-004 - Non-blocking failure handling

- Description: alert query/evaluation failures must not stop runtime loop.
- Acceptance criteria:
  - [x] AC1: overview query failure logs `webhook alert evaluation failed` and continues next cycle.
  - [x] AC2: dispatcher delivery semantics remain unchanged.
- Notes: alerting is best-effort.

### FR-005 - Dispatcher decoupling

- Description: webhook dispatcher handles delivery only.
- Acceptance criteria:
  - [x] AC1: dispatcher worker does not evaluate alert thresholds.
  - [x] AC2: dispatcher no longer depends on alert monitor and overview use case.
- Notes: avoids per-replica duplicate alert evaluation.

### FR-006 - Deployment and docs alignment

- Description: deployment config and docs reflect split runtime model.
- Acceptance criteria:
  - [x] AC1: docker compose includes `webhook-alert-worker` profile/service.
  - [x] AC2: Makefile supports independent alert worker scaling/config.
  - [x] AC3: README documents alert runtime split and usage examples.
- Notes: alert env should only be injected to alert worker container.

## Non-functional requirements

- Performance (NFR-001): alert runtime performs one overview query per poll cycle.
- Availability/Reliability (NFR-002): query failures do not terminate alert loop.
- Security/Privacy (NFR-003): logs never include payload body or secret values.
- Compliance (NFR-004): no new compliance scope.
- Observability (NFR-005): alert logs include worker id, signal, current, threshold, cooldown.
- Maintainability (NFR-006): dispatch worker and alert worker are independently testable and decoupled.

## Dependencies and integrations

- External systems: log collectors.
- Internal services: config loader, DI composition root, webhook outbox overview use case/read model, docker compose and make orchestration.
