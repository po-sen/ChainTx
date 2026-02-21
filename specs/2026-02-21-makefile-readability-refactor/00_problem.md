---
doc: 00_problem
spec_date: 2026-02-21
slug: makefile-readability-refactor
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

- Background: current `Makefile` service targets include very long single-line shell commands with many inline env vars.
- Users or stakeholders: maintainers who need to safely change service startup and smoke-test commands.
- Why now: user requested refactoring because long lines are hard to review and easy to break.

## Constraints (optional)

- Technical constraints: keep command behavior unchanged; this is readability-focused refactor only.
- Timeline/cost constraints: small scoped change in one file (`Makefile`) and no runtime feature changes.
- Compliance/security constraints: no new secrets or env semantics introduced.

## Problem statement

- Current pain: `service-up` and `local-receive-test` targets have very long one-liner commands that are difficult to diff and edit.
- Evidence or examples: large inline env assignment chains reduce maintainability and increase edit risk.

## Goals

- G1: split oversized shell/env lines into readable multiline blocks.
- G2: preserve all existing behavior and env wiring.

## Non-goals (out of scope)

- NG1: change runtime behavior, environment variable names, or target semantics.
- NG2: redesign build/deploy flow or introduce new scripts.

## Assumptions

- A1: multiline `env` formatting and shell continuation are acceptable for this repository style.
- A2: `make -n` command expansion checks are sufficient to validate unchanged wiring for this refactor.

## Open questions

- Q1: none for this scope.
- Q2: none for this scope.

## Success metrics

- Metric: maintainability/readability
- Target: no oversized one-line env chains remain in edited targets.
