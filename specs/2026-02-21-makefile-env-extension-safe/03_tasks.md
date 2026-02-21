---
doc: 03_tasks
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

# Task Plan

## Mode decision

- Selected mode: Quick
- Rationale: Makefile-only maintainability refactor with no functional behavior change.
- Upstream dependencies (`depends_on`): []
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`
- If `02_design.md` is skipped (Quick mode):
  - Why it is safe to skip: no architecture or integration changes.
  - What would trigger switching to Full mode: behavior changes or script/compose flow redesign.
- If `04_test_plan.md` is skipped:
  - Where validation is specified (must be in each task): task validation checklist below.

## Milestones

- M1: spec package ready and linted.
- M2: Makefile env centralization implemented and validated.

## Tasks (ordered)

1. T-001 - Centralize `service-up` env assignment lists
   - Scope: extract keyset-json, compose-up, and sync-catalog env lists to top-level variables.
   - Output: `service-up` recipe references reusable env blocks.
   - Linked requirements: FR-001 / FR-002 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `make -n service-up`
     - [x] Expected result: command expansion remains valid and includes required env variables.
     - [x] Logs/metrics to check (if applicable): N/A
2. T-002 - Verify no regression
   - Scope: run tests after Makefile refactor.
   - Output: validation evidence.
   - Linked requirements: NFR-001 / NFR-002
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./...`
     - [x] Expected result: tests pass.
     - [x] Logs/metrics to check (if applicable): N/A

## Traceability (optional)

- FR-001 -> T-001
- FR-002 -> T-001
- NFR-002 -> T-001, T-002
- NFR-006 -> T-001

## Validation evidence

- `make -n service-up` passed.
- `make -n local-receive-test` passed.
- `go test -short ./...` passed.
