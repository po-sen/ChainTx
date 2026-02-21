---
doc: 00_problem
spec_date: 2026-02-21
slug: webhook-ops-admin-auth-audit
mode: Full
status: DONE
owners:
  - posen
depends_on:
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

- Background: webhook outbox operation endpoints were added for overview, DLQ listing, requeue, and cancel.
- Users or stakeholders: operators/SRE and security reviewers.
- Why now: user requested priority #1 hardening to protect ops-grade endpoints and track who performed manual actions.

## Constraints (optional)

- Technical constraints: preserve existing architecture boundaries and avoid breaking current payment/webhook flows.
- Timeline/cost constraints: complete within current iteration and verify with full test + receive/webhook run.
- Compliance/security constraints: manual mutation operations must be authenticated and auditable.

## Problem statement

- Current pain: webhook outbox ops endpoints are callable without admin authentication.
- Current pain: manual requeue/cancel does not persist operator identity in DB audit fields.
- Evidence or examples: anyone with API network access can mutate failed webhook events without actor trace.

## Goals

- G1: enforce admin authentication for all `/v1/webhook-outbox/*` endpoints.
- G2: persist manual operation audit data (`action`, `actor`, `timestamp`) for requeue/cancel.
- G3: standardize webhook ops auth to Bearer-only and remove legacy custom-header ambiguity.
- G4: keep existing domain/application flow stable while adding security controls.
- G5: execute full validation including unit/short suite and local receive/webhook runtime proof.

## Non-goals (out of scope)

- NG1: mTLS rollout and certificate management.
- NG2: full RBAC/permissions matrix beyond single admin key contract.
- NG3: webhook receiver-side auth redesign.

## Assumptions

- A1: API key-based admin auth is acceptable for this iteration.
- A2: operator identity is supplied via request header and stored in outbox audit fields.

## Open questions

- Q1: none in this iteration.

## Success metrics

- Metric: endpoint protection.
- Target: unauthenticated requests to webhook outbox endpoints are rejected.
- Metric: auditability.
- Target: successful requeue/cancel writes actor/action/timestamp in DB row.
- Metric: regression safety.
- Target: full go verification and local receive + webhook flow pass.
