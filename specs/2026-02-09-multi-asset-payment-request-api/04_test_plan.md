---
doc: 04_test_plan
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

# Test Plan

## Scope

- Covered:
  - Asset discovery API contract and data integrity.
  - Payment request creation/query across BTC, ETH, and USDT-ERC20.
  - Idempotency replay/conflict behavior.
  - Transactional uniqueness under concurrent address allocation.
  - Error mapping, observability signals, and OpenAPI consistency.
- Not covered:
  - Blockchain confirmation ingestion and webhook delivery.
  - Production wallet key management implementation details.
  - Non-Ethereum USDT rails.

## Tests

### Unit

- TC-001: Create request validator and amount parsing

  - Linked requirements: FR-002, FR-007, NFR-003
  - Steps: Exercise request validator with valid payloads and invalid cases (bad enum, negative amount, float amount string, non-digit amount string, >78-digit amount string, oversize metadata, `expires_in_seconds < 60`, `expires_in_seconds > 2592000`).
  - Expected: Valid input passes; invalid input returns typed `invalid_request` mapping data.

- TC-002: Payment instruction mapper by asset type

  - Linked requirements: FR-003, FR-008
  - Steps: Unit-test instruction mapping for BTC, ETH, and USDT inputs.
  - Expected: Output fields include only required per-asset metadata and always include canonical shared fields.

- TC-003: Idempotency hash normalization
  - Linked requirements: FR-005, NFR-002
  - Steps: Compare hash outputs for semantically equivalent JSON payloads with reordered keys/whitespace differences using RFC 8785 JCS canonicalization fixtures.
  - Expected: Canonical hash remains identical for equivalent payloads and differs for material field changes.

### Integration

- TC-101: Asset discovery endpoint from seeded catalog

  - Linked requirements: FR-001, NFR-004
  - Steps: Seed DB catalog and call `GET /v1/assets`.
  - Expected: Response contains only enabled entries with complete metadata and no missing required fields; `decimals` follows major-unit exponent semantics (BTC=`8`, ETH=`18`, USDT=`6`) and `default_expires_in_seconds` is within `[60, 2592000]`.

- TC-102: BTC payment request create + read

  - Linked requirements: FR-002, FR-003, FR-004, FR-008
  - Steps: Call `POST /v1/payment-requests` for BTC, then `GET /v1/payment-requests/{id}`.
  - Expected: Create returns `201` and `Location`; read returns same id, status `pending`, non-null `expires_at`, and BTC instruction fields.

- TC-103: ETH payment request create + read

  - Linked requirements: FR-002, FR-003, FR-004
  - Steps: Call create/read for ETH on Ethereum mainnet.
  - Expected: Instruction payload includes `chain_id` and EVM address format.

- TC-104: USDT-ERC20 payment request create + read

  - Linked requirements: FR-002, FR-003, FR-004, NFR-004
  - Steps: Call create/read for USDT on Ethereum mainnet.
  - Expected: Instruction payload includes `token_standard`, `token_contract`, and `token_decimals` from catalog.

- TC-105: Idempotency replay same key + same body

  - Linked requirements: FR-005, FR-006, NFR-002
  - Steps: Submit same create request twice with identical `Idempotency-Key`.
  - Expected: Second response reuses same resource and does not consume a new derivation index.

- TC-106: Idempotency conflict same key + different body

  - Linked requirements: FR-005, FR-007
  - Steps: Submit two create requests with same key but different payload.
  - Expected: Second request returns `409 idempotency_key_conflict`; no new payment request row is inserted.

- TC-107: Unsupported asset/network validation

  - Linked requirements: FR-001, FR-007
  - Steps: Submit unsupported tuple (for example USDT on bitcoin/mainnet).
  - Expected: Request fails with `400` and code `unsupported_asset` or `unsupported_network`.

- TC-108: Concurrent allocation uniqueness

  - Linked requirements: FR-006, NFR-002
  - Steps: Fire >=200 concurrent create requests for same asset tuple with unique idempotency keys.
  - Expected: No duplicate `(wallet_account_id, derivation_index)` or `(chain, network, address_canonical)` rows.

- TC-109: Cross-asset collision test (ETH vs USDT shared allocator)

  - Linked requirements: FR-006, NFR-002
  - Steps: Submit >=50 ETH + >=50 USDT create requests concurrently on Ethereum mainnet with unique idempotency keys.
  - Expected: No address collisions across ETH and USDT requests under `(chain, network, address_canonical)` uniqueness.

- TC-110: Address canonicalization contract

  - Linked requirements: FR-003, FR-006
  - Steps: For EVM requests, verify DB stores lowercase canonical addresses and API returns deterministic EIP-55 checksum addresses; for BTC, verify network-specific address format validity and canonicalization rules.
  - Steps: Validate EIP-55 fixtures:
    - `0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed` -> `0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed`
    - `0xfb6916095ca1df60bb79ce92ce3ea74c37c5d359` -> `0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359`
    - `0xdbf03b407c01e7cd3cbea99509d93f8dddc8c6fb` -> `0xdbF03B407c01E7cd3CBea99509d93f8DDDC8c6Fb`
  - Expected: Storage and response canonicalization rules match spec; uniqueness checks are performed against canonical form.

- TC-111: Idempotency replay HTTP semantics

  - Linked requirements: FR-005, FR-009
  - Steps: Call create twice with same key and same payload; inspect status code and headers.
  - Expected: First response is `201`; second is `200` with `X-Idempotency-Replayed: true`; both include `Location`.

- TC-112: Startup catalog-mapping validation gate
  - Linked requirements: FR-001, FR-006
  - Steps: Seed one enabled asset row pointing to missing/inactive wallet account or set out-of-range `default_expires_in_seconds`, then start service.
  - Expected: Service startup fails before serving traffic with explicit configuration error.

### E2E (if applicable)

- TC-201: End-to-end merchant flow (v1 scope)

  - Linked requirements: FR-001, FR-002, FR-004, FR-009
  - Steps: Discover assets, create payment request, retrieve by id, inspect Swagger docs for same contract.
  - Expected: Runtime behavior and OpenAPI documentation match exactly.

- TC-202: Failure-path observability smoke test
  - Linked requirements: FR-007, NFR-005
  - Steps: Trigger validation error, idempotency conflict, and internal adapter failure in non-prod environment.
  - Expected: Logs/metrics include stable error code and operation phase; no secret leakage.

## Edge cases and failure modes

- Case: `expires_in_seconds` omitted.
- Expected behavior: Server applies catalog default and returns computed non-null `expires_at`. (FR-002, FR-008)

- Case: `expires_in_seconds` outside allowed range.
- Expected behavior: Request is rejected with `400 invalid_request`; no DB rows are created. (FR-002, FR-007)

- Case: Create transaction fails after address derivation but before commit.
- Expected behavior: No partial row commit and no irreversible index drift beyond transaction semantics. (FR-006)

- Case: Same address is produced unexpectedly by adapter due upstream bug.
- Expected behavior: DB unique constraint blocks duplicate and request fails with internal conflict classification. (FR-006, NFR-002)

- Case: Metadata payload exceeds size bound.
- Expected behavior: Request rejected with `400 invalid_request` and bounded error details. (FR-007, NFR-003)

## NFR verification

- Performance:
  - Run load test at 20 RPS and verify p95 thresholds from NFR-001 for all three endpoints.
- Reliability:
  - Execute TC-108 and TC-109 and verify duplicate counters remain zero; ensure replay path in TC-105/TC-111 does not allocate new index.
- Security:
  - Scan logs for forbidden secret patterns; confirm no private key/seed fragments are emitted.
- Compliance:
  - Verify USDT contract metadata in asset catalog maps to approved change record before enabling production flag.
- Observability:
  - Verify metrics/log labels required by NFR-005 are emitted and queryable.
- Maintainability:
  - Ensure CI quality gates (`go fmt`, `go vet`, `go list`, `go test`) pass with added code paths.
