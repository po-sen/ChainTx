---
doc: 02_design
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

# Technical Design

## High-level approach

- Summary: extend webhook dispatch command/config with jitter and retry budget, then apply both in use-case retry decision and `next_attempt_at` computation.
- Key decisions:
  - jitter control by `PAYMENT_REQUEST_WEBHOOK_RETRY_JITTER_BPS` (`0..10000`).
  - retry budget control by `PAYMENT_REQUEST_WEBHOOK_RETRY_BUDGET` (`0` disabled).
  - deterministic jitter factor per `(event_id, row_id, attempt)` via hash-based offset so behavior is stable and testable.

## System context

- Components:
  - `internal/infrastructure/config` parses new webhook runtime fields.
  - `internal/infrastructure/webhook/worker` passes new command fields.
  - `DispatchWebhookEventsUseCase` computes effective max attempts and jittered backoff.
- Interfaces:
  - `dto.DispatchWebhookEventsCommand` adds `RetryJitterBPS` and `RetryBudget`.

## Key flows

- Flow 1: retry under budget
  - compute `nextAttempts = row.attempts + 1`
  - compute effective max attempts using runtime budget
  - if under threshold, compute jittered backoff and call `MarkRetry`
- Flow 2: retry budget exhausted
  - if `nextAttempts >= effectiveMaxAttempts`, call `MarkFailed` directly
- Flow 3: jitter disabled
  - with jitter bps `0`, backoff remains existing exponential behavior

## Diagrams (optional)

```mermaid
flowchart TD
  A[Send failed] --> B[nextAttempts = attempts + 1]
  B --> C[effectiveMax = min(row.max_attempts, retry_budget+1 if budget>0)]
  C -->|nextAttempts >= effectiveMax| D[MarkFailed]
  C -->|nextAttempts < effectiveMax| E[base backoff]
  E --> F[jitter adjust]
  F --> G[MarkRetry(next_attempt_at)]
```

## Data model

- Entities: unchanged.
- Schema changes or migrations: none.
- Consistency and idempotency: unchanged outbox semantics; only retry timing and terminal threshold logic changes.

## API or contracts

- Endpoints or events: no HTTP API contract changes.
- Request/response examples:
  - new runtime env:
    - `PAYMENT_REQUEST_WEBHOOK_RETRY_JITTER_BPS=2000`
    - `PAYMENT_REQUEST_WEBHOOK_RETRY_BUDGET=3`

## Backward compatibility (optional)

- API compatibility: unchanged.
- Data migration compatibility: no migration required.

## Failure modes and resiliency

- Retries/timeouts:
  - invalid jitter range or negative budget fails fast during config load.
  - jittered backoff always clamped and positive.
- Backpressure/limits:
  - retry budget reduces runaway retries during prolonged endpoint failures.
- Degradation strategy:
  - defaults preserve previous behavior (jitter=0, budget=0).

## Observability

- Logs: existing cycle summaries remain; retry/failure counts naturally reflect budget behavior.
- Metrics: deferred.
- Traces: deferred.
- Alerts: deferred.

## Security

- Authentication/authorization: unchanged.
- Secrets: unchanged.
- Abuse cases: jitter/budget reduce synchronized retry load amplification risk.

## Alternatives considered

- Option A: full random jitter via shared RNG state.
- Option B: deterministic hash-based jitter.
- Why chosen: deterministic approach simplifies testing and avoids RNG synchronization concerns.

## Risks

- Risk: operators set budget too low, causing premature failures.
- Mitigation: document semantics clearly and keep default disabled.
