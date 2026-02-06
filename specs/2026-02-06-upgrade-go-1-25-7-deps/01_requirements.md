---
doc: 01_requirements
spec_date: 2026-02-06
slug: upgrade-go-1-25-7-deps
mode: Quick
status: DONE
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

# Requirements

## Glossary (optional)

- Go directive: The minimum Go language version declared in `go.mod` (for example `go 1.25`).
- Toolchain target: The concrete runtime/compiler baseline used by local and CI environments (for
  this spec, `go1.25.7`).

## Out-of-scope behaviors

- OOS1: Any API contract or handler behavior change.
- OOS2: Domain/application/adapter package refactor.
- OOS3: CI platform migration.

## Functional requirements

### FR-001 - Upgrade module toolchain target to Go 1.25.7

- Description: Update module-level Go version configuration to target the requested `1.25.7`
  toolchain.
- Acceptance criteria:
  - [ ] `go.mod` declares `go 1.25.7`.
  - [ ] No remaining project-level references imply `1.22` as active baseline.
- Notes: Keep declarations explicit so local and CI toolchains resolve deterministically.

### FR-002 - Reconcile dependencies with upgraded Go toolchain

- Description: Refresh dependency graph to a clean state under Go `1.25.7`.
- Acceptance criteria:
  - [ ] `go mod tidy` runs successfully.
  - [ ] `go.mod`/`go.sum` reflect only required dependencies.
  - [ ] Dependency updates do not introduce compile failures.
- Notes: Upgrade only what is needed for compatibility and hygiene.

### FR-003 - Preserve existing service behavior after upgrade

- Description: Ensure health and docs endpoints continue to work with no behavioral regression.
- Acceptance criteria:
  - [ ] `go test ./...` passes.
  - [ ] Existing integration tests for HTTP routes pass unchanged.
  - [ ] Local run path (`make run`) still starts service successfully.
- Notes: This is a maintenance change, not a feature change.

### FR-004 - Update developer guidance to match new baseline

- Description: Reflect new Go baseline in developer-facing documentation/config.
- Acceptance criteria:
  - [ ] `README.md` Go requirement text references `1.25.7` baseline.
  - [ ] Local workflow commands remain accurate for upgraded toolchain.
- Notes: Documentation must match actual enforced version.

## Non-functional requirements

- Performance (NFR-001): `go test ./...` wall-clock runtime should not regress by more than 20%
  versus pre-upgrade baseline on the same machine.
- Availability/Reliability (NFR-002): Build/test commands must complete with exit code `0` in a
  clean checkout with Go `1.25.7`.
- Security/Privacy (NFR-003): No newly added dependencies with known critical vulnerabilities in
  default `govulncheck` output.
- Compliance (NFR-004): Keep dependency set OSS and Go-module-compatible; no proprietary additions.
- Observability (NFR-005): Existing startup/shutdown and error logging behavior remains unchanged.
- Maintainability (NFR-006): Keep project layout aligned to `cmd/` + `internal/` conventions; avoid
  structural churn for a version-only upgrade.

## Dependencies and integrations

- External systems: None.
- Internal services: Existing HTTP service components only.
- Toolchain/runtime: Go `1.25.7`.
- Key libraries: `github.com/swaggo/http-swagger/v2`, transitive Swagger/OpenAPI packages.
