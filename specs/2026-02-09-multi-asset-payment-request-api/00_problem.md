---
doc: 00_problem
spec_date: 2026-02-09
slug: multi-asset-payment-request-api
mode: Full
status: READY
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

# Problem & Goals

## Context

- Background: The current ChainTx service exposes only system endpoints (`GET /healthz`, Swagger) and startup-time DB bootstrap; it has no payment-facing API yet.
- Users or stakeholders: Merchant platform engineers, backend/API engineers, QA, and wallet operations teams.
- Why now: The next delivery phase needs a stable payment intake surface that can issue deposit instructions for BTC, ETH, and USDT on Ethereum without API redesign per asset.

## Constraints (optional)

- Technical constraints: Preserve existing Clean Architecture + Hexagonal boundaries (`internal/domain`, `internal/application`, `internal/adapters`, `internal/bootstrap`).
- Technical constraints: API must remain REST-style resource modeling; avoid action endpoints like `/generate-address`.
- Technical constraints: `expected_amount_minor` must be integer-safe and transported as a string in API contracts.
- Technical constraints: Address allocation and index increment must be atomic under concurrency.
- Technical constraints: Idempotent create behavior is fixed as: first write returns `201`, replay returns `200` with `X-Idempotency-Replayed: true`, and both include `Location`.
- Technical constraints: Idempotency record TTL baseline is `7d`, with rule `TTL >= max(request.expires_in_seconds, 24h)`.
- Technical constraints: `expires_in_seconds` accepted input range is `60..2592000` (60 seconds to 30 days), and `expires_at` is always non-null.
- Compliance/security constraints: Wallet secret material cannot be exposed via API, logs, or DB application tables.
- Timeline/cost constraints: v1 scope is only "create payment request + return payment instructions + query request"; blockchain monitoring/webhook is deferred.

## Problem statement

- Current pain: There is no canonical resource to represent a receivable payment intent, so address allocation, expiry, and reconciliation cannot be tracked consistently.
- Current pain: Multi-asset support is likely to fragment into per-asset APIs unless contract/data model is unified now.
- Evidence or examples: Proposed integrations currently describe "generate address" as an action, but downstream monitoring, reconciliation, and idempotent retries require a stable resource identity.

## Goals

- G1: Define `Payment Request` as the canonical REST resource for inbound crypto payments.
- G2: Provide one unified API model supporting BTC, ETH, and USDT-ERC20 on Ethereum mainnet in v1.
- G3: Guarantee deterministic/idempotent creation behavior and duplicate-free address/index allocation under concurrent requests.
- G4: Publish asset/network metadata via discovery endpoint so clients do not hardcode decimals/contract details.
- G5: Keep contract extensible so future assets/chains can be added via configuration and discovery, not endpoint proliferation.

## Non-goals (out of scope)

- NG1: Blockchain confirmation tracking, state transitions from on-chain events, and webhook/event delivery.
- NG2: Merchant settlement, refunds, or payout workflows.
- NG3: Non-Ethereum USDT rails (for example TRON) in this increment.
- NG4: End-user wallet UX concerns (QR generation/branding).

## Assumptions

- A1: Existing PostgreSQL foundation from `2026-02-07-postgresql-backend-service-compose` is available and remains the source of truth for request/index state.
- A2: Each `(chain, network, asset)` resolves to exactly one configured wallet account, and multiple tuples may resolve to the same wallet account (for example ETH + USDT on Ethereum).
- A3: Address derivation runs through server-side wallet infrastructure; API never accepts caller-provided addresses.
- A4: USDT-ERC20 contract metadata must come from server-controlled asset catalog; any initial value is treated as untrusted until operations confirmation.
- A5: API authentication/authorization approach exists at platform level but is specified outside this document.

## Open questions

- Q1: What is the authoritative operational process for approving/updating token contract metadata before enabling an asset in production?
- Q2: Is `expected_amount_minor` optional for all assets, or required for specific merchant/account configurations?

## Success metrics

- Metric: Functional coverage
- Target: `POST /v1/payment-requests` supports BTC, ETH, and USDT (Ethereum/ERC20) on mainnet with one request schema.
- Metric: Concurrency safety
- Target: Under >=200 concurrent create requests for same asset tuple, no duplicate `(wallet_account_id, derivation_index)` and no duplicate `(chain, network, address_canonical)` are created.
- Metric: Idempotency correctness
- Target: Replayed request with same `Idempotency-Key` + same body returns same `payment_request.id` and same payment instructions 100% of the time.
- Metric: Client decoupling
- Target: Clients can discover decimals and token metadata from `GET /v1/assets` without hardcoded constants.
