---
doc: 04_test_plan
spec_date: 2026-02-20
slug: reconciler-horizontal-scaling
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
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
  - claim+lease repository behavior.
  - reconcile use case claim-first orchestration.
  - standalone reconciler runtime startup/shutdown.
  - compose/make multi-replica run path.
- Not covered:
  - deep blockchain finality semantics.
  - queue-based partitioning.

## Tests

### Unit

- TC-001:
  - Linked requirements: FR-002, FR-003, NFR-001, NFR-006
  - Steps: run use case tests for claim-driven branches and lease-aware transitions.
  - Expected: status changes happen only for claimed rows; stale updates are skipped.
- TC-002:
  - Linked requirements: FR-004, NFR-006
  - Steps: run config tests for lease seconds parsing and invalid values.
  - Expected: positive values parse; invalid values fail with explicit config code.

### Integration

- TC-101:
  - Linked requirements: FR-001, FR-005, NFR-005
  - Steps: start compose service with reconciler enabled and replicas > 1.
  - Expected: dedicated reconciler containers run and API app stays healthy.
- TC-102:
  - Linked requirements: FR-002, FR-003, NFR-001, NFR-003
  - Steps: create multiple open requests and observe worker logs for claimed counts across replicas.
  - Expected: same request is not claimed by two workers during active lease; rows recover after lease timeout if unfinished.

### E2E (if applicable)

- Scenario 1: full local receive flow with reconciler replicas confirms ETH/USDT requests.
- Scenario 2: kill one reconciler replica mid-cycle and verify remaining replica continues claims after lease timeout.

## Edge cases and failure modes

- Case: reconciler starts with invalid lease config.
- Expected behavior: startup fails fast with config error.
- Case: observer endpoint failure after successful claim.
- Expected behavior: row remains open until lease expiry and can be retried later.

## NFR verification

- Performance: observe cycle latency logs with batch size 100 and 2 replicas in local profile.
- Reliability: verify claimed rows are retried after lease timeout when processing is interrupted.
- Security: verify logs include worker/claim metadata but no secret key material.

## Execution results (2026-02-20)

- `go test -short ./...` passed with new claim/lease and worker-runtime tests.
- `make local-up-all` with `SERVICE_RECONCILER_ENABLED=true` and `SERVICE_RECONCILER_REPLICAS=2` succeeded.
- compose runtime confirms two reconciler containers are running.
- reconciler logs include distinct worker IDs and claim counters.
- `make local-receive-test` passed:
  - ETH request status `confirmed`
  - USDT request status `confirmed`
  - BTC remains `pending` when BTC observer endpoint is not configured (expected skip behavior).
