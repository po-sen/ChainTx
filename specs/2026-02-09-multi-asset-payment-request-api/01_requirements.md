---
doc: 01_requirements
spec_date: 2026-02-09
slug: multi-asset-payment-request-api
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-07-postgresql-backend-service-compose
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Requirements

## Glossary (optional)

- Payment Request: A server-managed receivable resource representing one inbound payment intent and its instructions.
- Payment Instructions: Chain/asset-specific transfer instructions returned to payer (address plus required metadata).
- Minor unit: Integer amount unit per asset (for example `sats`, `wei`, token smallest unit).
- Decimals: Number of decimal places from major unit to minor unit (BTC=`8`, ETH=`18`, USDT=`6`).
- Wallet account: Server-side allocation cursor keyed by chain/network/keyset; one or more assets can map to the same wallet account.

## Out-of-scope behaviors

- OOS1: Automatic transition to `paid` from blockchain monitoring.
- OOS2: Webhooks/event streaming for payment status updates.
- OOS3: Support for USDT on non-Ethereum networks in this increment.

## Functional requirements

### FR-001 - Expose supported assets and network metadata

- Description: API must provide discovery endpoint for supported `(chain, network, asset)` combinations and unit metadata.
- Acceptance criteria:
  - [ ] AC1: `GET /v1/assets` returns a list of supported assets with required fields: `chain`, `network`, `asset`, `minor_unit`, `decimals`, `address_scheme`, `default_expires_in_seconds`.
  - [ ] AC2: For tokenized assets, response includes token metadata fields (`token_standard`, `token_contract`, `token_decimals`) and chain identifiers where relevant (for example `chain_id` for EVM).
  - [ ] AC3: Discovery data is sourced from server-owned catalog/configuration, not inferred by client input.
  - [ ] AC4: `decimals` is defined as major-unit exponent (for example BTC=`8`, ETH=`18`, USDT=`6`) and must not be interpreted as minor-unit precision.
  - [ ] AC5: Startup validation fails service startup when an enabled asset catalog row references a missing/inactive wallet account, has incompatible allocator configuration, or has `default_expires_in_seconds` outside `[60, 2592000]`.
- Notes: Asset catalog values that involve external references (for example token contract) must be operationally verified before production enablement.

### FR-002 - Create payment request resource via REST collection

- Description: API must create a `Payment Request` resource at `POST /v1/payment-requests` and allocate instructions at creation time.
- Acceptance criteria:
  - [ ] AC1: Request body supports `chain`, `network`, `asset`, optional `expected_amount_minor` (string integer), optional `expires_in_seconds`, and optional `metadata` object.
  - [ ] AC2: Successful creation returns `201 Created` with `Location: /v1/payment-requests/{id}` and response body containing `id`, `status`, request attributes, timestamps, and `payment_instructions`.
  - [ ] AC3: `status` is persisted and initialized to `pending`.
  - [ ] AC4: If `expires_in_seconds` is omitted, server uses default from asset catalog entry.
  - [ ] AC5: `expires_at` is always set (non-null) and computed as `created_at + resolved_expires_in_seconds`.
  - [ ] AC6: If provided, `expires_in_seconds` must be an integer in `[60, 2592000]` (60 seconds to 30 days); out-of-range values return `400 invalid_request`.
- Notes: v1 does not support non-expiring requests; `expected_amount_minor` must not accept floating-point or scientific notation.

### FR-003 - Provide polymorphic payment instructions in one schema

- Description: Response must use one `payment_instructions` object that can represent BTC, ETH, and ERC20 token instructions without API branching.
- Acceptance criteria:
  - [ ] AC1: BTC instructions include at least `address`, `address_scheme`, and `derivation_index`.
  - [ ] AC2: ETH instructions include at least `address`, `address_scheme`, `derivation_index`, and `chain_id`.
  - [ ] AC3: USDT-ERC20 instructions include EVM address fields plus token metadata (`token_standard`, `token_contract`, `token_decimals`).
  - [ ] AC4: Returned address format is validated against chain/network rules before persistence.
  - [ ] AC5: EVM address canonicalization is deterministic: database uniqueness compares lowercase `0x`-hex form, and API response returns EIP-55 checksum format derived from that canonical value.
  - [ ] AC6: Bitcoin canonicalization is deterministic: bech32 addresses are stored/returned in lowercase, and base58 addresses are stored as-is after base58check validation.
- Notes: For USDT on Ethereum, transfer destination is EVM address; token metadata is mandatory to prevent transfer ambiguity.

### FR-004 - Query payment request by id

- Description: API must expose `GET /v1/payment-requests/{id}` to retrieve full request details.
- Acceptance criteria:
  - [ ] AC1: Existing id returns canonical resource representation consistent with create response fields.
  - [ ] AC2: Missing id returns `404` with standardized error body and machine-readable code.
- Notes: This endpoint is required for reconciliation/debugging even before chain monitoring is implemented.

### FR-005 - Enforce idempotent create semantics

- Description: `POST /v1/payment-requests` must support `Idempotency-Key` and deterministic replay behavior.
- Acceptance criteria:
  - [ ] AC1: Same `Idempotency-Key` + semantically same request body returns same `payment_request.id` and same `payment_instructions`.
  - [ ] AC2: Same `Idempotency-Key` + different request body returns `409 Conflict` with code `idempotency_key_conflict`.
  - [ ] AC3: Idempotency record stores canonical request hash and replay payload so timeout/retry does not allocate a second address.
  - [ ] AC4: Idempotency scope includes authenticated principal identity (or equivalent), HTTP method, and normalized path to prevent cross-principal or cross-endpoint key collisions.
  - [ ] AC5: Canonical request hashing uses RFC 8785 JSON Canonicalization Scheme (JCS) over UTF-8 bytes with SHA-256.
  - [ ] AC6: First successful create returns `201 Created`; replay returns `200 OK`; both return `Location` and replay includes `X-Idempotency-Replayed: true`.
- Notes: Header is strongly recommended for all create calls; platform may enforce as required in later phase.

### FR-006 - Guarantee atomic address allocation and uniqueness

- Description: Allocation of derivation index/address and creation of payment request must be transactionally atomic.
- Acceptance criteria:
  - [ ] AC1: For each create request, wallet account row is locked and exactly one `next_index` value is consumed.
  - [ ] AC2: DB enforces uniqueness for `(wallet_account_id, derivation_index)` and `(chain, network, address_canonical)`.
  - [ ] AC3: On any failure inside create flow, transaction is rolled back and `next_index` is not advanced.
  - [ ] AC4: Concurrent create requests do not produce duplicate address or index values.
  - [ ] AC5: Distinct asset tuples on the same chain/network (for example ETH and USDT on Ethereum mainnet) may share one wallet allocation cursor without violating address uniqueness.
- Notes: This requirement is mandatory for correct later reconciliation and on-chain monitoring.

### FR-007 - Validate requests and return consistent error schema

- Description: Validation and domain errors must map to one stable JSON error format.
- Acceptance criteria:
  - [ ] AC1: Validation failures (bad enum, bad amount format, oversized amount, metadata too large, invalid expiry range) return `400` with code `invalid_request`.
  - [ ] AC2: Unsupported combinations (for example `asset=USDT` on `bitcoin`) return `400` with code `unsupported_asset` or `unsupported_network`.
  - [ ] AC3: Error response body shape is `{ "error": { "code": string, "message": string, "details": object } }`.
  - [ ] AC4: If provided, `expected_amount_minor` must match `^[0-9]{1,78}$` and parse to an integer within DB precision (`NUMERIC(78,0)`).
  - [ ] AC5: If provided, `expires_in_seconds` must parse as integer and satisfy `60 <= expires_in_seconds <= 2592000`.
- Notes: Error code taxonomy should remain stable because client retries and UX rely on it.

### FR-008 - Define stable status model and timestamps for future lifecycle

- Description: Resource schema must reserve lifecycle fields needed for future monitoring while emitting only `pending` in this phase.
- Acceptance criteria:
  - [ ] AC1: Response includes `status`, `created_at`, and non-null `expires_at`; `expires_at` is always computed from resolved expiry seconds in v1.
  - [ ] AC2: `status` is modeled as extensible string; currently known values include `pending`, `observed`, `confirmed`, `paid`, `partial`, `overpaid`, `expired`, `failed`, and `reorged`, while v1 only emits `pending`.
  - [ ] AC3: OpenAPI v1 must not use a closed enum that would force breaking changes when new status values are introduced.
- Notes: Future webhook/state transitions must reuse this status field rather than introducing a parallel state model.

### FR-009 - Publish API contract in OpenAPI

- Description: OpenAPI contract must document v1 endpoints, schemas, and examples for BTC/ETH/USDT.
- Acceptance criteria:
  - [ ] AC1: `api/openapi.yaml` includes `GET /v1/assets`, `POST /v1/payment-requests`, and `GET /v1/payment-requests/{id}`.
  - [ ] AC2: Schema definitions describe `expected_amount_minor` as string integer and include polymorphic `payment_instructions` examples, including EVM checksum-format address responses.
  - [ ] AC3: Error responses are documented for 400/404/409 cases with matching error code examples.
  - [ ] AC4: OpenAPI documents idempotency replay behavior (`201` first write, `200` replay, `X-Idempotency-Replayed` header) and describes `status` as extensible.
  - [ ] AC5: OpenAPI documents `expires_in_seconds` bounds (`minimum: 60`, `maximum: 2592000`) and non-null `expires_at`.
- Notes: Swagger UI path remains `/swagger/index.html` as current service convention.

## Non-functional requirements

- Performance (NFR-001): For local dev profile, p95 latency <= 200ms for `GET /v1/assets` and <= 300ms for `POST /v1/payment-requests`/`GET /v1/payment-requests/{id}` under 20 RPS single-instance test load (excluding external node calls).
- Availability/Reliability (NFR-002): Under a 200-request concurrent create test per asset tuple plus cross-asset ETH/USDT mixed load, duplicate-address and duplicate-index count must be exactly `0`.
- Security/Privacy (NFR-003): No endpoint or logs may expose private keys, seed phrases, or full secret-bearing wallet configuration; metadata payload size must be bounded to <= 4KB.
- Compliance (NFR-004): Any production token contract entry must be traceable to an approved source-of-truth change record before enablement.
- Observability (NFR-005): Structured logs and metrics must include request id, idempotency replay indicator, allocation attempt outcome, and allocation latency.
- Maintainability (NFR-006): New feature code must preserve current clean architecture boundaries and pass `go fmt ./...`, `go vet ./...`, `go list ./...`, and `go test ./...`.

## Dependencies and integrations

- External systems: Wallet derivation/signing infrastructure (exact provider TBD), blockchain explorers/nodes are not required in this phase.
- Internal services: Existing PostgreSQL persistence stack and app bootstrap/DI pipeline.
- Configuration data: Server-controlled asset catalog for supported chain/network/asset metadata.
