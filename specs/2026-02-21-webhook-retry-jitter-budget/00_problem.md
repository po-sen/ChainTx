---
doc: 00_problem
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

# Problem & Goals

## Context

- Background: webhook dispatcher already supports retry with bounded exponential backoff and at-least-once delivery.
- Users or stakeholders: operations and downstream webhook receivers.
- Why now: deterministic retries can synchronize across replicas/events; retry control needs an explicit runtime budget knob.

## Constraints (optional)

- Technical constraints: keep existing outbox table and status model; no schema migration for this scope.
- Timeline/cost constraints: incremental runtime/configuration hardening.
- Compliance/security constraints: no secret exposure in logs; preserve current signature contract behavior.

## Problem statement

- Current pain: retries use deterministic schedule without jitter, causing bursty redelivery patterns under endpoint failures.
- Current pain: retry ceiling is only persisted `max_attempts`; operators need runtime-level retry budget control without changing persisted rows.
- Evidence or examples: synchronized failure windows can produce clustered retries and thundering-herd load on receiver endpoints.

## Goals

- G1: add configurable retry jitter to webhook backoff scheduling.
- G2: add runtime retry budget to cap retries per event below persisted `max_attempts` when needed.
- G3: preserve existing delivered/retried/failed transition semantics and compatibility.
- G4: keep architecture boundaries clean (use case logic + config + worker wiring only).

## Non-goals (out of scope)

- NG1: per-destination adaptive circuit breaker.
- NG2: global token-bucket rate limiter across workers.
- NG3: webhook DLQ replay API/UI.

## Assumptions

- A1: retry budget is defined as max retry count per event (excluding initial attempt), applied at runtime.
- A2: jitter is deterministic per event-attempt to avoid requiring shared RNG state.

## Open questions

- Q1: retry jitter range settled as BPS percentage with clamp `0..10000`.
- Q2: retry budget default `0` means disabled (use persisted `max_attempts` only).

## Success metrics

- Metric: retry spread.
- Target: retry schedule for failed events is no longer strictly deterministic exponential at whole-base durations.
- Metric: operator control.
- Target: runtime config can lower effective retry ceiling without schema/data changes.
- Metric: compatibility.
- Target: existing webhook delivery lifecycle and observability remain intact.
