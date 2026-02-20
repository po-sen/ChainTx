---
doc: 00_problem
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

# Problem & Goals

## Context

- Background: Current keyset safety checks rely on local scripts (`service_verify_keysets.sh`) and are not enforced by the app startup lifecycle itself.
- Background: Wallet-account hash matching currently uses a single HMAC secret. Rotating that secret can cause false key-rotation behavior (new wallet account IDs for unchanged key material).
- Background: Wallet-account sync actions are printed in logs but are not persisted as queryable audit records.
- Users or stakeholders: service operators rotating key material/secrets, backend engineers maintaining startup safety, and QA validating allocation behavior.
- Why now: The user asked to deliver feature set `1+2+3` together: secret-rotation continuity, app-level startup preflight, and audit visibility.

## Constraints (optional)

- Technical constraints: Keep `xpub/tpub/vpub` out of DB; DB can store only hashed fingerprints.
- Technical constraints: Keep chain derivation policy unchanged (bitcoin `bip84_p2wpkh`, ethereum `evm_bip44`, account-level policy enforced).
- Technical constraints: Respect existing Clean Architecture boundaries (application port drives startup orchestration; adapter owns DB/sql details).
- Compliance/security constraints: HMAC algorithm remains `hmac-sha256`; raw secret values must not be logged.

## Problem statement

- Current pain: Startup safety is not guaranteed for all deployment paths because keyset preflight is outside app runtime.
- Current pain: HMAC secret rotation can unintentionally create fresh wallet accounts, resetting to `next_index=0` even when key material is unchanged.
- Current pain: Operators cannot query a persistent audit trail to understand why wallet-account mapping changed (`reused/reactivated/rotated`).

## Goals

- G1: Enforce keyset index-0 preflight inside app startup and fail fast on any mismatch.
- G2: Support active + legacy HMAC secrets so secret rotation does not force unnecessary wallet-account rotation.
- G3: Persist queryable wallet-account sync events with action and context.
- G4: Keep wallet allocation deterministic and idempotent across repeated startups.

## Non-goals (out of scope)

- NG1: New custody providers or production key-management integrations.
- NG2: Proving spend authority with private keys.
- NG3: Exposing new public HTTP API endpoints for audit in this iteration.

## Assumptions

- A1: Devtest deployments will provide nested `PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON` entries with `chain/network/keyset_id/extended_public_key/expected_index0_address`.
- A2: Operators may rotate HMAC secret by setting a new active secret and passing old secrets in an auxiliary env variable.
- A3: Asset catalog rows for enabled assets remain the source of truth for chain/network scope to sync.

## Open questions

- Q1: Resolved in this spec: use optional `PAYMENT_REQUEST_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON` (JSON array) for legacy secret fallback.
- Q2: Resolved in this spec: persist startup sync actions in a dedicated table instead of log-only output.

## Success metrics

- Metric: Startup guard coverage
- Target: In devtest mode, app startup fails when any configured keyset index-0 verification fails.
- Metric: Secret rotation continuity
- Target: Rotating only HMAC secret keeps existing wallet account IDs (no forced `rotated` action) for unchanged key material.
- Metric: Auditability
- Target: Every startup wallet-account sync decision emits one DB row in audit table with action and timestamp.
