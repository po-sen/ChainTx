---
doc: 03_tasks
spec_date: 2026-02-12
slug: local-btc-eth-usdt-chain-sim
mode: Full
status: READY
owners:
  - posen
depends_on: []
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
- Rationale: This scope includes multi-stack external integrations (Bitcoin Core, local EVM, ERC20 deployment), profile orchestration, cross-compose networking contracts, and non-trivial failure modes.
- Upstream dependencies (`depends_on`): []
- Dependency gate before `READY`: every dependency is folder-wide `status: DONE`
- If `02_design.md` is skipped (Quick mode): Not applicable; Full mode required.
- If `04_test_plan.md` is skipped: Not applicable; Full mode requires test plan.

## Milestones

- M1: Local constants and compose/network contracts are fixed (`BTC regtest`, `EVM 31337`, `USDT decimals 6`).
- M2: BTC stack (descriptor bootstrap + artifacts) is stable and rerunnable.
- M3: ETH/USDT stacks and profile-based Make commands are stable.
- M4: Smoke, reset, and runbook flows are validated and reproducible.

## Tasks (ordered)

1. T-001 - Scaffold local chain layout and external network contract

   - Scope: Create `deployments/local-chains/` structure, rail-specific compose files, shared external network, and per-rail readiness checks.
   - Output: Compose files pass validation and consistently attach to `chaintx-local-net`.
   - Linked requirements: FR-001, FR-002, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): `docker compose -f deployments/local-chains/docker-compose.btc.yml config` and equivalent for ETH/USDT/service-local.
     - [ ] Expected result: Compose files parse successfully and declare shared external network contract.
     - [ ] Logs/metrics to check (if applicable): N/A

2. T-002 - Implement BTC stack bootstrap with descriptor-based receiver export

   - Scope: Add pinned BTC node (`bitcoin/bitcoin:29.0`) + bootstrap script to create/reuse descriptor wallets (`descriptors=true`), ensure spendable balance, and export descriptor/xpub artifacts.
   - Output: `btc.json` containing descriptor + xpub + derivation template.
   - Linked requirements: FR-004, FR-005, FR-010, NFR-002, NFR-005
   - Validation:
     - [ ] How to verify (manual steps or command): `make chain-up-btc` then run bootstrap twice.
     - [ ] Expected result: Wallets are stable across reruns, payer remains spendable, and artifact schema fields are complete.
     - [ ] Logs/metrics to check (if applicable): bootstrap logs include block height, spendable balance, and artifact path.

3. T-003 - Implement ETH stack with fixed chain-id guard and artifacts

   - Scope: Add local EVM compose + artifact exporter with fixed `chain_id=31337` contract.
   - Output: `eth.json` with deterministic funded account metadata.
   - Linked requirements: FR-006, FR-010, NFR-001, NFR-004, NFR-005
   - Validation:
     - [ ] How to verify (manual steps or command): `make chain-up-eth` then query `eth_chainId` and `eth_getBalance`.
     - [ ] Expected result: RPC is healthy and `eth_chainId` equals `0x7a69` (`31337`).
     - [ ] Logs/metrics to check (if applicable): ETH logs confirm startup and artifact export completion.

4. T-004 - Implement dedicated USDT deploy/mint stack

   - Scope: Add USDT deployer that preflight-checks ETH RPC/chain id, enforces deterministic reuse policy by chain fingerprint, deploys ERC20 (`decimals=6`) only when needed, mints test balance, and writes artifact.
   - Output: `usdt.json` with contract metadata and mint evidence.
   - Linked requirements: FR-001, FR-007, FR-010, NFR-001, NFR-004, NFR-005
   - Validation:
     - [ ] How to verify (manual steps or command): `make chain-up-usdt` after ETH stack is ready.
     - [ ] Expected result: Deploy/mint succeeds on chain `31337`; mismatch chain id path fails fast with clear error.
     - [ ] Logs/metrics to check (if applicable): deployment tx hash, contract address, token decimals, and minted amount logged.

5. T-005 - Implement profile-aware Make lifecycle commands

   - Scope: Extend Makefile with per-rail lifecycle, fixed `--project-name` wrappers, `local-up`/`local-up-all`, status commands, and ETH preflight gates.
   - Output: Deterministic command surface for minimal and full profiles.
   - Linked requirements: FR-002, FR-003, FR-006, FR-007, NFR-002, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): run `make local-up`, `make local-status`, `make local-down`, then `make local-up-all`, `make local-status`, `make local-down`.
     - [ ] Expected result: Default profile starts only service+BTC; full profile starts all rails; status output is readable and accurate.
     - [ ] Logs/metrics to check (if applicable): command output shows ordered steps and preflight results.

6. T-006 - Implement reset commands and state cleanup policy

   - Scope: Add `local-reset-btc/eth/usdt/all` for deterministic cleanup of volumes/state/artifacts.
   - Output: Reliable reset workflow with targeted and global cleanup modes.
   - Linked requirements: FR-002, FR-007, FR-009, FR-010, NFR-002, NFR-003
   - Validation:
     - [ ] How to verify (manual steps or command): create state, run each reset target, then rerun corresponding `chain-up-*`.
     - [ ] Expected result: Cleared rail restarts successfully with regenerated valid artifacts.
     - [ ] Logs/metrics to check (if applicable): reset output lists removed resources and artifact files.

7. T-007 - Implement smoke scripts for default and full profiles

   - Scope: Build `local-smoke` (service+BTC) and `local-smoke-all` (service+BTC+ETH+USDT), including stale-artifact detection.
   - Output: Machine-readable pass/fail summaries with actionable remediation on failure.
   - Linked requirements: FR-005, FR-006, FR-007, FR-010, NFR-002, NFR-005
   - Validation:
     - [ ] How to verify (manual steps or command): run smoke commands after corresponding profile startup.
     - [ ] Expected result: Default smoke validates BTC/service; full smoke validates BTC/ETH/USDT and detects stale artifacts with clear reset instructions.
     - [ ] Logs/metrics to check (if applicable): summary includes txids, contract address, and failing step diagnostics.

8. T-008 - Integrate service startup compatibility checks

   - Scope: Ensure service compose and environment wiring remain compatible with local stack profiles without requiring core business logic changes.
   - Output: Service starts cleanly in both profiles and remains API-compatible.
   - Linked requirements: FR-003, FR-008, NFR-001, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): `make local-up` then `curl -i http://localhost:8080/healthz`; repeat with `make local-up-all`.
     - [ ] Expected result: Health endpoint returns `200` and no startup regressions in existing APIs.
     - [ ] Logs/metrics to check (if applicable): app logs show successful startup sequence.

9. T-009 - Document runbook, constants, and gitignore policy

   - Scope: Update docs with fixed local constants, profile usage, reset playbook, troubleshooting, and artifact gitignore rules.
   - Output: Copy-paste runbook for contributors with minimal ambiguity.
   - Linked requirements: FR-009, FR-002, FR-010, NFR-003, NFR-006
   - Validation:
     - [ ] How to verify (manual steps or command): follow docs from clean environment to complete default and full smoke once.
     - [ ] Expected result: Contributor can run workflow without tribal knowledge.
     - [ ] Logs/metrics to check (if applicable): N/A

10. T-010 - Execute reliability cycles and collect evidence

- Scope: Run repeated profile cycles and edge-case checks (stale artifacts, low BTC balance) to prove readiness.
- Output: Validation evidence for NFR targets and known failure remediations.
- Linked requirements: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-010, NFR-001, NFR-002, NFR-003, NFR-004, NFR-005, NFR-006
- Validation:
  - [ ] How to verify (manual steps or command): run 3 cycles of `local-up` + `local-smoke` + `local-down`; run 1 cycle of `local-up-all` + `local-smoke-all` + `local-down`; run stale-artifact and low-balance BTC scenarios.
  - [ ] Expected result: Default profile is stable across repeated cycles, full profile succeeds, and failure scenarios return actionable remediation.
  - [ ] Logs/metrics to check (if applicable): cycle timing, failure codes, and remediation messages captured.

## Traceability (optional)

- FR-001 -> T-001, T-004, T-010
- FR-002 -> T-001, T-005, T-006, T-009, T-010
- FR-003 -> T-005, T-008, T-010
- FR-004 -> T-002, T-010
- FR-005 -> T-002, T-007, T-010
- FR-006 -> T-003, T-005, T-007, T-010
- FR-007 -> T-004, T-005, T-006, T-007, T-010
- FR-008 -> T-008, T-010
- FR-009 -> T-006, T-009, T-010
- FR-010 -> T-002, T-003, T-004, T-007, T-009, T-010
- NFR-001 -> T-003, T-004, T-008, T-010
- NFR-002 -> T-002, T-005, T-006, T-007, T-010
- NFR-003 -> T-006, T-009, T-010
- NFR-004 -> T-003, T-004, T-010
- NFR-005 -> T-002, T-003, T-004, T-007, T-010
- NFR-006 -> T-001, T-005, T-008, T-009, T-010

## Rollout and rollback

- Feature flag: Not required; this is local-only tooling and infrastructure.
- Migration sequencing: No application DB migration dependency in this spec.
- Rollback steps:
  - Remove/disable new local compose and Make targets.
  - Keep existing `deployments/docker-compose.yml` and current dev workflow unchanged.
  - Remove generated artifacts/volumes/networks via reset targets.
