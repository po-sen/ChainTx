GO ?= go
DATABASE_URL ?= postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable
TEST_DATABASE_URL ?= postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable

.PHONY: run test test-integration lint compose-up compose-down

run:
	DATABASE_URL=$(DATABASE_URL) $(GO) run ./cmd/server

test:
	$(GO) test ./...

test-integration:
	TEST_DATABASE_URL=$(TEST_DATABASE_URL) $(GO) test -p 1 -count=1 -tags=integration ./...

lint:
	$(GO) fmt ./...
	$(GO) vet ./...
	$(GO) list ./... > /dev/null

compose-up:
	docker compose -f deployments/docker-compose.yml up --build

compose-down:
	docker compose -f deployments/docker-compose.yml down
