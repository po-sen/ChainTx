GO ?= go

.PHONY: run test lint

run:
	$(GO) run ./cmd/server

test:
	$(GO) test ./...

lint:
	$(GO) fmt ./...
	$(GO) vet ./...
	$(GO) list ./... > /dev/null
