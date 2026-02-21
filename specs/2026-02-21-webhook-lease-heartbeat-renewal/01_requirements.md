---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- lease heartbeat:
- Periodic extension of `lease_until` by the current `lease_owner` while processing is active.
- in-flight event:
- Claimed webhook outbox event currently being delivered by a worker.

## Out-of-scope behaviors

- OOS1: downstream dedupe store implementation.
- OOS2: introducing external distributed lock service.

## Functional requirements

### FR-001 - Worker renews lease during active delivery

- Description: dispatcher must renew lease for each claimed in-flight event while delivery call is still executing.
- Acceptance criteria:
  - [x] AC1: renewal starts before or at delivery start for each claimed event.
  - [x] AC2: renewal updates `lease_until` forward in DB using current `lease_owner` guard.
  - [x] AC3: renewal stops immediately after terminal event update (`delivered`, `retry`, `failed`, or skipped update).
- Notes: renewal must not block delivery loop progress for other events.

### FR-002 - Lease renewal is ownership-safe

- Description: renewal must only succeed when row is still owned by same worker and still in `pending` delivery state.
- Acceptance criteria:
  - [x] AC1: renewal query includes `id`, `lease_owner`, and `delivery_status='pending'` constraints.
  - [x] AC2: if ownership check fails, renewal exits for that event without mutating unrelated rows.
  - [x] AC3: final mark operations (`MarkDelivered/MarkRetry/MarkFailed`) keep existing ownership checks.
- Notes: this avoids cross-worker lease corruption.

### FR-003 - Heartbeat timing is configurable-safe

- Description: renewal cadence must be deterministic and validated relative to lease duration.
- Acceptance criteria:
  - [x] AC1: heartbeat interval is derived from lease duration and always >0.
  - [x] AC2: heartbeat interval is strictly less than lease duration.
  - [x] AC3: invalid cadence configuration path fails fast with explicit validation error.
- Notes: no new required env var in this iteration unless strictly needed.

### FR-004 - Crash recovery remains intact

- Description: if worker stops heartbeating (process crash/cancel), stale claimed rows remain reclaimable after `lease_until`.
- Acceptance criteria:
  - [x] AC1: no permanent lock rows are introduced.
  - [x] AC2: claim query behavior for expired leases remains unchanged.
  - [x] AC3: retry/failure counters and status transitions preserve existing semantics.
- Notes: heartbeat must not compromise existing recovery model.

### FR-005 - Observability for renewal behavior

- Description: dispatcher logs must expose renewal failures with enough context for debugging.
- Acceptance criteria:
  - [x] AC1: renewal failure logs include event id, worker id, and error code/message.
  - [x] AC2: renewal stop reason is distinguishable between normal completion and error/ownership loss.
- Notes: no sensitive payload or secret logging.

## Non-functional requirements

- Performance (NFR-001): renewal overhead must remain bounded; dispatch throughput degradation should be negligible for normal lease settings.
- Availability/Reliability (NFR-002): heartbeating should reduce duplicate in-flight processing probability under slow endpoints.
- Security/Privacy (NFR-003): no new secret surfaces; renewal logs must not include webhook body/signature secret.
- Compliance (NFR-004): not applicable in this iteration.
- Observability (NFR-005): renewal-related failures are visible in logs with actionable identifiers.
- Maintainability (NFR-006): renewal logic must stay behind application port/repository boundaries (no SQL in worker/use case directly).

## Dependencies and integrations

- External systems: webhook endpoints with variable latency.
- Internal services: webhook dispatcher use case, webhook outbox repository adapter, worker runtime.
