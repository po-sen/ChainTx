---
doc: 02_design
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

# Technical Design

## High-level approach

- Summary: Add application-level reconcile use case executed by an optional background worker; use outbound adapters for DB transitions and chain observations.
- Key decisions:
  - Keep reconciler disabled by default.
  - Use polling in phase-1 with configurable interval/batch.
  - Use compare-and-set status updates for idempotency.

## System context

- Components:
  - `internal/application/use_cases/reconcile_payment_requests_use_case.go`
  - `internal/application/ports/out/payment_request_reconciliation_repository.go`
  - `internal/application/ports/out/payment_chain_observer_gateway.go`
  - `internal/adapters/outbound/persistence/postgresql/paymentrequest/*` reconciliation methods
  - `internal/adapters/outbound/chainobserver/devtest/*` HTTP observers
  - `internal/infrastructure/reconciler/worker.go` runtime loop
- Interfaces:
  - Reconcile use case takes open request rows and delegates chain observation and status transitions through ports.

## Key flows

- Flow 1: request expires
  - fetch open row
  - `expires_at <= now`
  - CAS update to `expired`
- Flow 2: BTC detected then confirmed
  - fetch Esplora address stats
  - if mempool+chain >= expected and chain < expected => `detected`
  - if chain >= expected => `confirmed`
- Flow 3: EVM confirmed
  - query balance (`eth_getBalance` or `eth_call balanceOf`)
  - if balance >= expected => `confirmed`

## Data model

- Entities:
  - Existing `app.payment_requests` status/metadata fields are reused.
- Schema changes or migrations:
  - Add migration for status/index support:
    - allowed status check constraint: `pending|detected|confirmed|expired|failed`
    - index on `(status, expires_at)` for polling.
- Consistency and idempotency:
  - CAS update condition `WHERE id=$1 AND status=$2`.

## API or contracts

- New env vars:
  - `PAYMENT_REQUEST_RECONCILER_ENABLED` (bool, default `false`)
  - `PAYMENT_REQUEST_RECONCILER_POLL_INTERVAL_SECONDS` (int, default `15`)
  - `PAYMENT_REQUEST_RECONCILER_BATCH_SIZE` (int, default `100`)
  - `PAYMENT_REQUEST_BTC_ESPLORA_BASE_URL` (optional)
  - `PAYMENT_REQUEST_EVM_RPC_URLS_JSON` (optional JSON object: `{"local":"http://..."}`)

## Backward compatibility (optional)

- API compatibility: no HTTP response schema change.
- Data migration compatibility: existing `pending` rows remain valid and become eligible for reconciliation.

## Failure modes and resiliency

- Retries/timeouts:
  - HTTP observers use per-request timeout.
  - Worker continues cycles after non-fatal observer failures.
- Degradation strategy:
  - Missing observer endpoint for chain/network causes request skip (no crash).

## Observability

- Logs:
  - cycle summary counters
  - transition logs with request_id/status_from/status_to
- Metrics:
  - deferred to future phase (logs only for now)
- Alerts:
  - startup config errors fail fast if invalid reconciler settings.

## Security

- Authentication/authorization: no new public endpoint.
- Secrets: observer config contains only public endpoint URLs.
- Abuse cases: malformed external payloads are treated as observer errors and skipped.

## Alternatives considered

- Option A: event-driven websocket listeners.
- Option B: polling worker (chosen for phase-1 simplicity and deterministic retries).
- Why chosen: lower operational complexity and easier testability in local/devtest.

## Risks

- Risk: balance-based observation may over-count if address is reused externally.
- Mitigation: service allocates unique derived addresses per request; future phase can add tx-level proofing.
