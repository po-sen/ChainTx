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
	if cfg.ReconcilerEnabled {
		t.Fatalf("expected reconciler disabled by default")
	}
	if cfg.ReconcilerPollInterval <= 0 {
		t.Fatalf("expected positive default reconciler poll interval")
	}
	if cfg.ReconcilerBatchSize <= 0 {
		t.Fatalf("expected positive default reconciler batch size")
	}
	if cfg.ReconcilerLeaseDuration <= 0 {
		t.Fatalf("expected positive default reconciler lease duration")
	}
	if cfg.ReconcilerWorkerID == "" {
		t.Fatalf("expected default reconciler worker id")
	}
	if cfg.ReconcilerDetectedBPS != 10000 {
		t.Fatalf("expected default detected bps 10000, got %d", cfg.ReconcilerDetectedBPS)
	}
	if cfg.ReconcilerConfirmedBPS != 10000 {
		t.Fatalf("expected default confirmed bps 10000, got %d", cfg.ReconcilerConfirmedBPS)
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
      "extended_public_key": "tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5",
      "expected_index0_address": "tb1qdummy"
    }
  },
  "ethereum": {
    "local": {
      "keyset_id": "ks_eth_local",
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj",
      "expected_index0_address": "0xdummy"
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
	if len(cfg.DevtestKeysetPreflights) != 2 {
		t.Fatalf("expected 2 preflight entries, got %d", len(cfg.DevtestKeysetPreflights))
	}
}

func TestLoadConfigRejectsNestedFormatWithoutKeysetID(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "ethereum": {
    "local": {
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj"
      ,"expected_index0_address":"0xdummy"
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

func TestLoadConfigRejectsNestedFormatWithoutExpectedAddress(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "bitcoin": {
    "testnet": {
      "keyset_id": "ks_btc_testnet",
      "extended_public_key": "tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"
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

func TestLoadConfigParsesLegacyHMACSecrets(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "ethereum": {
    "sepolia": {
      "keyset_id": "ks_eth_sepolia",
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj",
      "expected_index0_address": "0x61ed32e69db70c5abab0522d80e8f5db215965de"
    }
  }
}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON", `["legacy-a", "legacy-b", "legacy-a", " "]`)

	cfg, cfgErr := LoadConfig()
	if cfgErr != nil {
		t.Fatalf("expected no error, got %v", cfgErr)
	}
	if len(cfg.KeysetHashHMACLegacyKeys) != 2 {
		t.Fatalf("expected 2 legacy secrets, got %d", len(cfg.KeysetHashHMACLegacyKeys))
	}
	if cfg.KeysetHashHMACLegacyKeys[0] != "legacy-a" || cfg.KeysetHashHMACLegacyKeys[1] != "legacy-b" {
		t.Fatalf("unexpected legacy secrets: %+v", cfg.KeysetHashHMACLegacyKeys)
	}
}

func TestLoadConfigRejectsInvalidLegacyHMACSecrets(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "ethereum": {
    "sepolia": {
      "keyset_id": "ks_eth_sepolia",
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj",
      "expected_index0_address": "0x61ed32e69db70c5abab0522d80e8f5db215965de"
    }
  }
}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON", `{bad-json`)

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_INVALID" {
		t.Fatalf("expected CONFIG_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigParsesReconcilerConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "ethereum": {
    "local": {
      "keyset_id": "ks_eth_local",
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj",
      "expected_index0_address": "0x61ed32e69db70c5abab0522d80e8f5db215965de"
    }
  }
}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_ENABLED", "true")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_POLL_INTERVAL_SECONDS", "9")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_BATCH_SIZE", "55")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_LEASE_SECONDS", "33")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_WORKER_ID", "reconciler-a")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_DETECTED_THRESHOLD_BPS", "8000")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_CONFIRMED_THRESHOLD_BPS", "9500")
	t.Setenv("PAYMENT_REQUEST_EVM_RPC_URLS_JSON", `{"local":"http://eth-node:8545"}`)

	cfg, cfgErr := LoadConfig()
	if cfgErr != nil {
		t.Fatalf("expected no error, got %v", cfgErr)
	}
	if !cfg.ReconcilerEnabled {
		t.Fatalf("expected reconciler enabled")
	}
	if cfg.ReconcilerPollInterval.Seconds() != 9 {
		t.Fatalf("expected poll interval 9s, got %s", cfg.ReconcilerPollInterval)
	}
	if cfg.ReconcilerBatchSize != 55 {
		t.Fatalf("expected batch size 55, got %d", cfg.ReconcilerBatchSize)
	}
	if cfg.ReconcilerLeaseDuration.Seconds() != 33 {
		t.Fatalf("expected lease duration 33s, got %s", cfg.ReconcilerLeaseDuration)
	}
	if cfg.ReconcilerWorkerID != "reconciler-a" {
		t.Fatalf("expected worker id reconciler-a, got %s", cfg.ReconcilerWorkerID)
	}
	if cfg.ReconcilerDetectedBPS != 8000 {
		t.Fatalf("expected detected bps 8000, got %d", cfg.ReconcilerDetectedBPS)
	}
	if cfg.ReconcilerConfirmedBPS != 9500 {
		t.Fatalf("expected confirmed bps 9500, got %d", cfg.ReconcilerConfirmedBPS)
	}
	if cfg.EVMRPCURLs["local"] != "http://eth-node:8545" {
		t.Fatalf("expected local evm rpc url to be parsed")
	}
}

func TestLoadConfigRejectsReconcilerWithoutEndpoints(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "ethereum": {
    "local": {
      "keyset_id": "ks_eth_local",
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj",
      "expected_index0_address": "0x61ed32e69db70c5abab0522d80e8f5db215965de"
    }
  }
}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_ENABLED", "true")
	t.Setenv("PAYMENT_REQUEST_BTC_ESPLORA_BASE_URL", "")
	t.Setenv("PAYMENT_REQUEST_EVM_RPC_URLS_JSON", "")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_RECONCILER_ENDPOINTS_REQUIRED" {
		t.Fatalf("expected CONFIG_RECONCILER_ENDPOINTS_REQUIRED, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidReconcilerBatchSize(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "ethereum": {
    "local": {
      "keyset_id": "ks_eth_local",
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj",
      "expected_index0_address": "0x61ed32e69db70c5abab0522d80e8f5db215965de"
    }
  }
}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_BATCH_SIZE", "0")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_RECONCILER_BATCH_SIZE_INVALID" {
		t.Fatalf("expected CONFIG_RECONCILER_BATCH_SIZE_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidReconcilerLeaseSeconds(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "ethereum": {
    "local": {
      "keyset_id": "ks_eth_local",
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj",
      "expected_index0_address": "0x61ed32e69db70c5abab0522d80e8f5db215965de"
    }
  }
}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_LEASE_SECONDS", "0")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_RECONCILER_LEASE_SECONDS_INVALID" {
		t.Fatalf("expected CONFIG_RECONCILER_LEASE_SECONDS_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidReconcilerDetectedThresholdBPS(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "ethereum": {
    "local": {
      "keyset_id": "ks_eth_local",
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj",
      "expected_index0_address": "0x61ed32e69db70c5abab0522d80e8f5db215965de"
    }
  }
}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_DETECTED_THRESHOLD_BPS", "0")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_RECONCILER_DETECTED_THRESHOLD_BPS_INVALID" {
		t.Fatalf("expected CONFIG_RECONCILER_DETECTED_THRESHOLD_BPS_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsDetectedThresholdAboveConfirmed(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "ethereum": {
    "local": {
      "keyset_id": "ks_eth_local",
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj",
      "expected_index0_address": "0x61ed32e69db70c5abab0522d80e8f5db215965de"
    }
  }
}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_DETECTED_THRESHOLD_BPS", "9500")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_CONFIRMED_THRESHOLD_BPS", "9000")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_RECONCILER_DETECTED_THRESHOLD_BPS_INVALID" {
		t.Fatalf("expected CONFIG_RECONCILER_DETECTED_THRESHOLD_BPS_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidReconcilerConfirmedThresholdBPS(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "ethereum": {
    "local": {
      "keyset_id": "ks_eth_local",
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj",
      "expected_index0_address": "0x61ed32e69db70c5abab0522d80e8f5db215965de"
    }
  }
}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_CONFIRMED_THRESHOLD_BPS", "10001")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_RECONCILER_CONFIRMED_THRESHOLD_BPS_INVALID" {
		t.Fatalf("expected CONFIG_RECONCILER_CONFIRMED_THRESHOLD_BPS_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidEVMRPCURLs(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{
  "ethereum": {
    "local": {
      "keyset_id": "ks_eth_local",
      "extended_public_key": "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj",
      "expected_index0_address": "0x61ed32e69db70c5abab0522d80e8f5db215965de"
    }
  }
}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_EVM_RPC_URLS_JSON", `{bad-json`)

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_EVM_RPC_URLS_INVALID" {
		t.Fatalf("expected CONFIG_EVM_RPC_URLS_INVALID, got %s", cfgErr.Code)
	}
}
