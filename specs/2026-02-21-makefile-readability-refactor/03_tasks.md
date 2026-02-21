---
doc: 03_tasks
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

# Task Plan

## Mode decision

- Selected mode: Quick
- Rationale: Makefile readability-only refactor in existing targets; no schema/integration changes.
- Upstream dependencies (`depends_on`): []
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`
- If `02_design.md` is skipped (Quick mode):
  - Why it is safe to skip: no architecture or runtime behavior change.
  - What would trigger switching to Full mode: any deployment-flow redesign or behavior changes.
- If `04_test_plan.md` is skipped:
  - Where validation is specified (must be in each task): task-level validation steps below.

## Milestones

- M1: spec readiness and lint pass.
- M2: Makefile readability refactor complete and validation pass.

## Tasks (ordered)

1. T-001 - Refactor `service-up` long command lines
   - Scope: split long keyset generation, compose up env injection, and sync-catalog env chains into multiline blocks.
   - Output: `Makefile` `service-up` target readable sections with unchanged behavior.
   - Linked requirements: FR-001 / NFR-002 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `make -n service-up`
     - [x] Expected result: command expands successfully; env variables and script invocations remain present.
     - [x] Logs/metrics to check (if applicable): N/A
2. T-002 - Refactor `local-receive-test` long command line
   - Scope: rewrite the target to multiline `env` layout with same variables and script call.
   - Output: readable `local-receive-test` target.
   - Linked requirements: FR-002 / NFR-002 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `make -n local-receive-test`
     - [x] Expected result: command expands successfully with unchanged variable set.
     - [x] Logs/metrics to check (if applicable): N/A
3. T-003 - Run repository verification checks
   - Scope: ensure formatting and Go checks remain clean after Makefile edits.
   - Output: validation evidence in command outputs.
   - Linked requirements: NFR-001 / NFR-002
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./...`
     - [x] Expected result: all tests pass with no regressions.
     - [x] Logs/metrics to check (if applicable): N/A

## Traceability (optional)

- FR-001 -> T-001
- FR-002 -> T-002
- NFR-002 -> T-001, T-002, T-003
- NFR-006 -> T-001, T-002

## Validation evidence

- `make -n service-up` passed; command expansion includes all prior env keys and script calls in multiline form.
- `make -n local-receive-test` passed; command expansion remains valid.
- `go test -short ./...` passed.
