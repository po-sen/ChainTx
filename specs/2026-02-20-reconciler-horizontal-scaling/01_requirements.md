---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- Lease: temporary DB ownership for one worker to process an open payment request.
- Claim: atomic operation that selects open rows and writes lease owner/expiry.
- Worker role: runtime process that only executes reconciler loop (no HTTP server).

## Out-of-scope behaviors

- OOS1: queue-based architecture.
- OOS2: tx-level settlement proof redesign.

## Functional requirements

### FR-001 - Dedicated reconciler runtime role

- Description: system must provide a standalone reconciler executable suitable for a separate container.
- Acceptance criteria:
  - [x] AC1 executable starts persistence initialization and reconcile loop without starting HTTP listener.
  - [x] AC2 process exits cleanly on shutdown signal and logs worker lifecycle events.
  - [x] AC3 existing server executable remains operational.
- Notes: deployment should support app and reconciler as separate services.

### FR-002 - Atomic open-request claiming

- Description: reconciliation must atomically claim open rows before chain observation.
- Acceptance criteria:
  - [x] AC1 claim query includes only `pending` and `detected` requests with no active lease.
  - [x] AC2 claim operation is deterministic (`created_at,id`) and limited by batch size.
  - [x] AC3 each claimed row stores lease owner and lease expiry timestamp.
- Notes: claim should use PostgreSQL locking primitives to avoid duplicate ownership.

### FR-003 - Lease-aware status transition

- Description: status transitions must be protected by current status and lease ownership.
- Acceptance criteria:
  - [x] AC1 transition update uses compare-and-set on current status.
  - [x] AC2 transition update only succeeds when lease owner matches claimer (or no lease exists).
  - [x] AC3 successful transition clears lease fields.
- Notes: maintain idempotency under concurrent workers.

### FR-004 - Reconciler lease configuration

- Description: reconciler must support configurable lease duration and stable worker identity.
- Acceptance criteria:
  - [x] AC1 `PAYMENT_REQUEST_RECONCILER_LEASE_SECONDS` is parsed and validated as positive integer.
  - [x] AC2 worker identity is included in each cycle command and logs.
  - [x] AC3 invalid lease config fails startup with explicit config error code.
- Notes: worker ID may be explicit env or runtime-generated fallback.

### FR-005 - Parallel deployment ergonomics

- Description: local/docker workflow must support enabling and scaling dedicated reconciler replicas.
- Acceptance criteria:
  - [x] AC1 service compose includes dedicated reconciler service.
  - [x] AC2 make workflow can start reconciler with configurable replica count.
  - [x] AC3 app service can run without embedded reconciler by default in this workflow.
- Notes: no manual YAML editing should be required for normal scaling.

### FR-006 - Observability for claimed work

- Description: logs must expose claim and cycle outcomes for multi-replica troubleshooting.
- Acceptance criteria:
  - [x] AC1 cycle summary logs include worker ID and claimed count.
  - [x] AC2 transition logs retain request ID and target status.
  - [x] AC3 claim failures are logged with structured error code/message.
- Notes: logs must not include key material or private keys.

## Non-functional requirements

- Concurrency safety (NFR-001): with 3 worker replicas, the same open request is not concurrently claimed by more than one worker within an active lease window.
- Performance (NFR-002): one cycle with batch size 100 must complete under 5 seconds on local dev profile (excluding external chain latency spikes).
- Reliability (NFR-003): if one worker stops unexpectedly, unprocessed claimed rows become claimable again after lease timeout.
- Security/Privacy (NFR-004): no secret material is persisted in new lease fields or logs.
- Observability (NFR-005): worker logs include role/worker_id/claimed/scanned/confirmed/detected/expired/skipped/errors.
- Maintainability (NFR-006): unit tests cover claim selection, lease checks, worker role startup, and config parsing.

## Dependencies and integrations

- External systems: PostgreSQL locking behavior (`FOR UPDATE SKIP LOCKED`), existing chain observer endpoints.
- Internal services: current reconciler use case, persistence bootstrap, docker compose and make workflows.
