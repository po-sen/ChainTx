---
doc: 03_tasks
spec_date: 2026-02-21
slug: webhook-status-outbox-delivery
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
  - 2026-02-21-min-confirmations-configurable
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
- Rationale: includes API contract update, schema migration, and security validation flow change.
- Upstream dependencies (`depends_on`):
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
  - 2026-02-21-min-confirmations-configurable
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`.

## Milestones

- M1: payment request input contract supports required `webhook_url` with allowlist validation.
- M2: persistence + outbox destination snapshot wiring complete.
- M3: dispatcher/runtime/docs/tests updated and verified.

## Tasks (ordered)

1. T-001 - Add required `webhook_url` to payment request creation contract
   - Scope: update inbound request model/openapi/controller/use case DTO mapping to require `webhook_url`.
   - Output: create payment request path rejects missing `webhook_url` with explicit validation error.
   - Linked requirements: FR-006 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/adapters/inbound/http/controllers/... ./internal/application/use_cases/...`
     - [x] Expected result: missing `webhook_url` returns validation error; valid request passes.
     - [x] Logs/metrics to check (if applicable): validation error code appears in logs.
2. T-002 - Implement webhook URL allowlist config and validation
   - Scope: add `PAYMENT_REQUEST_WEBHOOK_URL_ALLOWLIST_JSON` parsing/validation and apply URL allowlist checks in creation flow.
   - Output: non-allowlisted destinations rejected before persistence.
   - Linked requirements: FR-007 / NFR-003 / NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/infrastructure/config/... ./internal/application/use_cases/...`
     - [x] Expected result: invalid allowlist config fails startup; out-of-allowlist `webhook_url` fails request validation.
     - [x] Logs/metrics to check (if applicable): explicit config/validation error codes.
3. T-003 - Persist request-level destination and snapshot into outbox
   - Scope: add schema and repository updates for payment request `webhook_url` and outbox `destination_url` snapshot.
   - Output: reconciler transition enqueue uses payment-bound destination URL in outbox record.
   - Linked requirements: FR-001 / FR-006 / FR-007 / NFR-002 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/adapters/outbound/persistence/postgresql/paymentrequest/...`
     - [x] Expected result: outbox row includes destination snapshot from payment request; no-op transition enqueues none.
     - [x] Logs/metrics to check (if applicable): N/A
4. T-004 - Dispatch to event-bound destination URL
   - Scope: adjust webhook dispatch path/ports/adapters so gateway receives destination per event rather than global URL.
   - Output: dispatcher sends webhook to outbox destination URL only.
   - Linked requirements: FR-002 / FR-004 / FR-005 / NFR-001 / NFR-002 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test -short ./internal/application/use_cases/... ./internal/adapters/outbound/webhook/http/...`
     - [x] Expected result: dispatch succeeds/fails based on per-event URL; retry/terminal behavior unchanged.
     - [x] Logs/metrics to check (if applicable): cycle log counters unchanged and still accurate.
5. T-005 - Runtime/config/docs alignment and full verification
   - Scope: update dispatcher startup checks (secret required), compose/Makefile/README/OpenAPI, and run full verification.
   - Output: operator docs and runtime behavior match new contract (required request `webhook_url`, allowlist-only destinations).
   - Linked requirements: FR-003 / FR-004 / FR-005 / FR-006 / FR-007 / NFR-003 / NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: all checks pass; docs/spec/code are consistent.
     - [x] Logs/metrics to check (if applicable): dispatcher startup errors are explicit for missing/invalid required settings.

## Traceability (optional)

- FR-001 -> T-003
- FR-002 -> T-004
- FR-003 -> T-005
- FR-004 -> T-004, T-005
- FR-005 -> T-004, T-005
- FR-006 -> T-001, T-003, T-005
- FR-007 -> T-002, T-003, T-005
- NFR-001 -> T-004
- NFR-002 -> T-003, T-004
- NFR-003 -> T-002, T-005
- NFR-005 -> T-001, T-002, T-005
- NFR-006 -> T-001, T-003, T-004

## Rollout and rollback

- Feature flag: `PAYMENT_REQUEST_WEBHOOK_ENABLED`.
- Migration sequencing: deploy schema changes first, then runtime code.
- Rollback steps: disable webhook dispatch runtime, keep additive schema, and revert API requirement only if explicitly needed.

## Validation evidence

- `go test ./...` passed.
- `go fmt ./...` completed with no pending formatting changes.
- `go mod tidy` completed with no dependency errors.
- `go list ./...` passed.
- `go test -short ./...` passed.
- `go vet ./...` passed.
- `SPEC_DIR=specs/2026-02-21-webhook-status-outbox-delivery bash scripts/spec-lint.sh` passed.
- `bash scripts/verify-agents.sh specs/2026-02-21-webhook-status-outbox-delivery` passed.
- `make local-up-all-no-explorer SERVICE_WEBHOOK_ENABLED=false SERVICE_RECONCILER_ENABLED=true SERVICE_EVM_RPC_URLS_JSON='{"local":"http://host.docker.internal:8545"}'` passed.
- `make local-receive-test` passed (`deployments/local-chains/artifacts/service-receive-local-all.json`).
- Manual API validation:
  - missing `webhook_url` returns `400 invalid_request`.
  - non-allowlisted host returns `400 webhook_url_not_allowed`.
  - allowlisted `localhost` request returns `201`.
  - same `Idempotency-Key` with different `webhook_url` returns `409 idempotency_key_conflict`.
- Manual DB validation:
  - `app.payment_requests.webhook_url` persists request value.
  - `app.webhook_outbox_events.destination_url` persists per-payment destination on new reconciled event rows.
- Manual webhook delivery validation:
  - started temporary listener inside `webhook-dispatcher` container on `localhost:18080`.
  - created ETH payment request with `webhook_url=http://localhost:18080/test-hook`, sent real local-chain payment, and observed outbox `delivery_status=delivered`.
  - captured raw webhook request including HMAC headers in `/tmp/webhook_requests.log` inside dispatcher container.
