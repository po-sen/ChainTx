---
doc: 03_tasks
spec_date: 2026-02-10
slug: wallet-allocation-asset-catalog
mode: Full
status: DONE
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

# Task Plan

## Mode decision

- Selected mode: Full
- Rationale: This feature introduces new persistent schema/migrations, startup configuration validation, wallet integration boundaries, and concurrency-critical transactional behavior.
- Upstream dependencies (`depends_on`):
  - 2026-02-09-multi-asset-payment-request-api
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`.
- If `02_design.md` is skipped (Quick mode): Not applicable; Full mode is required.
- If `04_test_plan.md` is skipped: Not applicable; Full mode requires test plan.

## Milestones

- M1: Schema + seed + startup validation foundations are in place.
- M2: Dev/test wallet allocation adapter and transactional cursor consumption are implemented.
- M3: Smoke/concurrency verification and readiness evidence are complete.

## Tasks (ordered)

1. T-001 - Add `wallet_accounts` and `asset_catalog` migrations with seed data

   - Scope: Create schema, constraints, and idempotent seed records for BTC/ETH/USDT across approved non-prod networks.
   - Output: Migration files + seed logic with ETH/USDT shared `wallet_account_id` per EVM network.
   - Linked requirements: FR-001, FR-007, NFR-004
   - Validation:
     - [ ] How to verify (manual steps or command): Run migrations twice and inspect rows by SQL query.
     - [ ] Expected result: Tables/constraints exist, seed is idempotent, and shared allocator mapping is present.
     - [ ] Expected result: `wallet_accounts` enforces `UNIQUE(chain, network, keyset_id)`; duplicate natural-key insert fails deterministically.
     - [ ] Expected result: Token/native metadata constraints are enforced for `asset_catalog` (token rows require token fields; native rows keep token fields null).
     - [ ] Expected result: Seeded `address_scheme` values follow chain allow-list (`bip84_p2wpkh` for Bitcoin, `evm_bip44` for EVM).
     - [ ] Logs/metrics to check (if applicable): Migration logs show successful apply/rerun.

2. T-002 - Implement startup catalog-wallet validation gate

   - Scope: Add bootstrap-time validation for enabled asset rows, wallet activity, chain/network compatibility, expiry bounds, token/native metadata invariants, centralized address-scheme allow-list rules, keyset depth metadata checks, and derivation-path suffix constraints.
   - Output: Validation module wired into startup path with structured error reporting.
   - Linked requirements: FR-002, FR-001, NFR-002, NFR-005
   - Validation:
     - [ ] How to verify (manual steps or command): Start service with valid seed, then with intentionally invalid seed fixtures.
     - [ ] Expected result: Valid config starts successfully; invalid config fails before serving traffic with deterministic error codes.
     - [ ] Expected result: Policy mismatches (depth metadata mismatch, non-`0/{index}` suffix, allow-list mismatch, token/native invariant violations) fail startup with `invalid_configuration`.
     - [ ] Expected result: Key parse/normalization incompatibility fails startup with `invalid_key_material_format`.
     - [ ] Logs/metrics to check (if applicable): Validation failure counters and reason-coded logs.

3. T-003 - Implement wallet allocation adapters (`devtest` + `prod` placeholder)

   - Scope: Define gateway contracts with derivation-ready input, implement deterministic BTC/EVM derivation for dev/test, add `prod` placeholder wiring, and validate runtime mode config.
   - Output: Working adapter package with typed errors and mode-based DI registration.
   - Linked requirements: FR-003, FR-004, FR-007, NFR-003, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): Run adapter unit tests with fixed vectors and invalid config cases.
     - [ ] Expected result: Deterministic outputs for known fixtures; unsupported/misconfigured cases return typed errors.
     - [ ] Expected result: EVM dev/test key parsing accepts serialized BIP32 `xpub`/`tpub`, normalizes to one internal representation, and rejects `ypub`/`zpub`/`vpub`.
     - [ ] Expected result: BTC dev/test key parsing accepts `tpub`/`vpub` and performs startup normalization to library-supported form without changing derivation payload.
     - [ ] Expected result: Derivation-path suffix validation rejects hardened segments (`'`/`h`) for xpub-based derivation.
     - [ ] Expected result: When normalization cannot be performed safely by runtime/library, adapter init fails closed with `invalid_key_material_format`.
     - [ ] Expected result: Mainnet requests in dev/test mode return `mainnet_allocation_blocked` unless override is explicitly enabled.
     - [ ] Logs/metrics to check (if applicable): Startup logs include selected mode and guardrail state.

4. T-004 - Wire transactional cursor consumption and application canonicalization in create flow

   - Scope: Integrate `FOR UPDATE` cursor lock, derive raw address, canonicalize in application layer, persist payment request, bump `next_index`, and rollback behavior into one transaction.
   - Output: Create flow guarantees atomic allocation and persistence with clear boundary responsibilities.
   - Linked requirements: FR-005, FR-004, FR-003, NFR-001, NFR-002
   - Validation:
     - [ ] How to verify (manual steps or command): Run integration tests that inject failures before commit and inspect DB state.
     - [ ] Expected result: On success, exactly one index is consumed; on failure, no index drift and no partial request row.
     - [ ] Expected result: EVM canonical lowercase storage and response EIP-55 formatting are both enforced by application layer.
     - [ ] Expected result: `payment_requests.address_canonical` stores canonical value, while API response uses `payment_instructions.address` formatted for clients.
     - [ ] Logs/metrics to check (if applicable): Allocation latency histogram and failure reason counters.

5. T-005 - Add smoke and concurrency test suite for allocator correctness

   - Scope: Add single-create smoke tests for BTC/ETH/USDT and parallel tests (same tuple + mixed ETH/USDT shared allocator).
   - Output: Automated tests proving zero duplicate cursor/address outcomes under load.
   - Linked requirements: FR-006, FR-005, NFR-002, NFR-005
   - Validation:
     - [ ] How to verify (manual steps or command): Execute targeted integration suite with >=200 concurrent operations.
     - [ ] Expected result: Zero duplicates for `(wallet_account_id, derivation_index)` and `(chain, network, address_canonical)`; `next_index` matches success count.
     - [ ] Logs/metrics to check (if applicable): Concurrency test logs include allocation totals and collision assertions.

6. T-006 - Finalize operational docs/config and readiness evidence

   - Scope: Document adapter env/config contract, mainnet guard semantics, key material format requirements, and record validation evidence for readiness review.
   - Output: Updated runbook/config docs + test evidence summary.
   - Linked requirements: FR-003, FR-004, FR-007, NFR-003, NFR-004, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): Review docs and run lint/quality/test commands.
     - [ ] Expected result: Configuration is unambiguous, quality gates pass, and evidence links are complete.
     - [ ] Logs/metrics to check (if applicable): N/A.

## Traceability (optional)

- FR-001 -> T-001, T-002
- FR-002 -> T-002
- FR-003 -> T-003, T-004, T-006
- FR-004 -> T-003, T-004, T-006
- FR-005 -> T-004, T-005
- FR-006 -> T-005
- FR-007 -> T-001, T-003, T-006
- NFR-001 -> T-004
- NFR-002 -> T-002, T-004, T-005
- NFR-003 -> T-003, T-006
- NFR-004 -> T-001, T-006
- NFR-005 -> T-002, T-005
- NFR-006 -> T-003, T-006

## Rollout and rollback

- Feature flag:
  - `PAYMENT_REQUEST_ALLOCATION_MODE=devtest|prod`
  - `PAYMENT_REQUEST_DEVTEST_ALLOW_MAINNET=false` (default)
- Migration sequencing:
  - Apply schema + seed migration.
  - Deploy application with startup validation enabled.
  - Enable create flow in non-prod and run smoke/concurrency tests.
- Rollback steps:
  - Switch traffic away from new build.
  - Keep consumed indices and created payment requests for auditability (no index reuse).
  - Revert application release while keeping additive schema in place.

## Readiness checklist (for status READY)

- [ ] Open questions in `00_problem.md` are resolved or accepted with tracked assumptions.
- [ ] Adapter mode/config contract is documented and reviewed by wallet operations.
- [ ] Smoke and concurrency test evidence is attached.
- [ ] `SPEC_DIR="specs/2026-02-10-wallet-allocation-asset-catalog" bash /Users/posen/.codex/skills/spec-driven-development/scripts/spec-lint.sh` passes.
- [ ] All produced docs remain frontmatter-consistent with `mode: Full` and identical `depends_on`.
