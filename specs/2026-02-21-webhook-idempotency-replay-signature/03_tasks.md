---
doc: 03_tasks
spec_date: 2026-02-21
slug: webhook-idempotency-replay-signature
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-21-webhook-status-outbox-delivery
  - 2026-02-21-webhook-lease-heartbeat-renewal
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
- Rationale: webhook contract and security-hardening changes affect async delivery behavior and receiver integration guidance.
- Upstream dependencies (`depends_on`):
  - 2026-02-21-webhook-status-outbox-delivery
  - 2026-02-21-webhook-lease-heartbeat-renewal
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`.

## Milestones

- M1: application DTO/use case can pass explicit delivery attempt metadata.
- M2: webhook HTTP gateway emits idempotency + anti-replay v1 headers (plus legacy signature compatibility).
- M3: docs/tests updated and verification complete.

## Tasks (ordered)

1. T-001 - Extend dispatch DTO/use case for delivery attempt header support

   - Scope: add delivery attempt metadata to gateway input and populate from outbox attempts.
   - Output: each outbound attempt carries deterministic 1-based `DeliveryAttempt`.
   - Linked requirements: FR-001 / FR-003 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/application/use_cases -run DispatchWebhookEvents -count=1`
     - [x] Expected result: dispatch tests pass and attempt count mapping remains correct (`DeliveryAttempt=attempts+1`).
     - [x] Logs/metrics to check (if applicable): N/A

2. T-002 - Implement versioned anti-replay webhook signature headers in HTTP gateway

   - Scope: generate nonce, build canonical v1 payload, emit v1 signature/version headers, keep legacy signature header.
   - Output: outbound request contains idempotency + replay-defense header set.
   - Linked requirements: FR-001 / FR-002 / FR-003 / NFR-001 / NFR-002 / NFR-003 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go test ./internal/adapters/outbound/webhook/http -count=1`
     - [x] Expected result: tests assert idempotency/attempt/replay headers and signature derivation for legacy + v1.
     - [x] Logs/metrics to check (if applicable): N/A

3. T-003 - Document receiver contract for idempotency and replay checks

   - Scope: update README webhook section with mandatory receiver verification steps and header mapping.
   - Output: concrete integration guidance for downstream systems.
   - Linked requirements: FR-004 / NFR-005
   - Validation:
     - [x] How to verify (manual steps or command): `rg -n "X-ChainTx-Signature-V1|Idempotency-Key|X-ChainTx-Nonce|X-ChainTx-Delivery-Attempt" README.md`
     - [x] Expected result: README includes canonical payload description and receiver-side idempotency/replay checklist.
     - [x] Logs/metrics to check (if applicable): N/A

4. T-004 - Run repository verification and close spec
   - Scope: run formatting/lint/tests/spec gates and update spec statuses.
   - Output: implementation and documentation validated.
   - Linked requirements: FR-001 / FR-002 / FR-003 / FR-004 / NFR-001 / NFR-002 / NFR-003 / NFR-005 / NFR-006
   - Validation:
     - [x] How to verify (manual steps or command): `go fmt ./... && go mod tidy && go list ./... && go test -short ./... && go vet ./...`
     - [x] Expected result: all commands pass; spec lint passes for this folder.
     - [x] Logs/metrics to check (if applicable): N/A

## Traceability (optional)

- FR-001 -> T-001, T-002, T-004
- FR-002 -> T-002, T-004
- FR-003 -> T-001, T-002, T-004
- FR-004 -> T-003, T-004
- NFR-001 -> T-002, T-004
- NFR-002 -> T-002, T-004
- NFR-003 -> T-002, T-004
- NFR-005 -> T-001, T-003, T-004
- NFR-006 -> T-001, T-002, T-004

## Rollout and rollback

- Feature flag: none.
- Migration sequencing: none.
- Rollback steps: revert webhook gateway header/signature additions and DTO field if receiver rollout needs rollback.

## Validation evidence

- 2026-02-21 commands executed:
  - `go test ./internal/application/use_cases -run DispatchWebhookEvents -count=1` -> `ok`
  - `go test ./internal/adapters/outbound/webhook/http -count=1` -> `ok`
  - `go fmt ./...` -> pass
  - `go mod tidy` -> pass
  - `go list ./...` -> pass
  - `go test -short ./...` -> pass
  - `go vet ./...` -> pass
  - `SPEC_DIR=specs/2026-02-21-webhook-idempotency-replay-signature bash scripts/spec-lint.sh` -> pass
  - `bash scripts/verify-agents.sh specs/2026-02-21-webhook-idempotency-replay-signature` -> pass
