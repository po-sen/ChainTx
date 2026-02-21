---
doc: 03_tasks
spec_date: 2026-02-21
slug: webhook-retry-jitter-budget
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-lease-heartbeat-renewal
  - 2026-02-21-webhook-idempotency-replay-signature
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Task Plan

## Mode decision

- Selected mode: Full
- Rationale: asynchronous failure-handling behavior changes and runtime configuration contract extension.
- Upstream dependencies (`depends_on`):
  - 2026-02-21-webhook-lease-heartbeat-renewal
  - 2026-02-21-webhook-idempotency-replay-signature
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`.

## Milestones

- M1: config/runtime command supports retry jitter + budget.
- M2: use-case retry decision/backoff includes jitter and budget logic.
- M3: tests/docs/verification complete and spec closed.

## Tasks (ordered)

1. T-001 - Add webhook runtime config fields for retry jitter and retry budget

   - Scope: extend config constants, parse validation, config struct, and wiring into worker command.
   - Output: dispatcher runtime receives `RetryJitterBPS` and `RetryBudget` with safe defaults.
   - Linked requirements: FR-001 / FR-002 / FR-004 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/infrastructure/config -count=1`
     - [x] Expected result: new envs parse/validate correctly; invalid values fail fast.
     - [x] Logs/metrics to check (if applicable): N/A

2. T-002 - Implement jittered backoff and effective retry budget logic in dispatch use case

   - Scope: compute effective max attempts and jittered backoff (deterministic hash-based) for retry path.
   - Output: retries are jittered and capped by runtime budget when configured.
   - Linked requirements: FR-001 / FR-002 / FR-003 / NFR-001 / NFR-002 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/application/use_cases -run DispatchWebhookEvents -count=1`
     - [x] Expected result: tests pass for jitter disabled/enabled and budget terminal behavior.
     - [x] Logs/metrics to check (if applicable): cycle counts reflect retry/failed transitions.

3. T-003 - Update README webhook config documentation

   - Scope: document new envs and behavior semantics (disabled defaults, ranges, effective attempt cap).
   - Output: operator-facing configuration contract is explicit.
   - Linked requirements: FR-004 / NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `rg -n "WEBHOOK_RETRY_JITTER_BPS|WEBHOOK_RETRY_BUDGET" README.md`
     - [x] Expected result: README contains both env rows and concise behavior notes.
     - [x] Logs/metrics to check (if applicable): N/A

4. T-004 - Full verification and spec closeout
   - Scope: run repository verification workflow and update spec status/evidence.
   - Output: production-ready change with test evidence.
   - Linked requirements: FR-001 / FR-002 / FR-003 / FR-004 / NFR-001 / NFR-002 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: all commands and spec lint pass.
     - [x] Logs/metrics to check (if applicable): N/A

## Traceability (optional)

- FR-001 -> T-001, T-002, T-004
- FR-002 -> T-001, T-002, T-004
- FR-003 -> T-002, T-004
- FR-004 -> T-001, T-003, T-004
- NFR-001 -> T-002, T-004
- NFR-002 -> T-002, T-004
- NFR-005 -> T-002, T-003
- NFR-006 -> T-001, T-002, T-004

## Rollout and rollback

- Feature flag: none.
- Migration sequencing: none.
- Rollback steps: revert config/worker/use-case jitter-budget changes; fallback to prior deterministic backoff + row max attempts.

## Validation evidence

- 2026-02-21 commands executed:
  - `go test ./internal/infrastructure/config -count=1` -> `ok`
  - `go test ./internal/application/use_cases -run DispatchWebhookEvents -count=1` -> `ok`
  - `go test ./internal/infrastructure/webhook -count=1` -> `ok`
  - `go test ./cmd/webhook-dispatcher -count=1` -> `ok`
  - `go fmt ./...` -> pass
  - `go mod tidy` -> pass
  - `go list ./...` -> pass
  - `go test -short ./...` -> pass
  - `go vet ./...` -> pass
  - `SPEC_DIR=specs/2026-02-21-webhook-retry-jitter-budget bash scripts/spec-lint.sh` -> pass
  - `bash scripts/verify-agents.sh specs/2026-02-21-webhook-retry-jitter-budget` -> pass
