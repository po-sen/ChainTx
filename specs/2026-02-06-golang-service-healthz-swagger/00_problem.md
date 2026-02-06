---
doc: 00_problem
spec_date: 2026-02-06
slug: golang-service-healthz-swagger
mode: Quick
status: READY
owners:
  - posen
depends_on: []
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: null
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Problem & Goals

## Context

- Background: The repository currently has no Go backend service scaffold, so there is no executable API surface for development or deployment.
- Users or stakeholders: Backend engineers, QA engineers, and local developers who need a stable starter service.
- Why now: A production-grade baseline is required before adding business APIs.

## Constraints (optional)

- Technical constraints: Implement with Go; keep architecture simple but maintainable; do not add database or external system coupling.
- Timeline/cost constraints: Deliver a single small feature set that can be run locally with one command.
- Compliance/security constraints: Expose only non-sensitive endpoints (`GET /healthz`, Swagger docs) without secrets in responses.

## Problem statement

- Current pain: No running server, no health probe, and no API contract documentation endpoint.
- Evidence or examples: `README.md` exists, but no `go.mod`, no executable `main`, and no API routes.

## Goals

- G1: Create a minimal Go backend service that starts locally and handles HTTP requests.
- G2: Provide `GET /healthz` returning a clear healthy response.
- G3: Provide Swagger UI/docs under `/swagger` so API consumers can inspect the contract.
- G4: Provide `Makefile` targets to run, test, and validate locally.

## Non-goals (out of scope)

- NG1: Authentication/authorization, persistence, or external service integrations.
- NG2: Production deployment manifests (Docker/Kubernetes/Terraform).
- NG3: Multiple business endpoints beyond health check.

## Assumptions

- A1: Go toolchain version is at least `1.22`, enabling modern standard library features.
- A2: Swagger documentation uses a static OpenAPI `3.0.3` file plus local Swagger UI handler.
- A3: A default local port `8080` is acceptable unless overridden by `PORT` environment variable.

## Open questions

- Q1: Should future APIs enforce auth at the gateway layer or service layer? (outside this scope)
- Q2: Should `/healthz` later include dependency checks (DB/cache)? (outside this scope)

## Success metrics

- Metric: Local startup workflow
- Target: `make run` starts a listening service on localhost without manual file edits.
- Metric: Health endpoint behavior
- Target: `GET /healthz` returns HTTP `200` with deterministic JSON payload.
- Metric: API docs accessibility
- Target: `GET /swagger/index.html` returns Swagger UI and loads OpenAPI `3.0.3` for `/healthz`.
