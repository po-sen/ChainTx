---
doc: 03_tasks
spec_date: null # set to YYYY-MM-DD
slug: null # set to kebab-case slug
mode: Quick
status: DRAFT
owners: [] # add at least one before READY/DONE
depends_on: [] # list prerequisite spec folder names; use block list when non-empty
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: null # set to 02_design.md in Full mode
  tasks: 03_tasks.md
  test_plan: null # set to 04_test_plan.md if produced
---

# Task Plan

## Mode decision

- Selected mode: Quick / Full
- Rationale:
- Upstream dependencies (`depends_on`): [] / list of `YYYY-MM-DD-slug`
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`
- If `02_design.md` is skipped (Quick mode):
  - Why it is safe to skip:
  - What would trigger switching to Full mode:
- If `04_test_plan.md` is skipped:
  - Where validation is specified (must be in each task):

## Milestones

- M1:
- M2:

## Tasks (ordered)

1. T-001 - title
   - Scope:
   - Output:
   - Linked requirements: FR-??? / NFR-???
   - Validation:
     - [ ] How to verify (manual steps or command):
     - [ ] Expected result:
     - [ ] Logs/metrics to check (if applicable):
2. T-002 - title
   - Scope:
   - Output:
   - Linked requirements: FR-??? / NFR-???
   - Validation:
     - [ ] How to verify (manual steps or command):
     - [ ] Expected result:
     - [ ] Logs/metrics to check (if applicable):

## Traceability (optional)

- FR-001 -> T-001, T-002
- NFR-001 -> T-002

## Rollout and rollback

- Feature flag:
- Migration sequencing:
- Rollback steps:
