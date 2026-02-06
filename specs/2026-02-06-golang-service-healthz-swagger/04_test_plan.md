---
doc: 04_test_plan
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

# Test Plan

## Scope

- Covered: HTTP server startup, `/healthz`, Swagger route availability, OpenAPI correctness for health endpoint, Makefile local workflows.
- Not covered: Deployment infra tests, load testing beyond lightweight local checks, auth/data-layer behavior.

## Tests

### Unit

- TC-001: Health handler returns OK JSON

  - Linked requirements: FR-002, NFR-003
  - Steps: Invoke health handler with `GET /healthz` using `httptest`.
  - Expected: HTTP `200`, JSON body includes `status=ok`, no sensitive fields.

- TC-002: Health route rejects non-GET method
  - Linked requirements: FR-002
  - Steps: Send `POST /healthz` to router.
  - Expected: Non-`200` response (e.g., `404` or `405` per router behavior).

### Integration

- TC-101: Server routes expose health and swagger paths

  - Linked requirements: FR-001, FR-002, FR-003
  - Steps: Boot router in test server; call `/healthz`, `/swagger`, `/swagger/index.html`, and OpenAPI file endpoint.
  - Expected: Health returns `200`; docs endpoints return redirect/HTML/spec as defined; OpenAPI top-level field is `openapi: 3.0.3`.

- TC-102: Makefile test target executes Go tests
  - Linked requirements: FR-004
  - Steps: Run `make test` from repo root.
  - Expected: Command exits `0`; test suite reports pass.

### E2E (if applicable)

- Scenario 1: Local boot and manual probe

  - Linked requirements: FR-001, FR-004, FR-005
  - Steps: Run `make run`; `curl http://localhost:8080/healthz`.
  - Expected: Server starts and endpoint returns healthy response.

- Scenario 2: Open docs in browser
  - Linked requirements: FR-003, FR-005
  - Steps: Navigate to `http://localhost:8080/swagger/index.html`.
  - Expected: Swagger UI loads and shows `/healthz` operation from an OpenAPI `3.0.3` spec.

## Edge cases and failure modes

- Case: Port already in use.
- Expected behavior: Startup fails fast with clear error log and non-zero exit code. (FR-001, NFR-002)

- Case: Missing OpenAPI spec file at runtime.
- Expected behavior: Docs endpoint returns error status; server remains running for `/healthz`. (FR-003)

## NFR verification

- Performance: Run 50 sequential local `GET /healthz` requests; p95 <= `50ms` (NFR-001).
- Reliability: Send `SIGINT` during runtime and verify graceful shutdown log appears (NFR-002).
- Security: Confirm responses never include env var values or stack traces in normal flows (NFR-003).
