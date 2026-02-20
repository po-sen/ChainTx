# AGENTS.md

ChainTx repository development operating manual.
This file consolidates the required behavior from:

- `spec-driven-development`
- `clean-architecture-hexagonal-components`
- `go-project-layout`

Use this as the default execution policy for all feature work, refactors, and fixes.

## 1. Scope and Intent

- Enforce spec-first delivery before implementation.
- Preserve strict Clean Architecture + Hexagonal boundaries.
- Keep Go project layout minimal, predictable, and import-cycle free.
- Produce verifiable outputs (specs, code, tests, and validation evidence).

## 2. Repository Baseline (current)

Detected baseline shape:

- Single Go module (has `go.mod` at repo root).
- No active `go.work` binding for this module.
- Single primary binary entrypoint: `cmd/server/main.go`.
- Core code under `internal/` with layer-aligned structure:
  - `internal/domain`
  - `internal/application`
  - `internal/adapters`
  - `internal/infrastructure`
  - `internal/shared_kernel`
- Existing spec folders under `specs/`.

Default rule:

- Preserve this structure unless explicitly requested to change it.

## 3. Skill Activation and Ordering

When tasks involve product behavior or code changes, apply skills in this order:

1. `spec-driven-development` (define and validate spec package first)
2. `clean-architecture-hexagonal-components` (place and implement logic with boundaries)
3. `go-project-layout` (ensure tree/package placement and verification workflow)

If all three apply, do not skip any.
If a conflict appears, prioritize in this order:

1. Build/test correctness and no import cycles
2. Architecture boundary correctness
3. Spec traceability and readiness gates
4. Layout minimality and consistency

## 4. Mandatory Spec-First Workflow

Do not modify implementation code before a spec package exists.
Minimum requirement is a Quick spec package in `specs/YYYY-MM-DD-slug/`.
Even if the user asks for "draft only", still create/update files in-repo and set `status: DRAFT`.
Never place any text before the opening `---` in spec files; frontmatter must remain valid.

### 4.1 Spec folder naming

- Folder format: `specs/YYYY-MM-DD-slug/`
- `slug` rules:
  - 3-5 keywords
  - lowercase kebab-case
  - remove filler words
  - max 40 chars

### 4.2 Mode selection

Default mode: `Quick`.

Use `Quick` when:

- small change (1-2 endpoints, simple flag, simple refactor)
- no new integration
- no new persistent schema/migration
- no meaningful NFR risk

Use `Full` when any of these is true:

- new DB schema or migration
- new external integration
- non-trivial async/failure flow
- meaningful NFR/security/availability impact

### 4.3 Required files by mode

Quick mode required:

- `00_problem.md`
- `01_requirements.md`
- `03_tasks.md`

Quick mode optional but recommended:

- `04_test_plan.md`

Full mode required:

- `00_problem.md`
- `01_requirements.md`
- `02_design.md`
- `03_tasks.md`
- `04_test_plan.md`

### 4.3.1 Scaffolding source templates (required)

When creating spec docs, copy from templates instead of creating empty files:

- `00_problem.md` from `assets/00_problem_template.md`
- `01_requirements.md` from `assets/01_requirements_template.md`
- `03_tasks.md` from `assets/03_tasks_template.md`
- `02_design.md` from `assets/02_design_template.md` (Full mode)
- `04_test_plan.md` from `assets/04_test_plan_template.md` (Full mode, optional in Quick)

### 4.4 Frontmatter contract (all spec docs)

Every spec file must start with valid YAML frontmatter and include:

- `spec_date`
- `slug`
- `mode` (`Quick` or `Full`)
- `status` (`DRAFT`, `READY`, `DONE`)
- `owners`
- `depends_on`
- `links` object with keys:
  - `problem`
  - `requirements`
  - `design`
  - `tasks`
  - `test_plan`

Rules:

- Keep `spec_date`, `slug`, `mode`, `status`, `depends_on` consistent in all files of the folder.
- Use `links.design: null` when design doc is intentionally absent (Quick mode).
- Links must be either `null` or point to an existing file.
- `owners: []` is allowed only while `status: DRAFT`.
- When mode changes (`Quick` <-> `Full`), update `mode` and `links` in every produced file immediately.

### 4.5 Dependency gate

- `depends_on` entries must reference existing folders under `specs/`.
- Before setting this spec to `READY`, each dependency must be folder-wide `status: DONE`.
- Never include current spec itself in `depends_on`.
- Canonical source for dependency checks: `00_problem.md`.
- Keep `depends_on` identical across all docs and in the same order as `00_problem.md`.
- `depends_on` format:
  - use `depends_on: []` when empty
  - for non-empty, use block list form only
  - do not use non-empty inline list form such as `depends_on: [a, b]`

### 4.6 Required traceability IDs

- Functional requirements: `FR-001`, `FR-002`, ...
- Non-functional requirements: `NFR-001`, ...
- Tasks: `T-001`, `T-002`, ...
- Test cases: `TC-001`, `TC-002`, ...

Rules:

- Every task `T-XXX` must reference one or more `FR/NFR`.
- Every test case `TC-XXX` must reference one or more `FR/NFR`.

### 4.7 Spec quality gates

- Requirements are testable/verifiable.
- NFRs are measurable when applicable.
- Every `FR-XXX` includes explicit acceptance criteria.
- Design (if present) covers:
  - flow
  - data
  - contracts
  - failure modes
  - observability
  - security
- Task plan is ordered and independently verifiable.
- If `04_test_plan.md` is not produced (Quick mode), keep explicit per-task validation steps in
  `03_tasks.md`.

### 4.8 Status lifecycle

- `DRAFT`: creation/clarification in progress
- `READY`: spec complete and lint-checked, implementation can start
- `DONE`: implementation and validation complete, docs reflect actual behavior

Status consistency rule:

- Always keep the same `status` value across all produced docs in the spec folder.
- Promote to `READY` only after lint + dependency gate pass.
- Promote to `DONE` only after implementation + validation complete.

### 4.9 Spec lint command

Run before promoting to `READY` or `DONE`:

```bash
SPEC_DIR="specs/YYYY-MM-DD-slug" bash scripts/spec-lint.sh
```

### 4.10 Clarification and assumption policy

- Ask only the minimum clarifying questions needed for:
  - goal/value
  - scope and non-goals
  - constraints
  - integrations
  - NFRs
  - upstream spec dependencies
- If answers are unavailable, proceed with explicit labeled assumptions in spec docs.
- If immediate coding is requested, first create a minimal spec package, then implement with clearly
  stated assumptions.
- Record mode decision (`Quick` or `Full`) and rationale in `03_tasks.md`.

### 4.11 Ready-to-code checklist (must pass before `READY`)

Quick mode:

- `specs/YYYY-MM-DD-slug/` exists with `00_problem.md`, `01_requirements.md`, `03_tasks.md`.
- Frontmatter fields are real values (`spec_date`, `slug`, `mode`, `status`, `owners`, `depends_on`)
  with no placeholders except `depends_on: []` when no dependencies.
- `owners` has at least one owner/team.
- `depends_on` targets existing spec folders and each dependency is folder-wide `DONE`.
- Frontmatter consistency across docs is preserved (`spec_date`, `slug`, `mode`, `status`, `depends_on`).
- Every produced spec doc includes full `links` key set.
- Mode decision and rationale are documented in `03_tasks.md`.
- Every `FR-XXX` has acceptance criteria.
- Every applicable `NFR-XXX` is measurable.
- Every `T-XXX` references one or more `FR/NFR`.
- If `04_test_plan.md` is skipped, `03_tasks.md` includes explicit task-level validation steps.
- `links.design` stays `null`.
- All links are valid (`null` or existing file).
- `scripts/spec-lint.sh` passes.

Full mode:

- All five docs exist.
- Frontmatter fields are real values (same rules as Quick).
- `owners` has at least one owner/team.
- `depends_on` targets existing `DONE` specs only.
- Frontmatter consistency across docs is preserved.
- Every produced doc includes full `links` key set.
- Every `FR-XXX` has acceptance criteria.
- Every applicable `NFR-XXX` is measurable.
- Design covers flows, data, contracts, failure modes, observability, and security.
- Every `T-XXX` references one or more `FR/NFR`.
- Every `TC-XXX` references one or more `FR/NFR`.
- All links are valid.
- `scripts/spec-lint.sh` passes.

### 4.12 Done checklist (must pass before `DONE`)

- Implementation tasks in `03_tasks.md` are complete.
- Validation evidence is recorded (tests/manual checks/metrics).
- Spec docs reflect final shipped behavior and scope.
- `status: DONE` is applied consistently across produced docs in the folder.

## 5. Architecture Policy (Clean + Hexagonal)

This repository is currently single-module with layer folders under `internal/`.
Treat it as strict Clean Architecture + Hexagonal.

### 5.1 Layer responsibilities

- `internal/domain`

  - entities, value objects, pure policies, domain rules
  - no IO/framework/persistence coupling

- `internal/application`

  - use-case orchestration
  - inbound ports in `ports/in`
  - outbound ports in `ports/out`
  - DTOs and mappers as application contracts

- `internal/adapters/inbound`

  - transport-specific request parsing/validation/mapping
  - call application inbound ports/use cases
  - no business rule execution

- `internal/adapters/outbound`

  - implementation of application outbound ports
  - vendor/driver mapping and ACL translation
  - persistence and external clients

- `internal/infrastructure`

  - composition root, drivers, config, server wiring
  - dependency binding and runtime bootstrap concerns

- `internal/shared_kernel`
  - shared primitives that do not depend on feature modules

### 5.2 Dependency direction (strict)

Allowed high-level direction:

- `domain <- application <- adapters <- infrastructure/bootstrap wiring`

Non-negotiable import rules:

- Domain must not import:
  - `application`
  - `adapters`
  - `infrastructure`
  - `bootstrap`
- Domain events belong in `domain/events` and describe model state changes.
- Cross-component communication must use integration events in shared kernel (or explicit ports/ACL),
  never direct domain imports across components.
- Application may import:
  - domain
  - shared kernel
  - application-internal packages
- Application must not import:
  - adapters
  - infrastructure
  - bootstrap
- Adapters may import:
  - application ports/contracts
  - domain/shared types for mapping only
- Transport schemas/validators belong in `adapters/inbound/<transport>/middleware` (or equivalent),
  not in `domain` or `application`.
- Outbound adapters must not invoke inbound use cases.
- Only composition root binds interfaces to concrete adapters.
- If command bus is used, the bus interface belongs to `application/ports/in`; framework-specific
  wiring stays in infrastructure/composition root.

### 5.3 Use case and port design rules

- One use case equals one primary business intent.
- Define one inbound port per use case.
- Shape outbound ports by core needs, not vendor API shape.
- Separate read models from aggregate repositories:
  - repositories return aggregates/aggregate IDs
  - query services return DTO/view models

### 5.4 Inbound adapter rules

- Validate/parsing of transport payload in inbound adapter layer.
- Map transport models to application commands/DTOs.
- Delegate business action to application use case.
- Convert structured core errors to transport response.

### 5.5 Outbound adapter rules

- Implement outbound ports from `application/ports/out`.
- Keep vendor SDK DTOs and transport models out of domain/application.
- Map external models through adapter ACL mappers.
- Keep business rules in domain/application, never in outbound adapter.

### 5.6 Error model

Use structured errors:

- `type`
- `code`
- `message`
- optional metadata

Transport adapters map structured errors to HTTP/CLI/MQ responses.

### 5.7 Domain modeling rules

- Prefer value objects/policies/specifications for pure rules.
- Use domain services only when logic does not belong naturally to one entity/value object.
- Domain services must not do IO or call repositories.

### 5.8 Components/bounded contexts (future-safe rule)

If repository evolves to multiple bounded contexts:

- keep each component isolated (`components/<name>/...` or equivalent)
- no direct component-to-component domain/application imports
- communicate via shared kernel events, ACL, or explicit query ports

### 5.9 Naming guidance

- Inbound: `<Verb><Noun>UseCase`, `<Verb><Noun>Handler`, `<Verb><Noun>Port`
- Outbound: `<Noun>Repository`, `<Capability>Gateway`, `<Capability>Publisher`
- Read-side queries: `<Noun>ReadModel`, `<Noun>QueryService`, `<Noun>Finder`
- Vendor client adapters: `<Vendor><Capability>Client` and keep vendor mapping in ACL mappers

## 6. Required SOLID Review Gate

Before finalizing implementation, answer all with "Yes" unless an exception is explicitly documented:

- SRP: one use case has one reason to change
- OCP: new transport/vendor mostly via new adapter, not core edits
- LSP: alternative adapter implementations satisfy same port contracts
- ISP: consumers depend only on methods they use
- DIP: framework/driver objects created only in composition roots

If any answer is "No":

- treat as design defect
- document exception and trade-off in spec/design notes

## 7. Go Project Layout Policy

Apply `go-project-layout` while preserving existing repo conventions.

### 7.1 Baseline shape for this repo

- Keep module files at root:
  - `go.mod`
  - `go.sum`
- Keep entrypoint(s) in `cmd/<app>/main.go`
- Keep non-public app code in `internal/`
- Do not introduce `src/`
- Before structural edits, inspect module/workspace state with:
  - `go env GOMOD`
  - `go env GOWORK`

### 7.2 `cmd` and `internal` usage

- `cmd/` contains bootstrap/wiring only.
- Business logic must stay in `internal/`.
- For this single-binary service, default to `internal/<domain>/...` and existing layered folders.

### 7.3 `pkg` usage policy

- Do not create `pkg/` unless API is intentionally public and stable.
- If added, require:
  - owner declaration
  - compatibility commitment
  - explicit reason in docs

### 7.4 Optional directories

Only create when concrete artifacts exist:

- `api/` for contracts/generator configs (not runtime handlers)
- `configs/` for non-secret examples/defaults
- `scripts/` for developer or CI scripts
- `build/`, `deployments/` for packaging/deploy assets
- `test/` for black-box assets or slow integration/e2e harnesses
- Do not put runtime handlers/controllers/server implementations under `api/`.

### 7.5 Go verification workflow after edits

Run (or explain why skipped):

```bash
go fmt ./...   # or repo formatter if CI mandates another formatter
go mod tidy
go list ./...
go test -short ./...
go vet ./...
```

Optional workflows when needed:

- full suite: `go test ./...`
- tagged suites: `go test -tags=integration ./...`, `go test -tags=e2e ./...`

Must pass:

- no import cycles
- package boundaries remain clean

### 7.6 Multi-module and `pkg/` constraints

- Default to one module per repository.
- Add extra modules only with explicit ownership/release boundaries.
- If multiple modules are required, wire with `go work` and document ownership.
- Keep shared code in `internal/` first; promote to `pkg/` only for intentional, stable external API.

## 8. Testing Taxonomy and Placement

### 8.1 Unit tests

- Scope: domain rules and application use-case orchestration
- Infra: mocks/fakes only, no real DB
- Placement: close to `internal/domain/**` and `internal/application/**`

### 8.2 Integration tests

- Scope: adapter + dependency collaboration
- Infra: real dependency preferred for persistence adapters
- Placement: adapter/infrastructure test locations
- Include contract tests for port implementation compatibility

### 8.3 Functional tests

- Scope: black-box user flow via real inbound interface
- Infra: fully wired app path, state reset per test
- Placement: repo-standard functional/e2e location

## 9. Default End-to-End Execution SOP

For any non-trivial request:

1. Understand request and classify scope/risk.
2. Scan architecture/layout cues in repo.
3. Scaffold spec folder (`DRAFT`) and required files.
4. Clarify gaps or declare explicit assumptions.
5. Decide Quick/Full mode and update links/frontmatter.
6. Complete problem/requirements/tasks (and design/test plan if required).
7. Run spec-lint and dependency gate checks.
8. Promote spec to `READY` only when gates pass.
9. Implement with strict architecture + layout rules.
10. Add/adjust tests per taxonomy.
11. Run Go verification workflow.
12. Update spec status:

- `DONE` only after implementation + validation are complete.

## 10. Definition of Done (engineering)

A change is done only when all are true:

- Spec exists in `specs/YYYY-MM-DD-slug/` and status is accurate.
- Requirements/tasks/test traceability is preserved.
- Architecture boundaries have no violations.
- Go layout conventions remain consistent.
- Tests are updated and relevant suites pass.
- Validation evidence is available in commit/notes/spec.

## 11. Explicit Anti-Patterns

Do not do any of the following:

- Start coding without a spec package.
- Put business rules in HTTP/controller/router or outbound clients.
- Import adapters/infrastructure into application/domain.
- Return read-model DTOs from aggregate repository ports.
- Put server/runtime handler implementation under `api/`.
- Introduce `pkg/` as a dumping ground.
- Create deep folders before needed.
- Add new architecture patterns (CQRS/event sourcing/command bus) unless requested or already present.

## 12. Documentation and Change Notes

When layout or architecture decisions are non-obvious:

- document rationale in spec/design docs
- keep docs aligned with actual implemented behavior
- prefer concise, testable, and auditable statements

This file is the standing implementation policy for this repository.

## 13. Self-Verification Command

Run a one-command policy check before finishing substantial edits:

```bash
bash scripts/verify-agents.sh
```

Optional explicit target spec:

```bash
bash scripts/verify-agents.sh specs/YYYY-MM-DD-slug
```
