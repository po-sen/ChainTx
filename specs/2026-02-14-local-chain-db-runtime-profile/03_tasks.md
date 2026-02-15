---
doc: 03_tasks
spec_date: 2026-02-14
slug: local-chain-db-runtime-profile
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-10-wallet-allocation-asset-catalog
  - 2026-02-12-local-btc-eth-usdt-chain-sim
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
- Rationale: This change affects data seed contracts, application derivation contract, startup orchestration, and cross-rail local integration.

## Milestones

- M1: DB tuples for EVM local + sepolia are defined and upsertable.
- M2: Derivation path uses DB chain id and no hardcoded non-mainnet id.
- M3: Service startup applies local DB tuples from artifacts.
- M4: Docs and tests cover local/sepolia coexistence.

## Tasks (ordered)

1. T-001 - Add DB upsert migration/seed for local EVM wallet account and catalog rows

   - Scope: Add/update migration SQL to include `wa_eth_local_001` and `ethereum/local/{ETH,USDT}` rows.
   - Linked requirements: FR-002, FR-003, FR-004
   - Validation:
     - [ ] Run migrations on clean DB.
     - [ ] Verify local/sepolia tuples coexist in `app.asset_catalog`.

2. T-002 - Update local catalog sync script to DB-first tuple updates

   - Scope: Extend script to upsert local rows and refresh local USDT contract from artifact.
   - Linked requirements: FR-001, FR-004, FR-007
   - Validation:
     - [ ] Run service startup twice.
     - [ ] Verify idempotent updates and correct local metadata.

3. T-003 - Extend wallet gateway port input with optional chain id

   - Scope: Add `ChainID *int64` to `DeriveAddressInput` and thread through use case call.
   - Linked requirements: FR-005, NFR-004
   - Validation:
     - [ ] Compile and run unit tests for use case/wallet gateway.
     - [ ] Confirm chain-id mismatch logic still guards invalid returns.

4. T-004 - Update devtest gateway EVM chain-id behavior

   - Scope: Use input chain id for non-mainnet EVM derivation instead of fixed sepolia default.
   - Linked requirements: FR-005, NFR-001, NFR-005
   - Validation:
     - [ ] Add tests for `network=local` (`31337`) and `network=sepolia` (`11155111`).
     - [ ] Keep `mainnet=1` behavior unchanged.

5. T-005 - Keep service env minimal and demote env chain-id override

   - Scope: Remove hard dependency on `PAYMENT_REQUEST_DEVTEST_ETH_SEPOLIA_CHAIN_ID` in local path.
   - Linked requirements: FR-001, FR-005, NFR-003
   - Validation:
     - [ ] Run local flow without setting that env.
     - [ ] Verify chain id comes from DB tuple.

6. T-006 - Update Make orchestration order for DB upsert hook

   - Scope: Ensure BTC/ETH artifacts plus USDT deploy artifact exist before service DB upsert in `local-up-all`.
   - Linked requirements: FR-007, NFR-002
   - Validation:
     - [ ] Run `make local-up-all` from clean state.
     - [ ] Confirm upsert hook runs after service postgres is ready.

7. T-007 - Update README and runbook for local network tuple

   - Scope: Document `network=local` usage for local ETH/USDT and tuple differences.
   - Linked requirements: FR-006, FR-008
   - Validation:
     - [ ] Follow README commands end-to-end.
     - [ ] Confirm local request example returns `chain_id=31337`.

8. T-008 - Add/adjust integration coverage for tuple coexistence and cursor split

   - Scope: Add tests proving local and sepolia rows can allocate independently.
   - Linked requirements: FR-002, FR-003, NFR-005
   - Validation:
     - [ ] Assert distinct wallet accounts and independent `next_index` progression.

9. T-009 - Validate smoke flows for local and sepolia tuples

   - Scope: Ensure smoke scripts cover local ETH/USDT and preserve BTC flow.
   - Linked requirements: FR-006, FR-007
   - Validation:
     - [ ] Run `scripts/local-chains/smoke_local.sh`.
     - [ ] Run full smoke including local EVM tuple checks.

10. T-010 - Final readiness verification and spec status update
    - Scope: Run lint/tests, verify docs and migrations, then keep spec at `READY` for implementation kickoff.
    - Linked requirements: FR-001, FR-008, NFR-001, NFR-002, NFR-005
    - Validation:
      - [ ] `go fmt ./...`
      - [ ] `go vet ./...`
      - [ ] targeted `go test` suites
      - [ ] spec lint pass

## Traceability (optional)

- FR-001 -> T-002, T-005, T-010
- FR-002 -> T-001, T-008
- FR-003 -> T-001, T-008
- FR-004 -> T-001, T-002, T-006
- FR-005 -> T-003, T-004, T-005
- FR-006 -> T-007, T-009
- FR-007 -> T-002, T-006, T-009
- FR-008 -> T-007, T-010
- NFR-001 -> T-004, T-010
- NFR-002 -> T-006, T-010
- NFR-003 -> T-005
- NFR-004 -> T-003
- NFR-005 -> T-004, T-008, T-010
