# ChainTx

Minimal Go service scaffold with:

- `GET /healthz`
- Swagger UI at `/swagger/index.html`
- OpenAPI spec at `/swagger/openapi.yaml`

## Requirements

- Go `1.22+`
- GNU Make

## Quickstart

```bash
make run
```

Server defaults to `http://localhost:8080`.
Set a custom port with:

```bash
PORT=9090 make run
```

## Endpoints

- Health check: `GET http://localhost:8080/healthz`
- Swagger UI: `GET http://localhost:8080/swagger/index.html`
- OpenAPI: `GET http://localhost:8080/swagger/openapi.yaml`

## Local Dev Commands

```bash
make test
make lint
```

## Project Layout

```text
cmd/server/main.go                   # entrypoint
internal/domain                      # pure domain objects
internal/application                 # use cases + ports + DTOs
internal/adapters                    # inbound/outbound adapters
internal/infrastructure              # HTTP server runtime
internal/bootstrap                   # config + DI composition
api/openapi.yaml                     # OpenAPI 3.0.3 contract
```
