GO ?= go
DATABASE_URL ?= postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable

.PHONY: run test lint compose-up compose-down

run:
	DATABASE_URL=$(DATABASE_URL) $(GO) run ./cmd/server

test:
	$(GO) test ./...

lint:
	$(GO) fmt ./...
	$(GO) vet ./...
	$(GO) list ./... > /dev/null

compose-up:
	docker compose -f deployments/docker-compose.yml up --build

compose-down:
	docker compose -f deployments/docker-compose.yml down
