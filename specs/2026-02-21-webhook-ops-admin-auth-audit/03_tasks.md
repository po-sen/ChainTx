---
doc: 03_tasks
spec_date: 2026-02-21
slug: webhook-ops-admin-auth-audit
mode: Full
status: DONE
owners:
  - posen
depends_on:
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
- Rationale: security contract change + DB migration + operational API behavior updates.
- Upstream dependencies (`depends_on`):
  - 2026-02-21-webhook-observability-dlq-ops
- Dependency gate before `READY`: dependency is folder-wide `status: DONE`.

## Milestones

- M1: config and controller auth gate in place.
- M2: audit schema + mutation writes implemented.
- M3: docs and full runtime validation complete.

## Tasks (ordered)

1. T-001 - Add admin key runtime config and auth gate in webhook outbox controller

   - Scope: parse admin keys env, wire config into controller, enforce Bearer-only auth for all webhook outbox endpoints.
   - Output: unauthorized requests are rejected; endpoint fails closed when no keys configured.
   - Linked requirements: FR-001 / NFR-001 / NFR-003 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/infrastructure/config ./internal/adapters/inbound/http/controllers -count=1`
     - [x] Expected result: config parse + auth guard tests pass.
     - [x] Logs/metrics to check (if applicable): no key/token value appears in logs.

2. T-002 - Add operator-id requirements and audit schema/mutation writes

   - Scope: extend commands/repository signatures, add migration, persist action/actor/timestamp on requeue/cancel.
   - Output: successful manual mutations are auditable in DB.
   - Linked requirements: FR-002 / FR-003 / NFR-002 / NFR-004 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/application/use_cases -count=1 && go test -short ./...`
     - [x] Expected result: use-case validation and migration-backed compile/runtime tests pass.
     - [x] Logs/metrics to check (if applicable): N/A

3. T-003 - Update OpenAPI/README + run full verification and local receive/webhook proof
   - Scope: document Bearer-only auth/env, wire OpenAPI bearer security scheme for Swagger `Authorize`, and run full go checks with local end-to-end receive/webhook validation.
   - Output: docs aligned and runtime proof artifact/logs recorded.
   - Linked requirements: FR-004 / NFR-002 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: full suite passes and spec lint passes.
     - [x] Logs/metrics to check (if applicable): local receive + webhook delivery verified.

## Traceability (optional)

- FR-001 -> T-001
- FR-002 -> T-002
- FR-003 -> T-002
- FR-004 -> T-003
- NFR-001 -> T-001
- NFR-002 -> T-002, T-003
- NFR-003 -> T-001
- NFR-004 -> T-002
- NFR-005 -> T-001, T-003
- NFR-006 -> T-001, T-002, T-003

## Rollout and rollback

- Feature flag: none.
- Migration sequencing: apply migration before traffic serving mutation endpoints.
- Rollback steps: revert auth gate + audit column writes and roll back migration.

## Validation evidence

- 2026-02-21 commands executed:
  - `go test ./...` -> `ok`
  - `go fmt ./...` -> pass
  - `go mod tidy` -> pass
  - `go list ./...` -> pass
  - `go test -short ./...` -> pass
  - `go vet ./...` -> pass
  - `SPEC_DIR=specs/2026-02-21-webhook-ops-admin-auth-audit bash scripts/spec-lint.sh` -> pass
  - `bash scripts/verify-agents.sh specs/2026-02-21-webhook-ops-admin-auth-audit` -> pass
- Local runtime validation:
  - fresh local docker stack booted with `reconciler + webhook-dispatcher` enabled and admin keys configured.
  - `make local-receive-test` passed for BTC/ETH/USDT (`deployments/local-chains/artifacts/service-receive-local-all.json`).
  - payment request status polling confirmed all three requests reached `confirmed`.
  - webhook receiver container captured one signed webhook per BTC/ETH/USDT request with required headers (`X-ChainTx-Event-Id`, `Idempotency-Key`, `X-ChainTx-Delivery-Attempt`, `X-ChainTx-Signature-Version`, `X-ChainTx-Signature-V1`, `X-ChainTx-Signature`).
  - unauthorized webhook outbox ops request returned `401`.
  - `Authorization: Bearer ops-key-1` on webhook ops endpoint returned `200`.
  - legacy `X-ChainTx-Admin-Key` on webhook ops endpoint returned `401`.
  - missing `X-Principal-ID` on cancel returned `400 invalid_request`.
  - successful cancel/requeue API calls persisted audit metadata (`manual_last_action`, `manual_last_actor`, `manual_last_at`) in `app.webhook_outbox_events`.
  - Swagger `/swagger/openapi.yaml` includes `components.securitySchemes.WebhookOpsBearerAuth` and webhook ops operation-level `security`.
  - Swagger `/swagger/index.html` sets `persistAuthorization: true` for smoother operator workflow.
