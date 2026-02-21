---
doc: 03_tasks
spec_date: 2026-02-21
slug: webhook-alert-threshold-monitoring
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-status-outbox-delivery
  - 2026-02-21-webhook-observability-dlq-ops
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
- Rationale: introduces async threshold logic, then runtime split, deployment profile changes, and worker boundary refactor.
- Upstream dependencies (`depends_on`):
  - 2026-02-21-webhook-status-outbox-delivery
  - 2026-02-21-webhook-observability-dlq-ops
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`.

## Milestones

- M1: alert env/config contract implemented.
- M2: alert lifecycle state machine implemented.
- M3: runtime split (`webhook-alert-worker`) and dispatcher decoupling completed.
- M4: deployment/docs alignment and verification complete.

## Tasks (ordered)

1. T-001 - Implement alert env contract and validation

   - Scope: add alert fields/defaults/validation in config loader.
   - Output: deterministic config behavior and error codes.
   - Linked requirements: FR-001 / NFR-002 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/infrastructure/config -count=1`
     - [x] Expected result: parse/default/validation tests pass.
     - [x] Logs/metrics to check (if applicable): N/A

2. T-002 - Implement alert monitor lifecycle behavior

   - Scope: implement `triggered/ongoing/resolved` with cooldown.
   - Output: reusable signal state machine and tests.
   - Linked requirements: FR-003 / FR-004 / NFR-001 / NFR-002 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/infrastructure/webhookalert -count=1`
     - [x] Expected result: lifecycle and failure-path tests pass.
     - [x] Logs/metrics to check (if applicable): log lines contain signal/current/threshold/cooldown.

3. T-003 - Add dedicated alert runtime and DI wiring

   - Scope: add `cmd/webhook-alert-worker` and `BuildWebhookAlertWorker`.
   - Output: standalone alert worker runtime with startup validation.
   - Linked requirements: FR-002 / FR-003 / FR-004 / NFR-002 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./cmd/webhook-alert-worker ./internal/infrastructure/di -count=1`
     - [x] Expected result: runtime config validation and build path pass.
     - [x] Logs/metrics to check (if applicable): startup log shows alert worker lifecycle.

4. T-004 - Decouple dispatcher from alert evaluation

   - Scope: remove alert dependencies from dispatch worker/DI path.
   - Output: dispatcher runs delivery only.
   - Linked requirements: FR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/infrastructure/webhook ./cmd/webhook-dispatcher -count=1`
     - [x] Expected result: dispatch tests pass with no alert coupling.
     - [x] Logs/metrics to check (if applicable): dispatcher logs no alert evaluation entries.

5. T-005 - Deployment and docs cleanup

   - Scope: compose/make/readme align with split runtime and env scope.
   - Output: explicit `webhook-alert-worker` profile/service and docs examples.
   - Linked requirements: FR-006 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `make -n service-up SERVICE_WEBHOOK_ENABLED=true SERVICE_WEBHOOK_ALERT_ENABLED=true`
     - [x] Expected result: includes `webhook-alert-worker` profile and scaling args.
     - [x] Logs/metrics to check (if applicable): N/A

6. T-006 - Full verification and spec closeout
   - Scope: run full Go/spec/policy checks.
   - Output: release-ready state.
   - Linked requirements: FR-001 / FR-002 / FR-003 / FR-004 / FR-005 / FR-006 / NFR-001 / NFR-002 / NFR-003 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: all checks pass.
     - [x] Logs/metrics to check (if applicable): N/A

## Traceability (optional)

- FR-001 -> T-001, T-006
- FR-002 -> T-003, T-006
- FR-003 -> T-002, T-003, T-006
- FR-004 -> T-002, T-003, T-006
- FR-005 -> T-004, T-006
- FR-006 -> T-005, T-006
- NFR-001 -> T-002, T-006
- NFR-002 -> T-001, T-002, T-003, T-006
- NFR-003 -> T-006
- NFR-005 -> T-002, T-006
- NFR-006 -> T-001, T-002, T-003, T-004, T-005, T-006

## Rollout and rollback

- Feature flag: `PAYMENT_REQUEST_WEBHOOK_ALERT_ENABLED`.
- Migration sequencing: none.
- Rollback steps: disable alert worker profile or revert runtime split commits.

## Validation evidence

- 2026-02-21 commands executed:
  - `go test ./internal/infrastructure/config -count=1` -> `ok`
  - `go test ./internal/infrastructure/webhookalert -count=1` -> `ok`
  - `go test ./internal/infrastructure/webhook ./cmd/webhook-dispatcher -count=1` -> `ok`
  - `go test ./cmd/webhook-alert-worker ./internal/infrastructure/di -count=1` -> `ok` (`di`: no test files)
  - `make -n service-up SERVICE_WEBHOOK_ENABLED=true SERVICE_WEBHOOK_ALERT_ENABLED=true` -> includes `webhook-alert-worker` profile/service
  - `go fmt ./...` -> pass
  - `go mod tidy` -> pass
  - `go list ./...` -> pass
  - `go test -short ./...` -> pass
  - `go vet ./...` -> pass
  - `COMPOSE_PROFILES=webhook-dispatcher,webhook-alert-worker PAYMENT_REQUEST_WEBHOOK_ENABLED=true PAYMENT_REQUEST_WEBHOOK_HMAC_SECRET=local-webhook-secret PAYMENT_REQUEST_WEBHOOK_ALERT_ENABLED=true PAYMENT_REQUEST_WEBHOOK_ALERT_FAILED_COUNT_THRESHOLD=1 docker compose -f deployments/service/docker-compose.yml --project-name chaintx-local-service up -d --build postgres app webhook-dispatcher webhook-alert-worker` -> pass
  - `docker compose -f deployments/service/docker-compose.yml --project-name chaintx-local-service logs --tail=80 webhook-alert-worker webhook-dispatcher` -> alert lifecycle log appears in `webhook-alert-worker`, dispatcher remains delivery-only
  - `COMPOSE_PROFILES=reconciler,webhook-dispatcher,webhook-alert-worker docker compose -f deployments/service/docker-compose.yml --project-name chaintx-local-service stop` -> pass
  - `SPEC_DIR=specs/2026-02-21-webhook-alert-threshold-monitoring bash scripts/spec-lint.sh` -> pass
  - `bash scripts/verify-agents.sh specs/2026-02-21-webhook-alert-threshold-monitoring` -> pass
