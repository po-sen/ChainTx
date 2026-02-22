---
doc: 03_tasks
spec_date: 2026-02-22
slug: payment-request-settlements-api
mode: Quick
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-reorg-recovery-finality-policy
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
- Rationale: additive read API only; no schema migration, no new external integration, and low risk change.
- Upstream dependencies (`depends_on`):
  - 2026-02-21-reorg-recovery-finality-policy
- Dependency gate before `READY`: dependency is folder-wide `status: DONE`.
- If `02_design.md` is skipped (Quick mode):
  - Why it is safe to skip: existing payment request read-path architecture already established and reused.
  - What would trigger switching to Full mode: adding schema changes, pagination contracts, or cross-service integration.
- If `04_test_plan.md` is skipped:
  - Where validation is specified (must be in each task): each task below includes command-level validation steps.

## Milestones

- M1: spec ready and architecture mapping finalized.
- M2: endpoint implemented across controller/use case/read model + OpenAPI.
- M3: tests and verification pass; spec status moved to DONE.

## Tasks (ordered)

1. T-001 - Add application query contract for settlement listing

   - Scope: define DTOs, inbound port, and use case for `payment-request-settlements` query including validation and not-found handling.
   - Output: application layer exposes settlement list use case aligned with existing error model.
   - Linked requirements: FR-001 / FR-003 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/application/use_cases -count=1`
     - [x] Expected result: new use case validation/not-found/success behavior passes.
     - [x] Logs/metrics to check (if applicable): N/A

2. T-002 - Implement outbound read-model and inbound HTTP wiring

   - Scope: add read model query in PostgreSQL adapter, wire new route/controller method, and update DI + router.
   - Output: `GET /v1/payment-requests/{id}/settlements` serves deterministic settlement list.
   - Linked requirements: FR-001 / FR-002 / FR-003 / NFR-001 / NFR-002 / NFR-003 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/adapters/inbound/http/... ./internal/adapters/outbound/persistence/postgresql/paymentrequest -count=1`
     - [x] Expected result: controller/router and read-model integration tests pass.
     - [x] Logs/metrics to check (if applicable): N/A

3. T-003 - Update API contract and run repo verification gates
   - Scope: update OpenAPI schema/path and execute required project checks and policy verification.
   - Output: Swagger renders correctly and repository verification passes without regressions.
   - Linked requirements: FR-001 / FR-002 / NFR-002 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: all commands pass.
     - [x] Logs/metrics to check (if applicable): `GET /swagger/openapi.yaml` includes new settlements endpoint.

## Traceability (optional)

- FR-001 -> T-001, T-002, T-003
- FR-002 -> T-002, T-003
- FR-003 -> T-001, T-002
- NFR-001 -> T-002
- NFR-002 -> T-002, T-003
- NFR-003 -> T-002
- NFR-005 -> T-001, T-003
- NFR-006 -> T-001, T-002, T-003

## Rollout and rollback

- Feature flag: none.
- Migration sequencing: none (read-only feature on existing schema).
- Rollback steps: revert route/controller/use case/read model additions and OpenAPI path.

## Validation evidence

- 2026-02-22 commands executed:
  - `go test ./internal/application/use_cases ./internal/adapters/inbound/http/controllers -count=1` -> `ok`
  - `go test -tags=integration ./internal/adapters/inbound/http/router -count=1` -> `ok`
  - `go test -tags=integration ./internal/adapters/outbound/persistence/postgresql/paymentrequest -run 'TestPaymentRequestReadModelListSettlementsByPaymentRequestID' -count=1` -> `ok`
  - `go fmt ./...` -> pass
  - `go mod tidy` -> pass
  - `go list ./...` -> pass
  - `go test ./...` -> pass
  - `go test -short ./...` -> pass
  - `go vet ./...` -> pass
  - `SPEC_DIR=specs/2026-02-22-payment-request-settlements-api bash scripts/spec-lint.sh` -> pass
  - `bash scripts/verify-agents.sh specs/2026-02-22-payment-request-settlements-api` -> pass
