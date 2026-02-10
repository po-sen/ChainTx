---
doc: 03_tasks
spec_date: 2026-02-09
slug: multi-asset-payment-request-api
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-07-postgresql-backend-service-compose
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
- Rationale: This scope introduces new persistent data models, transactional concurrency control, idempotency storage, and wallet allocation integration, which exceeds Quick-mode complexity.
- Upstream dependencies (`depends_on`):
  - 2026-02-07-postgresql-backend-service-compose
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`.
- If `02_design.md` is skipped (Quick mode): Not applicable; Full mode is mandatory.
- If `04_test_plan.md` is skipped: Not applicable; Full mode requires test plan.

## Milestones

- M1: Contract + domain model finalized for payment request resource and multi-asset instructions.
- M2: Persistence and allocation path implemented with transactional uniqueness and idempotency.
- M3: API endpoints, OpenAPI docs, and full verification suite completed.

## Tasks (ordered)

1. T-001 - Finalize asset catalog governance and seed data

   - Scope: Define source-of-truth process and seed records for BTC/ETH/USDT on supported networks, including explicit wallet-account mapping (ETH and USDT sharing one Ethereum allocator where intended).
   - Output: Asset catalog schema + seed/migration artifacts + operational note for contract verification and mapping integrity checks.
   - Linked requirements: FR-001, FR-003, NFR-004
   - Validation:
     - [ ] How to verify (manual steps or command): Query seeded catalog rows in local DB after migration.
     - [ ] Expected result: Exactly expected `(chain, network, asset)` entries are enabled with complete metadata (`decimals`, token fields) and correct `wallet_account_id` mapping.
     - [ ] Expected result: Startup validation fails fast when an enabled asset row references a missing/inactive wallet account, has incompatible allocator config, or has out-of-range `default_expires_in_seconds`.
     - [ ] Logs/metrics to check (if applicable): Migration logs show successful seed apply and idempotent rerun behavior.

2. T-002 - Implement domain and application contracts

   - Scope: Add value objects/enums, DTOs, inbound/outbound ports, and use cases for list/create/get flows.
   - Output: Compilable application layer with no adapter or driver leakage.
   - Linked requirements: FR-002, FR-003, FR-004, FR-008, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): `go test ./internal/application/... ./internal/domain/...`
     - [ ] Expected result: Unit tests pass for command validation, strict `expected_amount_minor` pattern/length rules, default expiry resolution, and non-null `expires_at` computation.
     - [ ] Logs/metrics to check (if applicable): N/A for pure unit scope.

3. T-003 - Add persistence schema and repositories

   - Scope: Create migrations and PostgreSQL repositories for `payment_requests`, `wallet_accounts`, `asset_catalog`, and idempotency records with canonical address and scope-aware idempotency keys.
   - Output: Transaction-capable repository layer with required unique/check constraints.
   - Linked requirements: FR-006, FR-005, FR-008, NFR-002, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): Run migrations twice, then run repository integration tests.
     - [ ] Expected result: Schema is present, reruns are idempotent, and constraint violations surface as typed errors (`wallet_account_id + derivation_index`, `chain + network + address_canonical`, scope + idempotency key, idempotency TTL minimum 24h, expiry bounds 60s..30d).
     - [ ] Expected result: Shared EVM allocator rows enforce compatible `address_scheme` and `chain_id`, and enabled catalog defaults enforce `default_expires_in_seconds` bounds.
     - [ ] Logs/metrics to check (if applicable): Migration/repository logs include stable error codes and operation phase.

4. T-004 - Implement wallet allocation adapter and transactional create flow

   - Scope: Implement allocation gateway and create-use-case orchestration with row lock + index increment + request persistence + rollback semantics, including cross-asset shared allocator behavior for EVM.
   - Output: `CreatePaymentRequestUseCase` with atomic behavior and bounded retry on transient conflicts.
   - Linked requirements: FR-002, FR-005, FR-006, NFR-002, NFR-005
   - Validation:
     - [ ] How to verify (manual steps or command): Concurrency integration test with >=200 parallel creates per asset tuple plus mixed ETH/USDT load.
     - [ ] Expected result: Zero duplicate `(wallet_account_id, derivation_index)` and zero duplicate `(chain, network, address_canonical)`.
     - [ ] Logs/metrics to check (if applicable): Allocation latency histogram and retry counters are emitted.

5. T-005 - Add HTTP endpoints and error mapping

   - Scope: Add controllers/router wiring for `GET /v1/assets`, `POST /v1/payment-requests`, and `GET /v1/payment-requests/{id}` using unified error schema and canonical address response formatting.
   - Output: Working REST endpoints with stable response and error contracts.
   - Linked requirements: FR-001, FR-002, FR-004, FR-005, FR-007, FR-008, NFR-003
   - Validation:
     - [ ] How to verify (manual steps or command): HTTP integration tests over in-memory/test DB-backed server.
     - [ ] Expected result: Correct HTTP codes (`200/201/400/404/409`), extensible status field behavior, and contract-compliant response bodies.
     - [ ] Logs/metrics to check (if applicable): Logs include request id, error code, and replay indicator where applicable.

6. T-006 - Implement idempotency persistence and replay behavior

   - Scope: Persist canonical request hash + replay payload, enforce conflict behavior, and wire request-body normalization using RFC 8785 JCS + SHA-256.
   - Output: Deterministic idempotency module integrated into create path.
   - Linked requirements: FR-005, FR-007, NFR-002, NFR-005
   - Validation:
     - [ ] How to verify (manual steps or command): Integration tests for same-key same-body replay and same-key different-body conflict.
     - [ ] Expected result: Replay returns same resource payload with `200` and `X-Idempotency-Replayed: true`; conflict returns `409 idempotency_key_conflict` without new allocation.
     - [ ] Logs/metrics to check (if applicable): `payment_request_idempotency_replay_total` increments only on true replay.

7. T-007 - Update OpenAPI and Swagger examples

   - Scope: Document all new endpoints, schemas, polymorphic instructions, and error cases in `api/openapi.yaml`.
   - Output: OpenAPI contract synchronized with implementation and consumable via existing Swagger route.
   - Linked requirements: FR-009, FR-003, FR-007
   - Validation:
     - [ ] How to verify (manual steps or command): OpenAPI schema validation plus route-level docs integration test.
     - [ ] Expected result: Spec is valid, examples cover BTC/ETH/USDT, and Swagger renders without errors.
     - [ ] Logs/metrics to check (if applicable): N/A.

8. T-008 - Execute full quality gates and publish readiness evidence

   - Scope: Run unit/integration/e2e tests and static checks; complete traceability evidence.
   - Output: Test evidence bundle and status transition decision (`DRAFT` -> `READY` when all gates pass).
   - Linked requirements: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, NFR-001, NFR-002, NFR-003, NFR-004, NFR-005, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): `go fmt ./... && go vet ./... && go list ./... && go test ./...` plus full test plan execution.
     - [ ] Expected result: All checks pass and traceability is complete.
     - [ ] Logs/metrics to check (if applicable): No unclassified internal errors in smoke runs.

## Traceability (optional)

- FR-001 -> T-001, T-005, T-008
- FR-002 -> T-002, T-004, T-005, T-008
- FR-003 -> T-001, T-002, T-007, T-008
- FR-004 -> T-002, T-005, T-008
- FR-005 -> T-003, T-004, T-005, T-006, T-008
- FR-006 -> T-003, T-004, T-008
- FR-007 -> T-005, T-006, T-007, T-008
- FR-008 -> T-002, T-003, T-005, T-008
- FR-009 -> T-007, T-008
- NFR-001 -> T-004, T-005, T-008
- NFR-002 -> T-003, T-004, T-006, T-008
- NFR-003 -> T-005, T-008
- NFR-004 -> T-001, T-008
- NFR-005 -> T-004, T-005, T-006, T-008
- NFR-006 -> T-002, T-003, T-008

## Rollout and rollback

- Feature flag: `PAYMENT_REQUESTS_V1_ENABLED` (default off until verification complete).
- Migration sequencing:
  - Apply schema migrations and asset seed.
  - Enable flag in non-production environment.
  - Run smoke tests for each asset.
- Rollback steps:
  - Disable `PAYMENT_REQUESTS_V1_ENABLED`.
  - Keep created rows for auditability; do not reuse consumed indices.
  - If required, roll back application deploy without dropping migration history.

## Readiness checklist (for status READY)

- [ ] Open questions in `00_problem.md` are resolved or explicitly accepted as tracked assumptions.
- [ ] Asset catalog source-of-truth and approval process is documented and approved.
- [ ] Full test plan execution evidence is attached.
- [ ] `SPEC_DIR="specs/2026-02-09-multi-asset-payment-request-api" bash /Users/posen/.codex/skills/spec-driven-development/scripts/spec-lint.sh` passes.
- [ ] All produced docs remain frontmatter-consistent with `mode: Full` and same `depends_on` set.
