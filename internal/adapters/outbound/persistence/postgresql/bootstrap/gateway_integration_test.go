//go:build integration

package bootstrap

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPersistenceBootstrapGatewayIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run integration test")
	}
	keysets := map[string]string{
		"ks_btc_regtest": "tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5",
		"ks_btc_testnet": "vpub5Xzfrm6ouSBPKVriRpkXyai4mvsHjRHq28wxS1znBCdwzLzeJUx8ruJeBnCMKs1AyqYsJ2mriQHuzxNoFtkkq94J4bJyNjGXkbZ8vCYwUy3",
		"ks_eth_sepolia": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj",
	}
	rawKeysets := os.Getenv("TEST_DEVTEST_KEYSETS_JSON")
	if rawKeysets != "" {
		override := map[string]string{}
		if err := json.Unmarshal([]byte(rawKeysets), &override); err != nil {
			t.Fatalf("failed to parse TEST_DEVTEST_KEYSETS_JSON: %v", err)
		}
		for key, value := range override {
			keysets[key] = value
		}
	}

	logger := log.New(io.Discard, "", 0)
	migrationsPath := filepath.Join("..", "migrations")
	resetDatabaseForMigrations(t, databaseURL)
	gateway := NewGateway(
		databaseURL,
		"integration-target",
		migrationsPath,
		ValidationRules{
			AllocationMode:      "devtest",
			DevtestKeysets:      keysets,
			DevtestAllowMainnet: false,
			AddressSchemeAllowList: map[string]map[string]struct{}{
				"bitcoin": {
					"bip84_p2wpkh": {},
				},
				"ethereum": {
					"evm_bip44": {},
				},
			},
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if appErr := gateway.CheckReadiness(ctx); appErr != nil {
		t.Fatalf("expected readiness success, got %v", appErr)
	}

	if appErr := gateway.RunMigrations(ctx); appErr != nil {
		t.Fatalf("expected first migration run success, got %v", appErr)
	}

	if appErr := gateway.ValidateAssetCatalogIntegrity(ctx); appErr != nil {
		t.Fatalf("expected asset catalog integrity validation success, got %v", appErr)
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

	var assetCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM app.asset_catalog WHERE enabled = TRUE").Scan(&assetCount); err != nil {
		t.Fatalf("failed to count enabled assets: %v", err)
	}
	if assetCount < 3 {
		t.Fatalf("expected at least 3 enabled assets, got %d", assetCount)
	}
}

func resetDatabaseForMigrations(t *testing.T, databaseURL string) {
	t.Helper()
	assertSafeIntegrationDatabaseURL(t, databaseURL)

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("failed to open db for reset: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err = db.ExecContext(ctx, `
DROP SCHEMA IF EXISTS app CASCADE;
DROP TABLE IF EXISTS schema_migrations;
`)
	if err != nil {
		t.Fatalf("failed to reset migration state: %v", err)
	}
}

func assertSafeIntegrationDatabaseURL(t *testing.T, databaseURL string) {
	t.Helper()

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("invalid TEST_DATABASE_URL: %v", err)
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	dbName := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(parsed.Path), "/"))
	hostAllowed := host == "localhost" || host == "127.0.0.1" || host == "postgres"
	dbAllowed := dbName == "chaintx" || strings.Contains(dbName, "test")

	if !hostAllowed || !dbAllowed {
		t.Fatalf("unsafe TEST_DATABASE_URL for destructive integration reset: host=%q db=%q", host, dbName)
	}
}
