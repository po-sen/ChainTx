---
doc: 02_design
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

# Technical Design

## High-level approach

- Summary: split runtime into API server and dedicated reconciler process, and switch reconciliation selection from plain list query to atomic claim+lease.
- Key decisions:
  - add `cmd/reconciler` as standalone worker runtime.
  - use PostgreSQL row claim with `FOR UPDATE SKIP LOCKED` and lease columns.
  - keep compare-and-set transitions and make them lease-aware.
  - keep embedded worker capability but set compose workflow default to dedicated worker service.

## System context

- Components:
  - `cmd/reconciler/main.go` (new runtime role)
  - `internal/infrastructure/reconciler/worker.go` (worker_id + lease_duration command fields)
  - `internal/application/use_cases/reconcile_payment_requests_use_case.go` (claim-first orchestration)
  - `internal/application/ports/out/payment_request_reconciliation_repository.go` (claim contract)
  - `internal/adapters/outbound/persistence/postgresql/paymentrequest/reconciliation_repository.go` (claim SQL + lease-aware CAS transition)
  - `deployments/service/docker-compose.yml` + `Makefile` (dedicated reconciler service and replica scaling)
- Interfaces:
  - repository: `ClaimOpenForReconciliation(ctx, now, limit, leaseOwner, leaseUntil)`
  - repository: `TransitionStatusIfCurrent(...)` with lease owner guard.

## Key flows

- Flow 1 (claim):
  - worker cycle computes `leaseUntil = now + leaseDuration`
  - repository atomically claims up to `batchSize` open rows with expired/no lease
  - claimed rows are returned with processing payload.
- Flow 2 (process):
  - for each claimed row, observer computes detected/confirmed state
  - use case executes lease-aware CAS transition for status changes.
- Flow 3 (recovery):
  - if worker crashes mid-cycle, lease naturally expires and row becomes claimable again.

## Diagrams (optional)

- Mermaid sequence / flow:

## Data model

- Entities:
  - `app.payment_requests` gains lease metadata for reconciler ownership.
- Schema changes or migrations:
  - add nullable columns:
    - `reconcile_lease_owner text`
    - `reconcile_lease_until timestamptz`
  - add index optimized for claim query:
    - `(status, reconcile_lease_until, created_at, id)`
- Consistency and idempotency:
  - claim is atomic and lock-safe (`FOR UPDATE SKIP LOCKED`).
  - transition remains CAS on current status and verifies lease owner.
  - successful transition clears lease fields.

## API or contracts

- Endpoints or events:
  - no public API changes.
  - new internal runtime command input fields:
    - `WorkerID`
    - `LeaseDuration`
- Request/response examples:
  - N/A for external clients.

## Backward compatibility (optional)

- API compatibility: unchanged.
- Data migration compatibility: existing rows get null lease fields and remain eligible for claiming.

## Failure modes and resiliency

- Retries/timeouts:
  - observer failures increment cycle errors and keep row open.
- Backpressure/limits:
  - batch size and lease duration bound per-cycle work and reduce duplicate scans.
- Degradation strategy:
  - if dedicated reconciler is disabled, API service continues as before.

## Observability

- Logs:
  - worker start/stop with worker_id, poll interval, lease duration.
  - cycle summary includes claimed/scanned/confirmed/detected/expired/skipped/errors.
- Metrics:
  - deferred; logs only in this phase.
- Traces:
  - not added in this phase.
- Alerts:
  - startup config parse failures remain fail-fast.

## Security

- Authentication/authorization: no new external interface.
- Secrets: lease fields store only worker ID and timestamps.
- Abuse cases: malformed observer responses remain non-fatal per row and are retried.

## Alternatives considered

- Option A: keep embedded worker only and rely on CAS transitions.
- Option B: add queue broker for work partition.
- Why chosen: DB lease claim is simplest change with immediate horizontal scaling benefit and no extra infrastructure.

## Risks

- Risk: long lease duration can delay retries after worker crash.
- Mitigation: configurable lease seconds with conservative default and explicit docs.
