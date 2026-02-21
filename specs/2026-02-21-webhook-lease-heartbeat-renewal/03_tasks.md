---
doc: 03_tasks
spec_date: 2026-02-21
slug: webhook-lease-heartbeat-renewal
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-status-outbox-delivery
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
- Rationale: async failure-flow hardening with concurrency and lease-safety behavior changes.
- Upstream dependencies (`depends_on`):
  - 2026-02-21-webhook-status-outbox-delivery
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`.

## Milestones

- M1: heartbeat renewal contract and repository adapter implemented.
- M2: use case orchestration updates and tests pass.
- M3: integration/smoke validation complete and docs/spec finalized.

## Tasks (ordered)

1. T-001 - Add outbox lease renewal port and PostgreSQL adapter implementation

   - Scope: extend `WebhookOutboxRepository` with `RenewLease` and implement guarded SQL update.
   - Output: renewal call can extend lease only for matching `id + lease_owner + pending` row.
   - Linked requirements: FR-001 / FR-002 / FR-004 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/adapters/outbound/persistence/postgresql/webhookoutbox/...`
     - [x] Expected result: package builds; SQL renewal guard is verified in code (`WHERE id=$1 AND delivery_status='pending' AND lease_owner=$2`).
     - [x] Logs/metrics to check (if applicable): N/A

2. T-002 - Implement per-event heartbeat lifecycle in dispatch use case

   - Scope: start/stop renewal loop around send call, derive heartbeat interval from lease duration, and keep mark semantics.
   - Output: active send attempts keep lease alive until completion.
   - Linked requirements: FR-001 / FR-003 / FR-004 / FR-005 / NFR-001 / NFR-002 / NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/application/use_cases/...`
     - [x] Expected result: long-running send path renews lease and delivered/retry/failed transitions still pass.
     - [x] Logs/metrics to check (if applicable): renewal failures surface through cycle error log (`code`, `message`, `details` include event/worker IDs).

3. T-003 - Add/adjust runtime-level tests for lease-expiry race prevention

   - Scope: add focused use-case tests for long send duration and lease-loss error path.
   - Output: test evidence that heartbeat renews during slow send and ownership-loss is surfaced.
   - Linked requirements: FR-001 / FR-004 / NFR-002
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/application/use_cases/... ./internal/infrastructure/webhook/...`
     - [x] Expected result: `DispatchWebhookEventsUseCase` tests pass for slow-send renewal, renewal error, and lease-loss scenarios.
     - [x] Logs/metrics to check (if applicable): cycle logs remain valid.

4. T-004 - End-to-end verification and spec closeout
   - Scope: run full Go verification and close spec with execution evidence.
   - Output: implementation validated; spec updated to DONE with evidence.
   - Linked requirements: FR-001 / FR-002 / FR-003 / FR-004 / FR-005 / NFR-001 / NFR-002 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: all checks pass.
     - [x] Logs/metrics to check (if applicable): N/A for this closeout pass.

## Traceability (optional)

- FR-001 -> T-001, T-002, T-003
- FR-002 -> T-001, T-004
- FR-003 -> T-002
- FR-004 -> T-001, T-002, T-003
- FR-005 -> T-002, T-004
- NFR-001 -> T-002, T-004
- NFR-002 -> T-002, T-003, T-004
- NFR-005 -> T-002, T-004
- NFR-006 -> T-001, T-004

## Rollout and rollback

- Feature flag: none (behavioral hardening under existing dispatcher flow).
- Migration sequencing: none.
- Rollback steps: revert use case/repository heartbeat changes; dispatcher returns to lease-only claim model.

## Validation evidence

- 2026-02-21 commands executed:
  - `go test ./internal/application/use_cases -run DispatchWebhookEvents -count=1` -> `ok`
  - `go test -short ./internal/adapters/outbound/persistence/postgresql/webhookoutbox/...` -> `[no test files]`
  - `go test -short ./internal/application/use_cases/... ./internal/infrastructure/webhook/...` -> use cases `ok`, webhook `[no test files]`
  - `go fmt ./...` -> pass
  - `go mod tidy` -> pass
  - `go list ./...` -> pass
  - `go test -short ./...` -> pass
  - `go vet ./...` -> pass
  - `SPEC_DIR=specs/2026-02-21-webhook-lease-heartbeat-renewal bash scripts/spec-lint.sh` -> pass
  - `bash scripts/verify-agents.sh` -> pass
