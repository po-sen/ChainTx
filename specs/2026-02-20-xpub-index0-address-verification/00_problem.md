---
doc: 00_problem
spec_date: 2026-02-20
slug: xpub-index0-address-verification
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-10-wallet-allocation-asset-catalog
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Problem & Goals

## Context

- Background: Current system validates xpub/tpub/vpub format and derivation policy, but operators still need a direct way to verify that a provided `xpub` derives the expected index `0` address.
- Background: Current wallet-account rows do not auto-rotate when xpub material changes, so index progression can continue from old key material unless manually reset.
- Background: Current deployment flow does not block service startup when xpub/index-0 expectation mismatches; verification is operator-manual.
- Users or stakeholders: service operator, backend engineer, QA who prepares keysets and wants to avoid misconfiguration before receiving funds.
- Why now: User explicitly requested both `xpub + index 0 address` correctness checks and automatic wallet account rotation without persisting raw xpub in DB.

## Constraints (optional)

- Technical constraints: Reuse existing wallet derivation primitives in `internal/infrastructure/walletkeys` and current chain/address-scheme policies.
- Technical constraints: Do not persist raw extended public key material in relational tables.
- Timeline/cost constraints: Keep scope scoped to keyset config contract, hash-based rotation, and verifier command in current release.
- Compliance/security constraints: Do not require private key (`xprv`) input or any custody action in this feature.

## Problem statement

- Current pain: Visual/manual validation of xpub strings is error-prone and cannot prove whether the configured material derives the expected address.
- Current pain: When xpub changes for an existing keyset, cursor state (`next_index`) may continue from previous material, creating operational confusion.
- Evidence or examples: A malformed or wrong-account xpub may still look plausible; mismatch is discovered late during payment testing.

## Goals

- G1: Provide deterministic verification that compares derived index `0` address from provided key material to user-provided expected address.
- G2: Return explicit pass/fail and reason when key material is malformed, violates account-level policy, or does not match expected address.
- G3: Provide an operator-friendly execution flow that can be run immediately when user provides `xpub` and expected index `0` address.
- G4: Automatically rotate active wallet account on xpub change while preserving historical cursor continuity when switching back to a previously used key hash.
- G5: Persist only hashed key material fingerprint in DB for equality checks; raw xpub remains outside DB.
- G6: Enforce startup preflight verification for configured keysets; any verification failure blocks service startup.

## Non-goals (out of scope)

- NG1: Proving private-key spend authority on-chain (this feature only verifies deterministic public derivation match).
- NG2: Adding new production custody/provider logic.
- NG3: Supporting arbitrary derivation templates beyond existing policy (`0/{index}`).
- NG4: Encrypting and storing raw xpub in DB (explicitly out of scope; only hash is allowed).

## Assumptions

- A1: User can provide `chain`, `network`, `address_scheme`, serialized extended public key, and expected address for `index=0`.
- A2: Verification follows existing chain-specific derivation rules in `walletkeys` (Bitcoin: `bip84_p2wpkh`; Ethereum: `evm_bip44`) using the provided `chain` and `network`.
- A3: Hash comparison uses configured key-material string with `hmac-sha256`; equal input string implies equal hash.

## Open questions

- Q1: Resolved - use `hmac-sha256` with required env secret.
- Q2: Resolved - add DB migration + startup backfill behavior for active rows missing hash.

## Success metrics

- Metric: Verification correctness
- Target: Known-good fixtures return pass; known-bad/mismatch fixtures return fail with typed reason.
- Metric: Operator latency
- Target: Single verification completes within 1 second on a local developer machine.
- Metric: Automatic rotation correctness
- Target: xpub hash change to unseen hash creates new `wallet_account_id` with `next_index=0`; hash reverting to previously used value reactivates matching historical wallet account and keeps its `next_index`.
- Metric: Startup gate correctness
- Target: `service-up` exits non-zero and does not start app when any configured keyset fails index-0 verification.
