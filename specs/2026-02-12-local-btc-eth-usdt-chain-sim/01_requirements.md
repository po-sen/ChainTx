---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- Rail stack: A dedicated Docker Compose stack for one integration rail (BTC, ETH, or USDT).
- Service stack: Docker Compose stack for ChainTx application + PostgreSQL.
- BTC payer wallet: Wallet used to create outbound BTC test payments in local simulation.
- BTC receiver wallet: Wallet whose account-level descriptor/xpub data is exported for receiving-address derivation tests.
- Artifact file: Generated local JSON file containing runtime values (RPC URL, addresses, xpub, contract address) used by smoke scripts.

## Out-of-scope behaviors

- OOS1: Production-grade node security (HSM, TLS mutual auth, HA clusters).
- OOS2: Mainnet deployment or real-fund operations.
- OOS3: Local BTC `testnet` simulation in this first iteration.
- OOS4: Non-Ethereum USDT rails in this spec.

## Functional requirements

### FR-001 - Provide separated compose files per rail

- Description: Repository must include independent compose files for BTC, ETH, and USDT local stacks.
- Acceptance criteria:
  - [ ] AC1: Compose files exist under `deployments/local-chains/` with explicit per-rail naming (`docker-compose.btc.yml`, `docker-compose.eth.yml`, `docker-compose.usdt.yml`).
  - [ ] AC2: Each rail compose can be started and stopped independently without requiring all other rails to run.
  - [ ] AC3: Readiness checks are explicit and deterministic per rail:
    - BTC ready: `getblockchaininfo` succeeds and `initialblockdownload=false`.
    - ETH ready: JSON-RPC `eth_chainId` and `eth_blockNumber` succeed.
    - USDT ready: deploy artifact exists and contract readiness check succeeds (deployment receipt or `balanceOf` probe).
  - [ ] AC4: USDT compose is separate from ETH compose and references ETH RPC via explicit configuration.
  - [ ] AC5: Rail compose files are isolated by default and do not require a shared external docker network.
  - [ ] AC6: Cross-rail dependency contracts use explicit RPC endpoint configuration (for example `ETH_RPC_URL`) instead of compose service DNS coupling.
  - [ ] AC7: Rail extension follows Open/Closed style: adding a new rail (for example `usdt-tron`) is done by adding a new compose file and new `chain-up/down-*` targets, without changing existing BTC/ETH/USDT rail contracts.
- Notes: Separation is required for operational clarity and partial test runs.

### FR-002 - Provide Makefile orchestration for local stacks

- Description: Makefile must expose only startup/shutdown lifecycle commands for rails and local profiles.
- Acceptance criteria:
  - [ ] AC1: Make targets include per-rail `up/down` commands for BTC, ETH, and USDT stacks.
  - [ ] AC2: Make targets include profile commands:
    - `local-up` / `local-down` (default minimal profile: service + BTC)
    - `local-up-all` / `local-down` (optional full profile: service + BTC + ETH + USDT)
  - [ ] AC3: `chain-up-usdt` uses explicit `ETH_RPC_URL` contract and fails fast when ETH RPC is unreachable or chain id is unexpected.
  - [ ] AC4: Commands are idempotent (re-running `up` does not create duplicate critical resources or fatal conflicts).
  - [ ] AC5: Each Make wrapper uses fixed compose project names via `docker compose --project-name`:
    - BTC: `chaintx-local-btc`
    - ETH: `chaintx-local-eth`
    - USDT: `chaintx-local-usdt`
    - Service: `chaintx-local-service`
- Notes: Service lifecycle is standardized on `service-up` / `service-down` targets.

### FR-003 - Start ChainTx service in local integration mode

- Description: ChainTx service must be runnable alongside local chain simulation with documented environment wiring.
- Acceptance criteria:
  - [ ] AC1: Service compose (existing or new local variant) starts PostgreSQL and app with valid defaults for local testing.
  - [ ] AC2: Service startup remains successful with default minimal profile and optional full profile, and `GET /healthz` returns `200`.
  - [ ] AC3: Service startup does not require ETH/USDT stacks in default profile.
  - [ ] AC4: Local environment docs specify how service consumes rail artifacts/endpoints for smoke flows.
- Notes: This spec does not require changing domain logic; startup integration is infrastructure/workflow level.

### FR-004 - Bootstrap BTC payer and receiver descriptor/xpub data automatically

- Description: BTC stack must include automated wallet initialization for payer and receiver flows with stable descriptor-based export.
- Acceptance criteria:
  - [ ] AC1: Bootstrap creates two wallets with stable names (one payer wallet, one receiver wallet).
  - [ ] AC2: Bootstrap funds payer wallet in regtest and mines enough blocks for spendable balance (coinbase maturity satisfied).
  - [ ] AC3: Bootstrap exports receiver account-level key material into artifact output (`receiver_descriptor`, `receiver_xpub`, derivation template).
  - [ ] AC4: Bootstrap is safe to re-run and does not corrupt existing wallet state.
  - [ ] AC5: Receiver export follows descriptor-based BIP84 policy (`wpkh`, external branch `/0/*`), extracted via `listdescriptors`-compatible flow.
  - [ ] AC6: Local BTC simulation network is fixed to `regtest` in this iteration.
  - [ ] AC7: BTC node image is pinned to `bitcoin/bitcoin:29.0` (or explicit equivalent pinned tag) and must support descriptor wallets plus `listdescriptors`.
  - [ ] AC8: Wallet bootstrap explicitly creates descriptor wallets (`createwallet` with `descriptors=true`) and rejects legacy-wallet mode in this feature.
- Notes: Exact key prefix normalization can follow current ChainTx key validation behavior.

### FR-005 - Provide BTC payment smoke capability

- Description: Local workflow must support an executable BTC payment smoke scenario using the payer wallet.
- Acceptance criteria:
  - [ ] AC1: Workflow can generate a receiving address from receiver descriptor/xpub path/index for smoke validation.
  - [ ] AC2: Workflow can send BTC from payer wallet to receiving address and mine blocks to confirm inclusion on regtest.
  - [ ] AC3: Smoke output includes txid, receiving address, and balance delta evidence.
  - [ ] AC4: On rerun with low spendable balance, bootstrap/smoke auto-mines top-up blocks and reports resulting balance.
- Notes: This requirement validates infrastructure readiness, not full production settlement logic.

### FR-006 - Provide local ETH execution environment

- Description: ETH rail stack must provide a local JSON-RPC network with deterministic funded accounts.
- Acceptance criteria:
  - [ ] AC1: ETH stack exposes JSON-RPC endpoint and reports readiness before dependent actions execute.
  - [ ] AC2: At least one funded payer account is available and exported via artifact output for smoke tests.
  - [ ] AC3: Local EVM chain id is fixed and enforced as `31337` across compose config, scripts, and artifacts.
- Notes: Implementation can use Anvil/Hardhat/Ganache as long as behavior is deterministic and scripted.

### FR-007 - Deploy and seed USDT contract via dedicated stack

- Description: USDT rail stack must deploy ERC20-compatible contract to local EVM and mint test balances.
- Acceptance criteria:
  - [ ] AC1: USDT stack performs contract deployment against ETH local RPC and stores deployed contract address as artifact.
  - [ ] AC2: USDT stack mints configurable test balance to payer account used in smoke workflow.
  - [ ] AC3: Contract metadata needed by tests (`contract_address`, `token_decimals`, `chain_id`) is emitted in machine-readable output with fixed values `token_decimals=6` and `chain_id=31337`.
  - [ ] AC4: Re-running deployment path is deterministic (reuse existing deployment artifact or follow documented reset behavior).
  - [ ] AC5: Deployment/mint flow fails fast when chain id mismatches the expected local value (`31337`).
  - [ ] AC6: Deterministic deploy policy is fixed:
    - if `usdt.json` exists and chain fingerprint matches current ETH chain (`chain_id + genesis_block_hash`), skip redeploy and reuse artifact
    - if fingerprint mismatches, fail with explicit remediation (`chain-down-usdt` then `chain-up-usdt`)
- Notes: USDT compose remains a separate operational unit even though it depends on ETH RPC.

### FR-008 - Keep local simulation changes isolated from core application logic

- Description: This work must not introduce business behavior regressions in ChainTx core modules.
- Acceptance criteria:
  - [ ] AC1: Core domain/application packages do not require mandatory logic changes to support local chain startup.
  - [ ] AC2: Most new artifacts are limited to `deployments/`, `scripts/`, `Makefile`, and documentation.
  - [ ] AC3: Existing API behavior (`/healthz`, `/v1/assets`, payment request APIs) remains backward compatible.
- Notes: Small dependency-injection wiring updates are acceptable only if strictly needed for local infra hooks.

### FR-009 - Document operator runbook for local chain simulation

- Description: README (or dedicated local-runbook doc) must document setup, commands, and troubleshooting.
- Acceptance criteria:
  - [ ] AC1: Docs list prerequisites, per-stack commands, fixed local constants (`BTC regtest`, `EVM 31337`, `USDT decimals 6`), and artifact file locations.
  - [ ] AC2: Docs include a minimal startup/shutdown sequence (default profile: service + BTC) and an optional full sequence (service + BTC + ETH + USDT).
  - [ ] AC3: Docs include common failure diagnostics (port conflict, RPC unreachable, wallet init failure, contract deploy failure, stale artifact mismatch).
  - [ ] AC4: Docs include cleanup strategy usage and when to run per-rail `down` commands with data removal options.
  - [ ] AC5: Docs include `.gitignore` rules for generated local artifact/key files.
- Notes: Documentation should prioritize copy-paste commands.

### FR-010 - Define versioned artifact contract and stale-state detection

- Description: Bootstrap and deploy scripts must produce stable artifact schemas and detect stale cross-stack state before smoke actions.
- Acceptance criteria:
  - [ ] AC1: Each artifact (`btc.json`, `eth.json`, `usdt.json`) contains at least `schema_version`, `generated_at`, `network`, `compose_project`, and rail-specific runtime fields.
  - [ ] AC1.1: `compose_project` field must exactly equal the fixed Make wrapper project names (`chaintx-local-btc|eth|usdt|service` as applicable).
  - [ ] AC1.2: ETH and USDT artifacts must include `genesis_block_hash` for chain fingerprint checks.
  - [ ] AC2: Artifacts include `warnings` array (may be empty) for non-fatal conditions.
  - [ ] AC3: `scripts/local-chains/smoke_local_all.sh` detects stale USDT artifact after ETH reset/restart using `chain_id + genesis_block_hash` fingerprint and returns actionable remediation (for example run `chain-down-usdt` then `chain-up-usdt`).
  - [ ] AC4: Artifact consumers fail fast with explicit errors when required fields are missing or malformed.
- Notes: Artifact schema version starts at `1`.

## Non-functional requirements

- Performance (NFR-001): From warm image cache, per-rail stack startup to ready state should complete within 120 seconds; default `local-up` within 180 seconds; optional `local-up-all` within 360 seconds.
- Availability/Reliability (NFR-002): `make local-up` + `scripts/local-chains/smoke_local.sh` + `make local-down` succeed for at least 3 consecutive cycles; `make local-up-all` + `scripts/local-chains/smoke_local_all.sh` + `make local-down` succeeds for at least 1 cycle.
- Security/Privacy (NFR-003): Local simulation uses test keys only; generated artifacts/keys are git-ignored; no production credentials are committed.
- Compliance (NFR-004): Local guards prevent accidental real-network usage by enforcing BTC `regtest` and EVM `chain_id=31337` defaults.
- Observability (NFR-005): Logs and status commands expose readiness probe results, preflight failures, artifact paths, and smoke summaries.
- Maintainability (NFR-006): All orchestration commands are deterministic, documented, and pass repository shell/static checks.

## Dependencies and integrations

- External systems: Docker Engine, Docker Compose plugin, container images for Bitcoin Core (`bitcoin/bitcoin:29.0`, descriptor wallet support required) and local EVM runtime.
- Internal services: Existing ChainTx app + PostgreSQL compose stack.
- Tooling/scripts: Shell scripts (or equivalent) for wallet bootstrap, contract deployment, artifact generation, readiness checks, reset, and smoke validation.
