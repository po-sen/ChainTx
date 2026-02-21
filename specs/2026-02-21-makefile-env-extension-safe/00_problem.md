---
doc: 00_problem
spec_date: 2026-02-21
slug: makefile-env-extension-safe
mode: Quick
status: DONE
owners:
  - posen
depends_on: []
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: null
  tasks: 03_tasks.md
  test_plan: null
---

# Problem & Goals

## Context

- Background: service startup rules currently contain inline env assignment blocks.
- Users or stakeholders: maintainers who frequently add env knobs for service/reconciler.
- Why now: user requested that future env additions should avoid editing Makefile rule bodies.

## Constraints (optional)

- Technical constraints: keep runtime behavior unchanged; this is a maintainability-only Makefile refactor.
- Timeline/cost constraints: single-file change expected (`Makefile`).
- Compliance/security constraints: no secret handling behavior changes.

## Problem statement

- Current pain: adding env mappings requires editing recipe commands under `service-up`.
- Evidence or examples: env growth repeatedly touches long command sections and increases merge risk.

## Goals

- G1: centralize service env mappings in top-level variables.
- G2: keep `service-up` and related rules stable when adding future env variables.

## Non-goals (out of scope)

- NG1: no functional changes to compose invocation or scripts.
- NG2: no changes to env names/semantics.

## Assumptions

- A1: maintainers accept adding new env mappings in variable definitions above rules.
- A2: `make -n` checks are sufficient to validate command wiring equivalence.

## Open questions

- Q1: none.
- Q2: none.

## Success metrics

- Metric: maintainability
- Target: future env additions only require edits in centralized variable section, not recipe body.
