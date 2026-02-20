---
doc: 01_requirements
spec_date: 2026-02-20
slug: chainobserver-solid-refactor
mode: Quick
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-chain-listener-payment-reconcile
  - 2026-02-20-reconciler-horizontal-scaling
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: null
  tasks: 03_tasks.md
  test_plan: null
---

# Requirements

## Out-of-scope behaviors

- OOS1: no new chain/provider support.
- OOS2: no status model changes.

## Functional requirements

### FR-001 - Chain observer responsibilities must be separated

- Description: adapter code must separate coordinator routing from BTC and EVM implementation details.
- Acceptance criteria:
  - [x] AC1 coordinator entrypoint delegates by chain type without embedding provider protocol details.
  - [x] AC2 BTC observation logic resides in dedicated component/file.
  - [x] AC3 EVM observation logic resides in dedicated component/file.

### FR-002 - RPC transport details must be isolated

- Description: JSON-RPC request/response handling must be extracted from coordinator logic.
- Acceptance criteria:
  - [x] AC1 RPC call encoding/decoding is handled by a focused helper/component.
  - [x] AC2 EVM observer consumes this helper instead of duplicating protocol handling.

### FR-003 - Existing behavior must remain compatible

- Description: BTC/EVM observed amount and confirmed/detected semantics must remain unchanged.
- Acceptance criteria:
  - [x] AC1 BTC confirmed/detected behavior matches pre-refactor tests.
  - [x] AC2 EVM confirmed behavior matches pre-refactor tests.
  - [x] AC3 unsupported chain/network behavior remains unchanged.

## Non-functional requirements

- Maintainability (NFR-001): each file/component has one primary reason to change.
- Testability (NFR-002): chain observer unit tests pass without relaxing assertions.
- Reliability (NFR-003): refactor introduces no regressions in `go test -short ./...`.

## Dependencies and integrations

- Internal services: `ReconcilePaymentRequestsUseCase` via `PaymentChainObserverGateway` port.
- External systems: BTC Esplora-compatible endpoints and EVM JSON-RPC endpoints.
