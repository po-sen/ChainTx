---
doc: 04_test_plan
spec_date: 2026-02-20
slug: chain-listener-payment-reconcile
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-keyset-startup-sync-audit-rotation
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Test Plan

## Scope

- Covered:
  - Reconciler config parsing/validation.
  - Reconcile use-case state transitions and idempotency.
  - BTC/EVM observer parsing and threshold decisions.
  - Runtime worker lifecycle behavior when enabled.
- Not covered:
  - Reorg rollback and deep-finality modeling.

## Tests

### Unit

- TC-001:

  - Linked requirements: FR-001, NFR-005
  - Steps: config tests for enabled/disabled, poll interval, batch size, invalid env values.
  - Expected: valid values parse; invalid values return config errors.

- TC-002:

  - Linked requirements: FR-003, FR-006, NFR-001
  - Steps: reconcile use case with expired pending request.
  - Expected: status transitions to `expired` via CAS.

- TC-003:

  - Linked requirements: FR-004, FR-006, NFR-001
  - Steps: use case with BTC observation `detected` then `confirmed` conditions.
  - Expected: `pending->detected` and `detected->confirmed` transitions occur.

- TC-004:

  - Linked requirements: FR-005, FR-006, NFR-001
  - Steps: use case with EVM observed amount meeting expected threshold.
  - Expected: open request transitions to `confirmed`.

- TC-005:
  - Linked requirements: FR-004, FR-005, NFR-005
  - Steps: observer adapter tests with mocked Esplora and JSON-RPC payloads.
  - Expected: parsed amounts and detected/confirmed booleans are correct.

### Integration

- TC-101:

  - Linked requirements: FR-002, FR-003, FR-006, NFR-001
  - Steps: repository integration for list-open + CAS status update.
  - Expected: only open statuses selected; CAS prevents stale double updates.

- TC-102:
  - Linked requirements: FR-007, NFR-004
  - Steps: run app with reconciler enabled against local stack and inspect logs.
  - Expected: periodic cycle logs emitted and worker exits on shutdown.

### E2E (if applicable)

- TC-201:

  - Linked requirements: FR-004, FR-005, FR-007
  - Steps: local receive flow creates requests, sends funds, waits at least one poll cycle.
  - Expected: relevant requests become `confirmed` via API `GET /v1/payment-requests/{id}`.

- TC-202:
  - Linked requirements: FR-003, FR-007
  - Steps: create short-expiry request and wait until expiration.
  - Expected: request transitions to `expired`.

## Edge cases and failure modes

- Case: observer endpoint missing for chain/network.
- Expected behavior: request is skipped, status unchanged, cycle logs include skip/error count.

- Case: request has null expected amount.
- Expected behavior: threshold defaults to positive observed amount.

- Case: stale concurrent status update.
- Expected behavior: CAS update returns no-op and cycle continues safely.

## NFR verification

- Performance: observe cycle latency logs with batch size 100 on local setup.
- Reliability: run multiple cycles and assert no invalid state regressions.
- Security: verify no secret/private key values are logged by observer/worker.

## Execution results (2026-02-20)

- `go test -short ./...` passed, including new unit tests for config/use case/observer/worker.
- `make local-up-all` with reconciler enabled and EVM RPC configured completed successfully.
- `make local-receive-test` passed and produced `deployments/local-chains/artifacts/service-receive-local-all.json`.
- Runtime verification:
  - ETH request status became `confirmed`.
  - USDT request status became `confirmed`.
  - BTC request stayed `pending` when no BTC observer endpoint was configured (cycle showed `skipped`).
