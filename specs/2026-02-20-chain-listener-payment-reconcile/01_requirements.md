---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- Reconciler: background polling worker that inspects open payment requests and updates status.
- Open request: payment request in status `pending` or `detected`.
- Observation: chain-side measured amount for a request address.

## Out-of-scope behaviors

- OOS1: Websocket/subscription listeners.
- OOS2: Deep finality/reorg compensation workflows.

## Functional requirements

### FR-001 - Reconciler runtime configuration

- Description: Service must support config-driven enable/disable and polling parameters for reconciler loop.
- Acceptance criteria:
  - [x] AC1 `PAYMENT_REQUEST_RECONCILER_ENABLED` controls whether worker starts.
  - [x] AC2 Poll interval and batch size are configurable with sane defaults and validation.
  - [x] AC3 Invalid reconciler config fails startup with explicit config error code.
- Notes: Defaults must not break existing users when feature is disabled.

### FR-002 - Open request selection for reconciliation

- Description: Repository layer must list open requests for polling.
- Acceptance criteria:
  - [x] AC1 Selection includes only statuses `pending` and `detected`.
  - [x] AC2 Selection is ordered deterministically and limited by batch size.
  - [x] AC3 Query shape supports efficient polling via indexes.
- Notes: Exclude already terminal statuses (`confirmed`, `expired`, `failed`).

### FR-003 - Expiry transition

- Description: Reconciler must expire unpaid requests after `expires_at`.
- Acceptance criteria:
  - [x] AC1 If open request has `expires_at <= now`, status transitions to `expired`.
  - [x] AC2 Transition is compare-and-set on previous status to preserve idempotency.
  - [x] AC3 Repeated polls do not re-transition already expired rows.
- Notes: Applies regardless of chain observer availability.

### FR-004 - BTC observation via Esplora

- Description: Reconciler must observe BTC request addresses using configured Esplora endpoint.
- Acceptance criteria:
  - [x] AC1 For BTC requests, observer reads address chain/mempool funded sums.
  - [x] AC2 If confirmed funded amount meets expected amount, status becomes `confirmed`.
  - [x] AC3 If mempool+chain amount meets expected but confirmed chain amount does not, status becomes `detected`.
- Notes: If expected amount is absent, threshold defaults to `> 0`.

### FR-005 - EVM observation via JSON-RPC

- Description: Reconciler must observe EVM request addresses using configured RPC URL per network.
- Acceptance criteria:
  - [x] AC1 Native asset (ETH) uses `eth_getBalance`.
  - [x] AC2 Token asset (e.g. ERC20) uses `eth_call balanceOf(address)` against token contract in request snapshot.
  - [x] AC3 Observed amount meeting threshold transitions request to `confirmed`.
- Notes: In phase-1, EVM `detected` is optional and may directly transition to `confirmed`.

### FR-006 - Status transition persistence

- Description: Reconciler must persist lifecycle updates safely.
- Acceptance criteria:
  - [x] AC1 Status update uses compare-and-set by `(id, current_status)`.
  - [x] AC2 Metadata stores latest reconciliation observation summary without leaking secrets.
  - [x] AC3 Transition paths allowed in phase-1 are: `pending->detected`, `pending->confirmed`, `detected->confirmed`, `pending->expired`, `detected->expired`.
- Notes: Unsupported/observer-missing requests remain open and are retried.

### FR-007 - Runtime worker loop

- Description: Service runtime must execute reconcile loop periodically when enabled.
- Acceptance criteria:
  - [x] AC1 Worker starts after persistence initialization succeeds.
  - [x] AC2 Worker stops gracefully on process shutdown context cancellation.
  - [x] AC3 Each cycle logs summary counters (scanned/confirmed/detected/expired/skipped/errors).
- Notes: Worker failures should not crash process by default; cycle-level errors are logged and retried next cycle.

## Non-functional requirements

- Reliability (NFR-001): Repeated reconciliation cycles with unchanged chain state do not introduce duplicate or invalid transitions.
- Performance (NFR-002): One reconciliation cycle of 100 rows completes within 5 seconds on local dev profile.
- Security/Privacy (NFR-003): No private keys or secret material logged/stored by reconciler.
- Observability (NFR-004): Cycle summary logs and per-transition logs must include request ID, tuple, and target status.
- Maintainability (NFR-005): New use case, repository, and observer logic must have unit/integration tests for key branches.

## Dependencies and integrations

- External systems: BTC Esplora HTTP API; EVM JSON-RPC endpoints.
- Internal services: payment request persistence/read model, app startup/runtime wiring, config parser.
