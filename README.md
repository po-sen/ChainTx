# ChainTx

Minimal Go backend scaffold with Clean Architecture + Hexagonal boundaries and local PostgreSQL
compose workflow.

## Features

- `GET /healthz`
- Swagger UI at `/swagger/index.html`
- OpenAPI spec at `/swagger/openapi.yaml`
- Startup-time PostgreSQL readiness + `golang-migrate` migrations

## Requirements

- Go `1.25.7`
- GNU Make
- Docker Engine + Docker Compose plugin (for local stack)

## Local Quickstart (Compose)

```bash
make compose-up
```

Then verify:

```bash
curl -i http://localhost:8080/healthz
```

Stop services:

```bash
make compose-down
```

## Local Quickstart (Host Go process)

Start only PostgreSQL:

```bash
docker compose -f deployments/docker-compose.yml up -d postgres
```

Run app from host:

```bash
DATABASE_URL=postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable make run
```

## Configuration

- `DATABASE_URL` (required): the only app-level DB configuration input.
  - Compose default:
    `postgresql://chaintx:chaintx@postgres:5432/chaintx?sslmode=disable`
- `PORT` (optional, default `8080`)
- `OPENAPI_SPEC_PATH` (optional, default `api/openapi.yaml`)

## Endpoints

- Health check: `GET http://localhost:8080/healthz`
- Swagger UI: `GET http://localhost:8080/swagger/index.html`
- OpenAPI: `GET http://localhost:8080/swagger/openapi.yaml`

## Local Dev Commands

```bash
make test
make lint
```

## Troubleshooting

- Symptom: app exits during startup with DB initialization error.
  - Check PostgreSQL container health:
    `docker compose -f deployments/docker-compose.yml ps`
  - Verify `DATABASE_URL` host/port/user/password/database are correct.
  - If schema state is broken locally, reset dev volume:
    `docker compose -f deployments/docker-compose.yml down -v`

## Project Layout

```text
cmd/server/main.go                               # entrypoint orchestration
internal/domain                                  # pure domain objects
internal/application                             # use cases + ports + DTOs
internal/adapters                                # inbound/outbound adapters
internal/infrastructure                          # runtime server/infrastructure
internal/bootstrap                               # config + DI composition root
internal/adapters/outbound/persistence/postgresql/migrations
                                                 # golang-migrate SQL files
api/openapi.yaml                                 # OpenAPI 3.0.3 contract
build/package/Dockerfile                         # container image build recipe
deployments/docker-compose.yml                   # local app + postgres stack
```
