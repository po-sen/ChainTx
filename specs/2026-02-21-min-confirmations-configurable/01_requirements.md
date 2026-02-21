---
doc: 01_requirements
spec_date: 2026-02-21
slug: min-confirmations-configurable
mode: Quick
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
  - 2026-02-20-chainobserver-solid-refactor
  - 2026-02-20-detected-threshold-configurable
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: null
  tasks: 03_tasks.md
  test_plan: null
---

# Requirements

## Glossary (optional)

- minimum confirmations
- The minimum on-chain depth required before funds count toward `confirmed` evaluation.

- latest state
- EVM balance observed at `latest` block tag (not delayed by additional confirmations).

## Out-of-scope behaviors

- OOS1: per-asset/per-network override table.
- OOS2: persistence model changes for tracking per-transaction confirmations.

## Functional requirements

### FR-001 - Configurable minimum confirmations for BTC and EVM

- Description: system must accept separate env settings for BTC and EVM minimum confirmations and validate them at startup.
- Acceptance criteria:
  - [x] AC1: add `PAYMENT_REQUEST_RECONCILER_BTC_MIN_CONFIRMATIONS` with default `1`.
  - [x] AC2: add `PAYMENT_REQUEST_RECONCILER_EVM_MIN_CONFIRMATIONS` with default `1`.
  - [x] AC3: startup fails when either value is not a positive integer.
- Notes: values are global per chain family in this iteration.

### FR-002 - BTC confirmed logic respects configured minimum confirmations

- Description: BTC observer must compute `confirmed` amount using outputs with confirmations >= configured minimum.
- Acceptance criteria:
  - [x] AC1: when BTC minimum confirmations > current tx confirmations, request does not transition to `confirmed`.
  - [x] AC2: once tx confirmations reach configured minimum and amount threshold is met, request can transition to `confirmed`.
  - [x] AC3: `detected` still uses earlier signal (confirmed+pending visibility) so request can be `detected` before `confirmed`.
- Notes: keep existing unsupported behavior when BTC endpoint is absent.

### FR-003 - EVM confirmed logic respects configured minimum confirmations

- Description: EVM observer must evaluate confirmed amount at a block tag delayed by configured minimum confirmations.
- Acceptance criteria:
  - [x] AC1: when EVM minimum confirmations > 1, `confirmed` uses balance at block `latest-(min_confirmations-1)`.
  - [x] AC2: ETH and USDT both follow the same EVM minimum-confirmation rule.
  - [x] AC3: `detected` remains driven by `latest` state so request can be `detected` before `confirmed`.
- Notes: applies to EVM native and ERC-20 flows.

### FR-004 - Deployment/docs surface exposes new confirmation settings

- Description: local deployment and docs must include the new env settings and examples.
- Acceptance criteria:
  - [x] AC1: `deployments/service/docker-compose.yml` defines defaults for new envs.
  - [x] AC2: `Makefile` passes through matching `SERVICE_*` variables.
  - [x] AC3: `README.md` describes meaning, defaults, and valid values.
- Notes: no migration required.

## Non-functional requirements

- Performance (NFR-001): additional confirmation-depth calculations must not add more than one extra remote call per EVM observation and should remain bounded for BTC observation.
- Availability/Reliability (NFR-002): invalid confirmation config must fail startup deterministically with explicit config error code.
- Security/Privacy (NFR-003): no new secrets introduced; only numeric env settings.
- Compliance (NFR-004): not applicable in this iteration.
- Observability (NFR-005): observation metadata should include effective minimum confirmations used for decision.
- Maintainability (NFR-006): confirmation-depth logic should be encapsulated in observer adapters without leaking chain-specific details into use-case layer.

## Dependencies and integrations

- External systems: existing BTC Esplora-compatible endpoint and EVM JSON-RPC endpoint.
- Internal services: reconciler use case and chain observer adapter integration remains unchanged at port level.
