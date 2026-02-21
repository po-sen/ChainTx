---
doc: 00_problem
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

# Problem & Goals

## Context

- Background: webhook outbox has delivery workflow and observability APIs, but threshold alerting initially caused multi-replica duplicated signals when coupled inside dispatcher.
- Users or stakeholders: operators running multi-container deployments.
- Why now: user requested clean runtime split and clearer container responsibility.

## Constraints (optional)

- Technical constraints: keep strict Clean Architecture boundaries and avoid DB migration.
- Timeline/cost constraints: incremental refactor over existing webhook alert capability.
- Compliance/security constraints: alert logs must not include payload body or secrets.

## Problem statement

- Current pain (before merge): alert threshold logic needed to exist, but should not run in every dispatcher replica.
- Current pain (before merge): alert env scope across all containers was noisy and confusing.
- Evidence or examples: duplicated `triggered/ongoing/resolved` logs across dispatcher replicas.

## Goals

- G1: support webhook alert thresholds with validated env contract.
- G2: run alert evaluation in dedicated `webhook-alert-worker` runtime only.
- G3: keep dispatcher runtime focused on delivery only.
- G4: keep alert lifecycle semantics (`triggered/ongoing/resolved`) and non-blocking failure behavior.
- G5: align deployment/docs with split runtime model.

## Non-goals (out of scope)

- NG1: external alert sink integration (PagerDuty/Slack/email).
- NG2: persistent alert state in DB.
- NG3: changes to webhook delivery status machine.

## Assumptions

- A1: `webhook-alert-worker` default replica count is `1`.
- A2: alert poll interval can reuse webhook poll interval setting.

## Open questions

- Q1: none for this iteration.

## Success metrics

- Metric: signal correctness.
- Target: alert logs emitted by dedicated alert worker with cooldown behavior.
- Metric: runtime isolation.
- Target: dispatcher logs no longer contain alert evaluation output.
- Metric: deployment clarity.
- Target: alert env scoped to `webhook-alert-worker` container.
