---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- retry jitter:
- Percentage randomization window around computed exponential backoff duration.
- retry budget:
- Runtime cap on number of retries per event (excluding initial delivery attempt).

## Out-of-scope behaviors

- OOS1: cross-worker shared retry token bucket.
- OOS2: destination-specific dynamic backoff policy.

## Functional requirements

### FR-001 - Configurable retry jitter for webhook backoff

- Description: webhook dispatcher must apply jitter to retry delay derived from exponential backoff.
- Acceptance criteria:
  - [x] AC1: dispatcher supports config `PAYMENT_REQUEST_WEBHOOK_RETRY_JITTER_BPS` with valid range `0..10000`.
  - [x] AC2: when jitter is `0`, backoff remains current deterministic behavior.
  - [x] AC3: when jitter > `0`, next attempt delay is adjusted within `base * (1 ± jitter_bps/10000)` and clamped to `(0, max_backoff]`.
- Notes: jitter should avoid synchronized retries while keeping bounded backoff.

### FR-002 - Runtime retry budget per event

- Description: dispatcher must support runtime retry budget to lower effective max retries per event.
- Acceptance criteria:
  - [x] AC1: dispatcher supports config `PAYMENT_REQUEST_WEBHOOK_RETRY_BUDGET` as non-negative integer.
  - [x] AC2: `retry_budget = 0` means disabled (use persisted `max_attempts` only).
  - [x] AC3: `retry_budget > 0` sets effective max attempts to `min(row.max_attempts, retry_budget + 1)`; above threshold transitions to `failed`.
- Notes: this is a runtime safeguard; does not mutate stored `max_attempts`.

### FR-003 - Preserve webhook lifecycle compatibility

- Description: jitter/budget hardening must not break current status transitions and lease behavior.
- Acceptance criteria:
  - [x] AC1: successful `2xx` flow still marks event `delivered`.
  - [x] AC2: retry path still uses `MarkRetry` with bounded `next_attempt_at` when under effective attempt limit.
  - [x] AC3: terminal path still uses `MarkFailed` when reaching effective attempt limit.
- Notes: lease heartbeat behavior remains unchanged.

### FR-004 - Runtime wiring and docs update

- Description: new jitter/budget settings must be wired from config through worker command and documented.
- Acceptance criteria:
  - [x] AC1: config parsing/wiring exposes jitter and budget to dispatch command.
  - [x] AC2: README configuration table includes both new env vars and behavior.
- Notes: default values must preserve previous behavior unless explicitly configured.

## Non-functional requirements

- Performance (NFR-001): jitter computation overhead must be negligible relative to network I/O.
- Availability/Reliability (NFR-002): jitter should reduce synchronized retry bursts during endpoint failure windows.
- Security/Privacy (NFR-003): no new secret or sensitive payload logging.
- Compliance (NFR-004): not applicable in this iteration.
- Observability (NFR-005): retry behavior remains debuggable via attempt counts and status transitions.
- Maintainability (NFR-006): changes remain in config/runtime/use-case layers without violating architecture boundaries.

## Dependencies and integrations

- External systems: webhook receivers affected by retry cadence.
- Internal services: config loader, webhook worker, dispatch use case.
