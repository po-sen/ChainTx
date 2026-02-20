//go:build !integration

package config

import "testing"

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PORT", "")
	t.Setenv("OPENAPI_SPEC_PATH", "")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "devtest-hmac-secret")

	cfg, cfgErr := LoadConfig()
	if cfgErr != nil {
		t.Fatalf("expected no error, got %v", cfgErr)
	}

	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.Port)
	}

	if cfg.OpenAPISpecPath != "api/openapi.yaml" {
		t.Fatalf("expected default openapi path, got %s", cfg.OpenAPISpecPath)
	}

	if cfg.DatabaseTarget != "localhost:5432/chaintx" {
		t.Fatalf("expected parsed database target, got %s", cfg.DatabaseTarget)
	}
	if cfg.AllocationMode != "devtest" {
		t.Fatalf("expected default allocation mode devtest, got %s", cfg.AllocationMode)
	}
	if cfg.DevtestAllowMainnet {
		t.Fatalf("expected default devtest allow mainnet false")
	}
}

func TestLoadConfigRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}

	if cfgErr.Code != "CONFIG_DATABASE_URL_REQUIRED" {
		t.Fatalf("expected CONFIG_DATABASE_URL_REQUIRED, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidScheme(t *testing.T) {
	t.Setenv("DATABASE_URL", "mysql://localhost:3306/chaintx")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}

	if cfgErr.Code != "CONFIG_DATABASE_URL_SCHEME_INVALID" {
		t.Fatalf("expected CONFIG_DATABASE_URL_SCHEME_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRequiresDevtestKeysets(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_ALLOCATION_MODE", "devtest")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", "")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_DEVTEST_KEYSETS_REQUIRED" {
		t.Fatalf("expected CONFIG_DEVTEST_KEYSETS_REQUIRED, got %s", cfgErr.Code)
	}
}

func TestLoadConfigAllowsCustomAllocationMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_ALLOCATION_MODE", "custom-mode")

	cfg, cfgErr := LoadConfig()
	if cfgErr != nil {
		t.Fatalf("expected no error, got %v", cfgErr)
	}
	if cfg.AllocationMode != "custom-mode" {
		t.Fatalf("expected custom-mode, got %s", cfg.AllocationMode)
	}
}

func TestLoadConfigAddressSchemeAllowListOverride(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "devtest-hmac-secret")
	t.Setenv("PAYMENT_REQUEST_ADDRESS_SCHEME_ALLOW_LIST_JSON", `{"bitcoin":["bip84_p2wpkh","alt_btc"],"ethereum":["evm_bip44"]}`)

	cfg, cfgErr := LoadConfig()
	if cfgErr != nil {
		t.Fatalf("expected no error, got %v", cfgErr)
	}

	if _, ok := cfg.AddressSchemeAllowList["bitcoin"]["alt_btc"]; !ok {
		t.Fatalf("expected overridden allow list to include bitcoin/alt_btc")
	}
}

func TestLoadConfigRejectsInvalidAddressSchemeAllowList(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "devtest-hmac-secret")
	t.Setenv("PAYMENT_REQUEST_ADDRESS_SCHEME_ALLOW_LIST_JSON", `{bad-json`)

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_ADDRESS_SCHEME_ALLOW_LIST_INVALID" {
		t.Fatalf("expected CONFIG_ADDRESS_SCHEME_ALLOW_LIST_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigSupportsDevtestKeysetsObjectFormat(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_eth_local":{"extended_public_key":"xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj"}}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "devtest-hmac-secret")

	cfg, cfgErr := LoadConfig()
	if cfgErr != nil {
		t.Fatalf("expected no error, got %v", cfgErr)
	}
	if cfg.DevtestKeysets["ks_eth_local"] == "" {
		t.Fatalf("expected object-format keyset to be parsed")
	}
}

func TestLoadConfigSupportsDevtestKeysetsNestedFormat(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "bitcoin": {
    "regtest": {
      "keyset_id": "ks_btc_regtest",
      "extended_public_key": "tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"
    }
  },
  "ethereum": {
    "local": {
      "keyset_id": "ks_eth_local",
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj"
    }
  }
}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "devtest-hmac-secret")

	cfg, cfgErr := LoadConfig()
	if cfgErr != nil {
		t.Fatalf("expected no error, got %v", cfgErr)
	}
	if cfg.DevtestKeysets["ks_btc_regtest"] == "" {
		t.Fatalf("expected nested-format keyset ks_btc_regtest to be parsed")
	}
	if cfg.DevtestKeysets["ks_eth_local"] == "" {
		t.Fatalf("expected nested-format keyset ks_eth_local to be parsed")
	}
}

func TestLoadConfigRejectsNestedFormatWithoutKeysetID(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "ethereum": {
    "local": {
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj"
    }
  }
}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "devtest-hmac-secret")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_DEVTEST_KEYSETS_INVALID" {
		t.Fatalf("expected CONFIG_DEVTEST_KEYSETS_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRequiresKeysetHashSecretInDevtestMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_ALLOCATION_MODE", "devtest")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_KEYSET_HASH_HMAC_SECRET_REQUIRED" {
		t.Fatalf("expected CONFIG_KEYSET_HASH_HMAC_SECRET_REQUIRED, got %s", cfgErr.Code)
	}
}
