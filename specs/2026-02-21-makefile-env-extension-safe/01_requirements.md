---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- Centralized env mapping:
- Top-level Makefile variable blocks that contain env assignments used by rules.

## Out-of-scope behaviors

- OOS1: change service runtime configuration values.
- OOS2: add/remove make targets.

## Functional requirements

### FR-001 - Centralize env assignments outside rule body

- Description: env assignment lists used by `service-up` must be moved into reusable top-level variables.
- Acceptance criteria:
  - [x] AC1: `service-up` keyset generation env list is sourced from a top-level variable.
  - [x] AC2: `service-up` compose env list is sourced from a top-level variable.
  - [x] AC3: `service-up` sync-catalog env list is sourced from a top-level variable.
- Notes: behavior must remain equivalent.

### FR-002 - Future env extension avoids rule edits

- Description: adding a new service env mapping should require touching only centralized variables, not recipe body.
- Acceptance criteria:
  - [x] AC1: `service-up` recipe lines remain generic and reference env variables.
  - [x] AC2: documentation-by-structure is clear (variable naming indicates where to add future envs).
- Notes: enforced by Makefile layout, not runtime logic.

## Non-functional requirements

- Performance (NFR-001): no runtime overhead change.
- Availability/Reliability (NFR-002): command expansion remains valid (`make -n service-up`).
- Security/Privacy (NFR-003): no new env leaks or secret output changes.
- Compliance (NFR-004):
- Observability (NFR-005): not applicable.
- Maintainability (NFR-006): env mapping locations are explicit and isolated from rule body.

## Dependencies and integrations

- External systems:
- Internal services:
