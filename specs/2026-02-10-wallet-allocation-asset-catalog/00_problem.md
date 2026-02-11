---
doc: 00_problem
spec_date: 2026-02-10
slug: wallet-allocation-asset-catalog
mode: Full
status: READY
owners:
  - posen
depends_on:
  - 2026-02-09-multi-asset-payment-request-api
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Problem & Goals

## Context

- Background: Current `Payment Request` specification depends on real wallet-derived addresses, but implementation still lacks allocator foundations (`wallet_accounts`, `asset_catalog`, startup mapping validation, deterministic derivation adapter).
- Users or stakeholders: Backend/API engineers, wallet operations, QA, and merchants integrating `POST /v1/payment-requests`.
- Why now: Without this feature, API responses can return address-like strings that are not guaranteed to be controlled by our wallet infrastructure.

## Constraints (optional)

- Technical constraints: Must preserve Clean Architecture + Hexagonal boundaries and keep wallet derivation behind outbound ports.
- Technical constraints: Asset catalog must support shared allocator cursor for ETH and USDT on same EVM network.
- Technical constraints: Allocation must remain transactional (`FOR UPDATE` cursor lock + atomic index consumption).
- Technical constraints: Derivation index handling must be 64-bit safe end-to-end (`BIGINT` in DB, non-negative in application) with no 32-bit narrowing.
- Technical constraints: `devtest` network scope is fixed to BTC `regtest`/`testnet` and Ethereum `sepolia` for this feature.
- Technical constraints: Canonicalization responsibility is in application layer (adapter derives raw valid address; application canonicalizes for storage/response).
- Timeline/cost constraints: Deliver a dev/test-capable adapter first; production HSM/wallet-service adapter can remain placeholder.
- Compliance/security constraints: No private key or seed material may be persisted in business tables or logs.

## Problem statement

- Current pain: Allocation cursor and asset-to-wallet mapping are not persisted as first-class data, so correctness rules cannot be enforced at startup/runtime.
- Current pain: Wallet adapter is not yet capable of deterministic, recoverable derivation for BTC and EVM in dev/test.
- Evidence or examples: `POST /v1/payment-requests` cannot safely claim a returned address is spendable by operator-controlled keys without this foundation.

## Goals

- G1: Introduce `wallet_accounts` and `asset_catalog` schema + seed with strict integrity checks.
- G2: Enforce startup validation for enabled catalog rows and wallet-account mapping compatibility.
- G3: Deliver `WalletAllocationGateway` implementation for dev/test that derives real BTC/EVM addresses from controlled public derivation material.
- G4: Integrate transactional allocator cursor consumption path so create flow can atomically consume index and persist payment request.
- G5: Prove uniqueness/correctness with single-request smoke tests and concurrent allocation tests.

## Non-goals (out of scope)

- NG1: Production HSM or external wallet service final integration.
- NG2: Blockchain monitoring, payment state transitions, and webhook delivery.
- NG3: New public API surface beyond current payment-request flows.

## Assumptions

- A1: Upstream spec `2026-02-09-multi-asset-payment-request-api` is the API source of truth; folder-wide frontmatter status was verified as `DONE` on 2026-02-10.
- A2: Dev/test environments can supply deterministic public derivation material through configuration.
- A3: ETH and USDT for the same EVM network are expected to share one allocator cursor (`wallet_account_id`).
- A4: Existing `payment_requests` persistence model (including uniqueness constraints) is available or will be implemented in the same implementation wave.

## Open questions

- Q1: What is the approved operational format for storing and rotating dev/test keyset configuration in CI/CD (per-key env vars vs. secret-backed JSON blob)?

## Success metrics

- Metric: Schema and seed correctness
- Target: Migrations create `wallet_accounts` + `asset_catalog` with required constraints; seed includes BTC/ETH/USDT rows for approved non-prod networks, and ETH/USDT per network share one allocator.
- Metric: Startup validation gate
- Target: Service refuses startup 100% when enabled catalog rows violate mapping, `address_scheme` allow-list, expiry bounds, or chain requirements.
- Metric: Deterministic derivation
- Target: For fixed `(keyset_id, derivation_path_template, derivation_index)`, adapter returns identical address across repeated calls.
- Metric: Concurrency safety
- Target: Under >=200 parallel allocation/create operations (including mixed ETH+USDT), duplicate `(wallet_account_id, derivation_index)` and duplicate `(chain, network, address_canonical)` counts are exactly `0`.
