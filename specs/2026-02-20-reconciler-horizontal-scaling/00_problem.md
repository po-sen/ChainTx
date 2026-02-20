---
doc: 00_problem
spec_date: 2026-02-20
slug: reconciler-horizontal-scaling
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Problem & Goals

## Context

- Background: payment request reconciler currently runs inside API server process and fetches open rows with non-claiming query.
- Users or stakeholders: operators who need predictable reconciliation throughput and application developers who need safe horizontal scaling.
- Why now: user requested separate container and easy parallel expansion.

## Constraints (optional)

- Technical constraints: keep Clean Architecture boundaries and existing API behavior.
- Technical constraints: support multi-replica reconciler without duplicate row ownership in one cycle.
- Compliance/security constraints: no new secret storage and no key material leakage in logs.

## Problem statement

- Current pain: embedded worker couples API scale and reconcile scale.
- Current pain: multiple replicas can scan the same open rows because selection does not claim ownership before external RPC calls.
- Evidence or examples: existing query is `WHERE status IN (...) ORDER BY ... LIMIT ...` without lock/lease semantics.

## Goals

- G1: provide dedicated reconciler runtime entrypoint so it can run as separate container.
- G2: add DB-backed claim/lease mechanism for reconciliation batches to support safe horizontal worker scaling.
- G3: add simple deployment knobs to scale reconciler replicas in local/docker workflows.

## Non-goals (out of scope)

- NG1: blockchain confirmation-depth/reorg strategy redesign.
- NG2: replacing polling with websocket/event-stream listeners.
- NG3: introducing distributed queue infra (Kafka/Redis Streams/etc.).

## Assumptions

- A1: PostgreSQL is the source of truth and supports `FOR UPDATE SKIP LOCKED`.
- A2: one request should be claimed by at most one worker during an active lease window.

## Open questions

- Q1: resolved for this phase: lease timeout is fixed config (seconds), not adaptive.
- Q2: resolved for this phase: app process keeps optional embedded worker support, but recommended deploy mode is dedicated reconciler container.

## Success metrics

- Metric: horizontal scaling safety
- Target: with 3 replicas running, no duplicate claim ownership for the same request within a lease window.
- Metric: operability
- Target: one command can start service with configurable reconciler replicas.
- Metric: compatibility
- Target: existing API behavior unchanged when reconciler profile is disabled.
