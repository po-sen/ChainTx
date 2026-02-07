---
doc: 00_problem
spec_date: 2026-02-07
slug: postgresql-backend-service-compose
mode: Full
status: DONE
owners:
  - posen
depends_on:
  - 2026-02-06-golang-service-healthz-swagger
links:
  problem: 00_problem.md
  requirements: 01_requirements.md
  design: 02_design.md
  tasks: 03_tasks.md
  test_plan: 04_test_plan.md
---

# Problem & Goals

## Context

- Background: The current Go service provides `GET /healthz` and Swagger docs but has no database integration and no local container orchestration workflow.
- Users or stakeholders: Backend engineers, QA engineers, and local developers who need a reproducible backend environment with PostgreSQL.
- Why now: Upcoming backend features require persistent storage, and local setup must be one-command and consistent across contributors.

## Constraints (optional)

- Technical constraints: Keep the existing single-module clean architecture layout (`internal/domain`, `internal/application`, `internal/adapters`, `internal/bootstrap`) and add PostgreSQL through outbound ports/adapters only.
- Technical constraints: Use `golang-migrate` as the primary SQL migration tool for startup schema management.
- Timeline/cost constraints: Deliver a minimal first increment that enables local development and verification without introducing production deployment complexity.
- Compliance/security constraints: Avoid committing non-local secrets; provide DB credentials via a single `DATABASE_URL` environment variable.

## Problem statement

- Current pain: Service behavior is fully in-memory and does not exercise a real database dependency, making future persistence features risky and slower to validate.
- Evidence or examples: Repository had no DB config, no persistence adapter, and no compose deployment manifest for app + database startup.

## Goals

- G1: Define and implement a PostgreSQL integration path that respects clean architecture and hexagonal boundaries.
- G2: Add a simple local `docker compose` workflow that starts PostgreSQL (`postgres:latest`) and the backend service together.
- G3: Ensure service startup verifies DB availability/migrations before serving traffic.
- G4: Keep Go project layout boundaries clear (`cmd/` for entrypoint, `internal/` for private implementation).

## Non-goals (out of scope)

- NG1: No production-grade orchestration (Kubernetes, Terraform, managed DB services).
- NG2: No broad business-domain CRUD API expansion beyond minimal DB integration needs.
- NG3: No multi-database support in this phase.

## Assumptions

- A1: Local developers have Docker Engine and Docker Compose plugin available.
- A2: Local Compose uses official PostgreSQL image tag `postgres:latest`.
- A3: Existing service remains a single binary (`cmd/server`) and does not require module splitting.

## Open questions

- Q1: Should DB availability be reflected in existing `GET /healthz` response, or remain startup-only in this increment?

## Success metrics

- Metric: Local startup workflow
- Target: `docker compose -f deployments/docker-compose.yml up --build` starts app + PostgreSQL from repo root and exposes backend HTTP on `localhost:8080`.
- Metric: Runtime DB configuration surface
- Target: Backend service accepts DB connection input from only one app-level variable: `DATABASE_URL`.
- Metric: DB dependency enforcement
- Target: Service fails startup with non-zero exit when PostgreSQL is unreachable or migrations fail.
- Metric: Architecture consistency
- Target: DB driver usage is confined to outbound adapters/infrastructure; no domain/application imports from adapters/infrastructure.
