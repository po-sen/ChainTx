FROM golang:1.25.7 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/server /usr/local/bin/server
COPY api ./api
COPY internal/adapters/outbound/persistence/postgresql/migrations ./internal/adapters/outbound/persistence/postgresql/migrations

ENV PORT=8080
ENV OPENAPI_SPEC_PATH=/app/api/openapi.yaml

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/server"]
