---
doc: 01_requirements
spec_date: 2026-02-20
slug: detected-threshold-configurable
mode: Quick
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
  - 2026-02-20-chainobserver-solid-refactor
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: null
  tasks: 03_tasks.md
  test_plan: null
---

# Requirements

## Glossary (optional)

- `detected`
- Payment has reached a configurable percentage of expected amount but not yet confirmed threshold.

- `confirmed`
- Payment has reached or exceeded confirmed threshold.

## Out-of-scope behaviors

- OOS1: Introduce new status values (for example `underpaid`) in this iteration.
- OOS2: Per-chain or per-asset threshold overrides.

## Functional requirements

### FR-001 - EVM chains support `detected`

- Description: ETH native and USDT (ERC-20 in devtest) observations must set `Detected=true` when total observed amount is at or above detected threshold and below confirmed threshold.
- Acceptance criteria:
  - [ ] AC1: For an EVM payment request where received amount is >= detected threshold and < confirmed threshold, reconcile moves request from `pending` to `detected`.
  - [ ] AC2: For same request when received amount later becomes >= confirmed threshold, reconcile moves request from `detected` to `confirmed`.
- Notes: Keep existing supported/unsupported behavior unchanged.

### FR-002 - BTC and EVM use same threshold policy

- Description: BTC/ETH/USDT observation logic must evaluate detected/confirmed against shared configured thresholds.
- Acceptance criteria:
  - [ ] AC1: BTC mempool+confirmed aggregation uses configured thresholds rather than hardcoded 100%.
  - [ ] AC2: ETH/USDT observation uses configured thresholds and emits consistent metadata with actual observed amount.
- Notes: Thresholds apply globally in this iteration.

### FR-003 - Operator configurable thresholds with validation

- Description: System must expose detected and confirmed thresholds through environment config and fail fast on invalid values.
- Acceptance criteria:
  - [ ] AC1: Add envs for detected and confirmed threshold in bps and load them into runtime config.
  - [ ] AC2: Startup fails when detected threshold is < 1 or > confirmed threshold.
  - [ ] AC3: Startup fails when confirmed threshold is > 10000 bps.
- Notes: Defaults must preserve current behavior (`detected=10000`, `confirmed=10000`).

### FR-004 - Documentation and deploy defaults updated

- Description: Local/dev deployment surface must include the new envs and clear examples.
- Acceptance criteria:
  - [ ] AC1: `deployments/service/docker-compose.yml` includes explicit defaults for new env vars.
  - [ ] AC2: `README.md` documents env purpose, valid ranges, and examples.
- Notes: Do not require DB migration.

## Non-functional requirements

- Performance (NFR-001): Additional threshold checks must be constant-time integer math with no additional outbound RPC calls per observation.
- Availability/Reliability (NFR-002): Invalid threshold configuration must terminate startup deterministically before serving traffic.
- Security/Privacy (NFR-003): No secrets added beyond existing envs; thresholds are non-sensitive numeric settings.
- Compliance (NFR-004): Not applicable for this iteration.
- Observability (NFR-005): Observation metadata must continue to include observed amount for reconciliation traceability.
- Maintainability (NFR-006): Observer threshold logic must remain centralized/reused to avoid per-chain drift.

## Dependencies and integrations

- External systems: Existing Esplora and EVM JSON-RPC devtest endpoints only.
- Internal services: Reconciler use case consumes `chainobserver` output; no port contract changes expected.
