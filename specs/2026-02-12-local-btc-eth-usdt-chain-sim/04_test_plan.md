---
doc: 04_test_plan
spec_date: 2026-02-12
slug: local-btc-eth-usdt-chain-sim
mode: Full
status: DONE
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

# Test Plan

## Scope

- Covered:
  - Rail-level compose separation and lifecycle commands (BTC/ETH/USDT).
  - Default profile (`service + BTC`) and optional full profile (`service + BTC + ETH + USDT deploy`).
  - BTC descriptor/xpub bootstrap and payment smoke flow.
  - ETH local RPC readiness and chain-id guard (`31337`).
  - USDT local ERC20 deployment/mint with `token_decimals=6`.
  - Artifact schema/version checks, stale-artifact detection, and reset behavior.
  - Documentation-driven reproducibility.
- Not covered:
  - Mainnet behavior or production wallet providers.
  - Blockchain confirmation webhook/state-machine business logic.
  - Non-Ethereum USDT rails.

## Tests

### Unit

- TC-001: Make target contract validation

  - Linked requirements: FR-002, FR-009, NFR-006
  - Steps: Parse `Makefile` to verify required targets exist (`local-up`, `local-up-all`, `local-down`, and per-rail `chain-up/-down`) and each wrapper passes fixed `--project-name`.
  - Expected: Target names/profile semantics match runbook and project names map to `chaintx-local-btc|eth|usdt|service`.

- TC-002: BTC bootstrap idempotency and top-up behavior

  - Linked requirements: FR-004, FR-005, NFR-002, NFR-005
  - Steps: Run BTC bootstrap twice; simulate low spendable balance; rerun bootstrap.
  - Expected: Rerun is safe, wallets are reused, and script auto-mines/top-ups to restore spendable balance.

- TC-003: USDT deploy chain-id guard

  - Linked requirements: FR-002, FR-007, NFR-004
  - Steps: Invoke deploy script against mismatched chain id.
  - Expected: Script fails before deployment/mint with explicit chain-id mismatch error.

- TC-004: Artifact schema validation

  - Linked requirements: FR-010, NFR-005
  - Steps: Validate generated artifacts against required fields (`schema_version`, `generated_at`, `network`, `compose_project`, `warnings`, plus rail-specific fields including `genesis_block_hash` for ETH/USDT).
  - Expected: Artifacts pass schema checks, `compose_project` matches wrapper project name, and fixed constants are present where applicable.

- TC-005: BTC descriptor extraction correctness
  - Linked requirements: FR-004, NFR-006
  - Steps: Verify receiver export uses descriptor-compatible BIP84 external branch and derivation template format.
  - Expected: Artifact contains valid descriptor/xpub/template values for regtest flow.

### Integration

- TC-101: BTC stack independent lifecycle (regtest)

  - Linked requirements: FR-001, FR-004, FR-005, NFR-001
  - Steps: `make chain-up-btc`, verify readiness probe, run bootstrap, then `make chain-down-btc`.
  - Expected: BTC stack runs independently and emits valid `btc.json`.

- TC-102: ETH stack independent lifecycle (chain id fixed)

  - Linked requirements: FR-001, FR-006, NFR-001, NFR-004
  - Steps: `make chain-up-eth`, query `eth_chainId` and `eth_blockNumber`, then `make chain-down-eth`.
  - Expected: ETH RPC is reachable and `eth_chainId == 0x7a69` (`31337`).

- TC-103: USDT deploy utility lifecycle on ETH chain

  - Linked requirements: FR-001, FR-007, NFR-001, NFR-004
  - Steps: Run `make chain-up-eth`; verify artifact/token metadata and on-chain contract code through ETH RPC.
  - Expected: USDT deploy succeeds only on expected ETH chain and outputs `token_decimals=6`, `chain_id=31337`.

- TC-104: Service startup with default profile

  - Linked requirements: FR-002, FR-003, FR-008, NFR-002
  - Steps: `make local-up`, call `GET /healthz`, inspect status output, then `make local-down`.
  - Expected: Service is healthy and default profile does not require ETH/USDT.

- TC-105: Default smoke flow

  - Linked requirements: FR-002, FR-005, FR-010, NFR-005
  - Steps: Run `make local-up` then `scripts/local-chains/smoke_local.sh`.
  - Expected: Smoke verifies service + BTC path and outputs pass/fail summary with evidence.

- TC-106: Full smoke flow

  - Linked requirements: FR-002, FR-006, FR-007, FR-010, NFR-005
  - Steps: Run `make local-up-all` then `scripts/local-chains/smoke_local_all.sh`.
  - Expected: Smoke verifies BTC/ETH/USDT flows, applies USDT reuse policy by chain fingerprint, and outputs full summary including contract and transfer checks.

- TC-107: Reset command behavior
  - Linked requirements: FR-002, FR-009, FR-010, NFR-002
  - Steps: Generate state/artifacts, run per-rail cleanup using `docker compose ... down` (with `-v` when needed), then rerun corresponding startup target.
  - Expected: Targeted state is removed and recreated successfully with fresh valid artifacts.

### E2E (if applicable)

- TC-201: Default profile repeatability

  - Linked requirements: FR-002, FR-003, FR-005, FR-010, NFR-001, NFR-002
  - Steps: Execute 3 cycles of `make local-up` -> `scripts/local-chains/smoke_local.sh` -> `make local-down`.
  - Expected: All 3 cycles pass without manual cleanup.

- TC-202: Optional full profile cycle

  - Linked requirements: FR-002, FR-003, FR-006, FR-007, FR-010, NFR-001, NFR-002
  - Steps: Execute 1 cycle of `make local-up-all` -> `scripts/local-chains/smoke_local_all.sh` -> `make local-down`.
  - Expected: Full profile passes one complete cycle with deterministic artifacts.

- TC-203: Stale artifact detection after ETH chain reset
  - Linked requirements: FR-010, FR-007, FR-002, NFR-002, NFR-005
  - Steps:
    - Run full profile once.
    - Execute `make chain-down-eth && make chain-up-eth` only.
    - Keep stale `eth.json` USDT metadata without redeploy.
    - Run `scripts/local-chains/smoke_local_all.sh`.
  - Expected: Smoke fails fast when `chain_id + genesis_block_hash` fingerprint mismatches and returns actionable remediation (for example `chain-down-eth` then `chain-up-eth`).

## Edge cases and failure modes

- Case: BTC RPC port conflict on host.
- Expected behavior: `chain-up-btc` fails fast with clear port binding error and remediation hint.

- Case: ETH RPC is unavailable while `chain-up-eth` executes deploy phase.
- Expected behavior: USDT deploy step exits quickly with explicit dependency failure and remediation (`make chain-down-eth && make chain-up-eth`).

- Case: ETH chain reset causes stale USDT artifact.
- Expected behavior: Full smoke detects mismatch and prints precise reset workflow.

- Case: BTC bootstrap rerun with insufficient spendable balance or wallet lock state.
- Expected behavior: Bootstrap recovers via unlock/retry/top-up mining path, or exits with actionable error if recovery is impossible.

## NFR verification

- Performance:
  - Measure per-rail readiness latency, default profile startup time, and full-profile startup time against NFR-001 thresholds.
- Reliability:
  - Validate repeatability with TC-201/TC-202 and stale-state scenario TC-203.
- Security:
  - Verify generated artifacts/keys are git-ignored and no production credentials are used.

## Execution results

- Execution date: 2026-02-12.
- Passed: TC-101, TC-102, TC-103, TC-104, TC-105, TC-106, TC-107 (single-cycle execution).
- Passed: unit-level config/syntax checks for TC-001, TC-004.
- Passed: repository regression checks (`make lint`, `go test ./...`).
- Pending future reliability burn-in: TC-201 and TC-202 multi-cycle repetition target (recommended follow-up in CI or long-run local validation).
