---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- `business confirmation`
- Minimum confirmation depth used to treat a payment as service-level `confirmed`.

- `finality confirmation`
- Higher confirmation depth indicating low reorg risk and allowing reduced monitoring.

- `reorged`
- A payment request status indicating previously canonical settlement evidence is no longer valid on the current canonical chain.

- `observe window`
- Duration after first `confirmed` in which reconciler must keep monitoring for reorg rollback even if finality is reached.

## Out-of-scope behaviors

- OOS1: per-merchant or per-payment-request custom confirmation/finality policy.
- OOS2: cross-chain bridge finality models or non-EVM additional rails.

## Functional requirements

### FR-001 - Extend lifecycle with reversible reorg status

- Description: payment-request lifecycle must explicitly support rollback and
  recovery under chain reorg.
- Acceptance criteria:
  - [ ] AC1: Domain status parser and DB status constraint support `reorged` in addition to existing statuses.
  - [ ] AC2: Reconciler transitions `confirmed -> reorged` when canonical
        evidence falls below confirmation threshold or tracked evidence is orphaned.
  - [ ] AC3: Reconciler transitions `reorged -> confirmed` when canonical
        evidence later satisfies confirmation requirements again.
  - [ ] AC4: Reconciler keeps existing transitions
        (`pending -> detected -> confirmed`, `pending/detected/reorged -> expired`)
        intact.
- Notes: `reorged` is non-terminal.

### FR-002 - Persist tx-level settlement evidence for decisioning

- Description: reconciliation must be driven by persisted settlement evidence,
  not only balance snapshots, and must preserve one row per observed payment
  installment.
- Acceptance criteria:
  - [ ] AC1: Settlement storage key remains
        `(payment_request_id, evidence_ref)` where `evidence_ref` is a chain-native
        payment item reference, not an aggregate snapshot label.
  - [ ] AC2: Adapter emits one evidence item per installment and persists each
        item separately (for example BTC `txid:vout`, ETH `tx_hash`, ERC20
        `tx_hash:log_index`).
  - [ ] AC3: Evidence row tracks at least amount, block reference, confirmation
        depth, canonical/orphan flag, and last persisted observation timestamp.
  - [ ] AC4: Reconcile cycle writes settlement rows only when evidence is new
        or changed; previously known canonical evidence missing from current
        observation is marked non-canonical.
  - [ ] AC5: Status decisions aggregate only canonical evidence amounts for detected/confirmed checks.
- Notes: this requirement intentionally disallows synthetic aggregate evidence
  refs such as balance snapshots as the primary settlement source.

### FR-003 - Configurable business/finality confirmation policy

- Description: operators must control confirmation depth for status and
  finality independently by chain family.
- Acceptance criteria:
  - [ ] AC1: Config supports `BTC` and `EVM` pairs: `business_min_confirmations` and `finality_min_confirmations`.
  - [ ] AC2: Startup validation fails when business confirmations < 1 or
        finality confirmations < business confirmations.
  - [ ] AC3: Defaults preserve existing semantics
        (`business = current min confirmations`, `finality = business`) unless
        explicitly overridden.
  - [ ] AC4: Reconciliation metadata/logs include business/finality values used in each decision.
- Notes: per-asset override is deferred.

### FR-004 - Explicit observation horizon after confirmation

- Description: system must define how long confirmed requests remain under reorg monitoring.
- Acceptance criteria:
  - [ ] AC1: Add `PAYMENT_REQUEST_RECONCILER_REORG_OBSERVE_WINDOW_SECONDS` with positive integer validation.
  - [ ] AC2: Confirmed/reorged requests are still claimable for reconciliation
        until `observe_window` expires from first-confirmed timestamp.
  - [ ] AC3: If finality is not reached, request remains claimable regardless of window elapsed.
  - [ ] AC4: Once finality reached and observe window elapsed, reconciler stops
        active chain polling for that request.
- Notes: lifecycle remains queryable after active polling stops.

### FR-005 - Anti-flap transition stabilization

- Description: prevent status thrashing from short-lived or noisy observations.
- Acceptance criteria:
  - [ ] AC1: Add configurable consecutive-cycle stability threshold for non-orphan promote/demote decisions.
  - [ ] AC2: `detected -> confirmed` and `confirmed -> reorged` (non-orphan path) require threshold cycles before transition.
  - [ ] AC3: Explicit orphan evidence may trigger immediate `confirmed -> reorged` without waiting cycles.
  - [ ] AC4: Decision details include applied stability counters.
- Notes: observer errors continue to count as cycle errors and must not directly downgrade status.

### FR-006 - Webhook contract includes reorg reason context

- Description: status-change webhook payload must expose reason context for
  rollback/recovery automation.
- Acceptance criteria:
  - [ ] AC1: `payment_request.status_changed` payload includes additive fields:
        `transition_reason`, `finality_reached`, and `evidence_summary`.
  - [ ] AC2: `transition_reason` supports at least `payment_confirmed`,
        `payment_reorged`, `payment_reconfirmed`, `payment_detected`,
        `payment_expired`.
  - [ ] AC3: Existing consumers that ignore new fields remain compatible.
- Notes: no new webhook event type is introduced.

## Non-functional requirements

- Performance (NFR-001): reconciler cycle processing for 500 claim rows must
  keep p95 cycle latency <= 2x current baseline in local profile.
- Availability/Reliability (NFR-002): reorg rollback/recovery transitions must
  be deterministic and idempotent under repeated polling and multi-worker
  leases.
- Security/Privacy (NFR-003): no private key/xpub/raw secret data is written to settlement evidence or logs.
- Compliance (NFR-004): not applicable in this iteration.
- Observability (NFR-005): every status transition log must include
  `request_id`, `status_from`, `status_to`, `transition_reason`, and
  confirmation metadata.
- Maintainability (NFR-006): new logic must preserve existing dependency
  direction (`domain <- application <- adapters <- infrastructure`) with no
  import-cycle regressions.
- Storage efficiency (NFR-007): when observer output is unchanged for a
  `(payment_request_id, evidence_ref)`, reconciler must not issue settlement row
  inserts/updates for that evidence in that cycle.
- Auditability (NFR-008): operators must be able to enumerate each installment
  payment from settlement rows alone without reconstructing from aggregate
  balances.

## Dependencies and integrations

- External systems: BTC Esplora API, EVM JSON-RPC endpoints used by current chain observer adapter.
- Internal services: existing reconciler worker, payment-request reconciliation repository, webhook outbox dispatch flow.
