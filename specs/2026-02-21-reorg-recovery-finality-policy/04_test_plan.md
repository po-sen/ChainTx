---
doc: 04_test_plan
spec_date: 2026-02-21
slug: reorg-recovery-finality-policy
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-detected-threshold-configurable
  - 2026-02-21-min-confirmations-configurable
  - 2026-02-20-reconciler-horizontal-scaling
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
  - lifecycle rollback/recovery (`confirmed -> reorged -> confirmed`)
  - settlement evidence persistence and canonical/orphan handling
  - split-installment visibility with one settlement evidence row per payment item
  - configuration validation for business/finality/observe-window/stability
  - webhook payload additions for transition reason context
- Not covered:
  - non-EVM new rails
  - downstream consumer business remediation logic

## Tests

### Unit

- TC-001: status parser and transition matrix includes `reorged`

  - Linked requirements: FR-001, NFR-006
  - Steps:
    - add parser/transition tests for valid `reorged` parsing and invalid status rejection.
  - Expected:
    - parser accepts `reorged`; existing statuses unchanged.

- TC-002: reconcile decision engine rollback/recovery logic

  - Linked requirements: FR-001, FR-005, NFR-002
  - Steps:
    - feed deterministic observation sequences for normal confirm, orphan rollback, and re-confirm.
  - Expected:
    - transitions occur only when policy criteria/stability counters are met.

- TC-003: config validation matrix
  - Linked requirements: FR-003, FR-004, FR-005, NFR-002
  - Steps:
    - test valid defaults and invalid env values (negative, finality < business, zero observe window).
  - Expected:
    - invalid input returns deterministic config errors and blocks startup.

### Integration

- TC-101: settlement evidence change-only write/orphan behavior

  - Linked requirements: FR-002, NFR-002, NFR-007
  - Steps:
    - run PostgreSQL integration tests that submit the same evidence payload
      across repeated cycles, then submit changed payload and orphan scenarios.
  - Expected:
    - no duplicates; unchanged payload does not update settlement rows;
      canonical aggregation uses only `is_canonical=true` rows.

- TC-102: webhook payload additive fields

  - Linked requirements: FR-006, NFR-005
  - Steps:
    - trigger status transitions through repository/use case and inspect outbox payload JSON.
  - Expected:
    - payload contains `transition_reason`, `finality_reached`, and `evidence_summary` with existing fields intact.

- TC-103: active-monitoring horizon query behavior

  - Linked requirements: FR-004, NFR-001
  - Steps:
    - seed confirmed rows with different first-confirmed timestamps/finality flags and claim for reconciliation.
  - Expected:
    - only rows within policy are actively claimed.

- TC-104: no-op reconciliation cycle write suppression

  - Linked requirements: FR-002, NFR-007
  - Steps:
    - run two consecutive reconcile cycles with identical observer evidence for the same request.
  - Expected:
    - `payment_request_settlements.updated_at` remains unchanged between cycles for unchanged evidence refs.

- TC-105: BTC split payments persist per-item evidence refs

  - Linked requirements: FR-002, NFR-008
  - Steps:
    - create one BTC request, send two confirmed payments to the same request address, and run reconciliation.
  - Expected:
    - settlement table contains at least two canonical rows for the request with distinct `txid:vout` evidence refs.

- TC-106: ETH split payments persist per-item tx evidence refs

  - Linked requirements: FR-002, NFR-008
  - Steps:
    - create one ETH request, send multiple successful native transfers to the
      request address, and run reconciliation.
  - Expected:
    - settlement table contains one canonical row per transfer with distinct transaction-hash evidence refs.

- TC-107: ERC20 split payments persist per-item log evidence refs
  - Linked requirements: FR-002, NFR-008
  - Steps:
    - create one ERC20 request, send multiple transfers (including multiple logs
      where applicable), and run reconciliation.
  - Expected:
    - settlement table contains one canonical row per transfer log with distinct `tx_hash:log_index` evidence refs.

### E2E (if applicable)

- TC-201: BTC forced reorg rollback/recovery

  - Linked requirements: FR-001, FR-002, FR-006, NFR-002
  - Steps:
    - create request, pay and confirm, invalidate tip blocks to orphan settlement, re-mine alternate chain including payment.
  - Expected:
    - status path shows `confirmed -> reorged -> confirmed`; webhook events include reorg/reconfirm reasons.

- TC-202: EVM forced reorg rollback/recovery

  - Linked requirements: FR-001, FR-002, FR-006, NFR-002
  - Steps:
    - create request, send transaction, force local chain reorg scenario, then re-include settlement transaction on canonical branch.
  - Expected:
    - lifecycle and webhook behavior match BTC expectations.

- TC-203: observe-window stop condition

  - Linked requirements: FR-004, FR-003, NFR-001
  - Steps:
    - configure short observe window; move request to finality-reached confirmed; run multiple reconciler cycles past window.
  - Expected:
    - request no longer actively polled after policy stop condition.

- TC-204: split-payment visibility survives reorg reconciliation
  - Linked requirements: FR-001, FR-002, NFR-002, NFR-008
  - Steps:
    - execute split payments, force local reorg that orphans at least one
      installment, then reconcile to canonical recovery.
  - Expected:
    - orphaned installments are marked non-canonical and canonical replacements remain individually queryable.

## Edge cases and failure modes

- Case: observer endpoint timeout during reorg window.
- Expected behavior: cycle error increments, no forced downgrade without canonical negative evidence.

- Case: duplicate evidence payload from observer in same cycle.
- Expected behavior: idempotent result; totals unchanged; no row update for unchanged evidence.

- Case: orphan signal arrives for unknown evidence reference.
- Expected behavior: row remains safe; log includes structured warning and no panic.

## NFR verification

- Performance:
  - verify p95 cycle latency target with synthetic 500-row claim workload in local profile.
- Reliability:
  - replay same observation snapshots across repeated cycles and assert deterministic end status.
- Security:
  - ensure logs and DB rows do not contain sensitive secret/private-key material.
- Storage efficiency:
  - verify unchanged evidence cycles do not mutate settlement rows (`updated_at` stable for unchanged refs).
