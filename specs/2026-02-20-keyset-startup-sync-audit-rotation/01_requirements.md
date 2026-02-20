---
doc: 01_requirements
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

# Requirements

## Glossary (optional)

- Active HMAC secret: `PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET`, used to write latest hash.
- Legacy HMAC secrets: optional previous secrets provided by `PAYMENT_REQUEST_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON`.
- Keyset preflight entry: nested keyset config entry including `expected_index0_address`.
- Sync action: one of `reused`, `reactivated`, `rotated`.

## Out-of-scope behaviors

- OOS1: Public API for querying sync audit events.
- OOS2: Automatic external secret-manager integration.

## Functional requirements

### FR-001 - Parse active and legacy HMAC secret config

- Description: System must parse one active secret and optional legacy secret list for key-material hash matching.
- Acceptance criteria:
  - [x] AC1 `PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET` remains required in devtest mode.
  - [x] AC2 `PAYMENT_REQUEST_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON` is optional and, when provided, must be a JSON string array.
  - [x] AC3 Invalid legacy secret format returns startup config error with explicit code/message.
- Notes: Legacy secrets are used only for matching; new writes always use active secret.

### FR-002 - App-level startup keyset preflight

- Description: App startup flow must verify configured keysets derive the expected index-0 address before serving traffic.
- Acceptance criteria:
  - [x] AC1 Preflight runs during startup initialization path (not only shell scripts).
  - [x] AC2 For each devtest keyset preflight entry, derive index-0 address and compare with `expected_index0_address`.
  - [x] AC3 Any preflight mismatch or invalid key material aborts startup (non-zero exit via startup error path).
- Notes: Address comparison is case-insensitive for both bitcoin and ethereum textual forms.

### FR-003 - Wallet-account sync with legacy-secret fallback

- Description: Startup wallet-account sync must resolve account action using active hash first, then legacy hashes.
- Acceptance criteria:
  - [x] AC1 If active wallet account hash matches active hash (or hash missing), action is `reused` and wallet_account_id remains stable.
  - [x] AC2 If no active match but an inactive historical row matches active or legacy hash, action is `reactivated` and that historical wallet account becomes active.
  - [x] AC3 If no row matches active/legacy hashes, action is `rotated` and a new wallet account is created with `next_index=0`.
- Notes: Exactly one active wallet account per `(chain, network, keyset_id)` must remain after sync.

### FR-004 - Hash upgrade on legacy match

- Description: When match is found via legacy secret hash, selected wallet account row must be re-written with active-secret hash.
- Acceptance criteria:
  - [x] AC1 Selected row stores active-secret digest in `key_material_hash` with algo `hmac-sha256`.
  - [x] AC2 Legacy hash value is not preserved as current active hash in selected row after sync.
- Notes: This makes secret migration converge without forcing wallet-account rotation.

### FR-005 - Persist queryable sync audit events

- Description: Each sync decision must write one event row to a dedicated DB table.
- Acceptance criteria:
  - [x] AC1 Event row includes `chain`, `network`, `keyset_id`, `wallet_account_id`, `action`, `key_material_hash`, `key_material_hash_algo`, timestamp, and structured details.
  - [x] AC2 Event row records whether match source was `active`, `legacy`, or `unhashed`.
  - [x] AC3 Events are queryable by `(chain, network, keyset_id)` and by `wallet_account_id` via indexes.
- Notes: Event rows are append-only; no update-in-place.

### FR-006 - Startup orchestration order

- Description: Startup orchestration must include sync before catalog integrity validation.
- Acceptance criteria:
  - [x] AC1 Initialize persistence flow order is readiness -> migrations -> wallet sync/preflight -> catalog integrity validation.
  - [x] AC2 Any failure in sync/preflight returns startup error and prevents server start.
- Notes: Applies to app runtime startup path.

## Non-functional requirements

- Reliability (NFR-001): Startup sync must be idempotent; repeated startup without config/data change must keep active `wallet_account_id` stable for all keysets.
- Security/Privacy (NFR-002): Raw key material and raw HMAC secret must not be stored in DB or emitted to logs.
- Observability (NFR-003): Startup logs must include action summaries with chain/network/keyset_id/wallet_account_id and hash prefix only.
- Performance (NFR-004): Added startup preflight + sync should complete within 5 seconds for up to 20 keysets on local PostgreSQL.
- Maintainability (NFR-005): New logic must be covered by unit tests for decision branches and config parsing.

## Dependencies and integrations

- External systems: PostgreSQL schema migrations and startup SQL queries.
- Internal services: `InitializePersistenceUseCase`, bootstrap gateway adapter, config loader, wallet derivation primitives.
