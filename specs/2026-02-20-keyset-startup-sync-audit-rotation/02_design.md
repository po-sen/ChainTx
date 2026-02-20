---
doc: 02_design
spec_date: 2026-02-20
slug: keyset-startup-sync-audit-rotation
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-20-xpub-index0-address-verification
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Technical Design

## High-level approach

- Summary: Move keyset safety and wallet-account sync into app startup adapter flow, add legacy-secret hash matching, and persist sync audit events.
- Key decisions:
  - Keep active secret env unchanged and add optional legacy secret JSON env.
  - Keep hash algorithm fixed at `hmac-sha256`.
  - Keep legacy keyset JSON formats parseable for key material map, but startup preflight requires nested entries with expected index-0 address.

## System context

- Components:
  - `internal/infrastructure/config`: parse active/legacy secrets and preflight entries.
  - `internal/application/use_cases/initialize_persistence_use_case.go`: orchestrate new sync step.
  - `internal/adapters/outbound/persistence/postgresql/bootstrap`: implement preflight + wallet sync + audit insert.
  - `cmd/server/main.go`: unchanged failure behavior; startup exits on use-case error.
- Interfaces:
  - Extend `PersistenceBootstrapGateway` with `SyncWalletAllocationState(ctx)`.

## Key flows

- Flow 1 (startup success):
  - readiness check
  - migrations
  - devtest keyset preflight
  - wallet-account hash sync (reuse/reactivate/rotate)
  - sync audit event insert
  - catalog integrity validation
  - server start
- Flow 2 (startup failure):
  - any config/preflight/sql/hash error during sync returns app error -> startup exits before server bind.

## Data model

- Entities:
  - existing `app.wallet_accounts` and `app.asset_catalog`
  - new `app.wallet_account_sync_events`
- Schema changes or migrations:
  - add migration `000004_wallet_account_sync_events` with append-only audit table and indexes.
- Consistency and idempotency:
  - For each `(chain,network,keyset_id)` keep one active row.
  - Repeated sync with unchanged state records `reused` action but does not change `wallet_account_id`.

## API or contracts

- Startup config envs:
  - required: `PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET`
  - optional: `PAYMENT_REQUEST_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON` (example: `["old-secret-a","old-secret-b"]`)
- Keyset config contract for preflight:
  - nested `chain -> network -> {keyset_id, extended_public_key, expected_index0_address}`

## Backward compatibility (optional)

- API compatibility: no HTTP contract change.
- Data migration compatibility: existing rows with null hash are upgraded on first successful sync (`match_source=unhashed`).

## Failure modes and resiliency

- Retries/timeouts: startup sync uses existing startup context timeout boundaries.
- Degradation strategy: no degraded mode; startup must fail closed on preflight/sync error.

## Observability

- Logs:
  - startup preflight pass/fail with chain/network/keyset.
  - sync action logs with `action`, `wallet_account_id`, hash prefix.
- Metrics:
  - no new metrics in this scope.
- Alerts:
  - operators detect failure via startup exit and logs.

## Security

- Secrets: only active/legacy secrets in env; never persisted.
- Hashing: use HMAC-SHA256 and persist only digest.
- Abuse cases: malformed keyset config or wrong expected address is blocked at startup.

## Alternatives considered

- Option A: keep script-only preflight and sync.
- Option B: move logic into app startup adapter (chosen).
- Why chosen: guarantees deployment-path independent safety and auditable in-DB history.

## Risks

- Risk: stricter startup checks can break legacy flat keyset config deployments.
- Mitigation: explicit startup error message and updated README/env examples with nested format.
