---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- Long env chain:
- A single command line embedding many environment assignments inline.

## Out-of-scope behaviors

- OOS1: changing deployment logic or adding/removing Make targets.
- OOS2: changing service runtime configuration keys/values.

## Functional requirements

### FR-001 - Refactor service-up command formatting for readability

- Description: `service-up` must be reformatted into multiline blocks while preserving existing behavior.
- Acceptance criteria:
  - [x] AC1: keyset generation command in `service-up` is split into readable multiline env assignments.
  - [x] AC2: compose startup env injection in `service-up` is reformatted from one-liner to multiline `env` block.
  - [x] AC3: sync-catalog invocation env injection in `service-up` is reformatted to multiline and remains behavior-compatible.
- Notes: no env names may be renamed in this refactor.

### FR-002 - Refactor local smoke target command formatting for readability

- Description: `local-receive-test` command env chain must be reformatted into multiline readable layout.
- Acceptance criteria:
  - [x] AC1: `local-receive-test` uses multiline `env` style with same variables and script call.
  - [x] AC2: command expansion via `make -n local-receive-test` remains valid.
- Notes: keep invocation path and variable wiring unchanged.

## Non-functional requirements

- Performance (NFR-001): no measurable execution overhead change from refactor.
- Availability/Reliability (NFR-002): no target execution regressions; dry-run output remains valid.
- Security/Privacy (NFR-003): no new secret exposure paths added.
- Compliance (NFR-004):
- Observability (NFR-005): not applicable for this refactor.
- Maintainability (NFR-006): edited targets must be clearly segmented for future changes.

## Dependencies and integrations

- External systems:
- Internal services:
