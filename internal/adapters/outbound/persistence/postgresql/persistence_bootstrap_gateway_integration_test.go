//go:build integration

package postgresql

import (
	"context"
	"database/sql"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPersistenceBootstrapGateway_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run integration test")
	}

	logger := log.New(io.Discard, "", 0)
	migrationsPath := filepath.Join("migrations")
	gateway := NewPersistenceBootstrapGateway(databaseURL, "integration-target", migrationsPath, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if appErr := gateway.CheckReadiness(ctx); appErr != nil {
		t.Fatalf("expected readiness success, got %v", appErr)
	}

	if appErr := gateway.RunMigrations(ctx); appErr != nil {
		t.Fatalf("expected first migration run success, got %v", appErr)
	}

	if appErr := gateway.RunMigrations(ctx); appErr != nil {
		t.Fatalf("expected second migration run success, got %v", appErr)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var value string
	if err := db.QueryRowContext(ctx, "SELECT value FROM app.bootstrap_metadata WHERE key='bootstrap_version'").Scan(&value); err != nil {
		t.Fatalf("failed to query bootstrap_metadata: %v", err)
	}
	if value != "v1" {
		t.Fatalf("expected bootstrap_version=v1, got %q", value)
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM app.bootstrap_metadata WHERE key='bootstrap_version'").Scan(&count); err != nil {
		t.Fatalf("failed to count bootstrap_metadata rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one bootstrap_version row, got %d", count)
	}
}
