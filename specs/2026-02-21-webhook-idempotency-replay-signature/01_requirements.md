---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- webhook event id:
- Stable identifier for one outbox event; unchanged across delivery retries.
- replay nonce:
- Per-delivery-attempt random token included in signature input to detect replayed requests at receiver side.

## Out-of-scope behaviors

- OOS1: receiver-side data store implementation details for nonce cache.
- OOS2: retry jitter and retry budget policy.

## Functional requirements

### FR-001 - Stable downstream idempotency key contract

- Description: outbound webhook requests must expose a deterministic idempotency key contract tied to event identity.
- Acceptance criteria:
  - [x] AC1: webhook request includes `X-ChainTx-Event-Id` and `Idempotency-Key`, both set to the same stable event id.
  - [x] AC2: event id remains unchanged across retries for the same outbox event.
  - [x] AC3: webhook request includes `X-ChainTx-Delivery-Attempt` as a 1-based attempt counter for observability.
- Notes: at-least-once delivery remains unchanged; downstream must dedupe by idempotency key.

### FR-002 - Versioned anti-replay signature contract

- Description: outbound webhook requests must include a versioned anti-replay signature contract with signed timestamp and nonce.
- Acceptance criteria:
  - [x] AC1: webhook request includes `X-ChainTx-Timestamp`, `X-ChainTx-Nonce`, and `X-ChainTx-Signature-Version`.
  - [x] AC2: webhook request includes `X-ChainTx-Signature-V1` with `sha256=<hex>` derived via HMAC-SHA256 from canonical payload containing timestamp, nonce, event id, event type, and body.
  - [x] AC3: canonical payload format is deterministic and documented for receiver verification.
- Notes: keep existing `X-ChainTx-Signature` header for migration compatibility in this iteration.

### FR-003 - Preserve dispatch behavior and backward compatibility

- Description: adding idempotency/replay headers must not change existing dispatch status transitions and retry flow.
- Acceptance criteria:
  - [x] AC1: delivered/retry/failed state transitions remain identical.
  - [x] AC2: webhook gateway still sends JSON body and existing event headers.
  - [x] AC3: legacy signature header `X-ChainTx-Signature` is still emitted during migration window.
- Notes: this avoids breaking existing receivers immediately.

### FR-004 - Receiver verification guidance in repository docs

- Description: documentation must define concrete receiver-side verification and dedupe steps.
- Acceptance criteria:
  - [x] AC1: README documents required receiver checks: signature verification, timestamp window, nonce replay cache, and event id dedupe.
  - [x] AC2: README includes exact signed payload components and header mapping for `X-ChainTx-Signature-V1`.
- Notes: this is contract-level guidance; enforcement is receiver-side.

## Non-functional requirements

- Performance (NFR-001): additional signature/nonce processing must not materially change dispatch latency.
- Availability/Reliability (NFR-002): idempotency/replay headers must be present on every outbound webhook attempt.
- Security/Privacy (NFR-003): signature inputs and logs must not expose HMAC secret; only non-sensitive metadata may be logged.
- Compliance (NFR-004): not applicable in this iteration.
- Observability (NFR-005): attempt number and stable event id must enable receiver and operator troubleshooting.
- Maintainability (NFR-006): changes remain within existing application port + outbound gateway boundaries with no architecture layer violations.

## Dependencies and integrations

- External systems: downstream webhook receivers implementing dedupe and replay checks.
- Internal services: dispatch webhook use case, webhook HTTP gateway, runtime config/README documentation.
