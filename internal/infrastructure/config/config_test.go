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
	if cfg.ReconcilerBTCMinConf != 1 {
		t.Fatalf("expected default btc min confirmations 1, got %d", cfg.ReconcilerBTCMinConf)
	}
	if cfg.ReconcilerBTCFinalityMinConf != 1 {
		t.Fatalf("expected default btc finality min confirmations 1, got %d", cfg.ReconcilerBTCFinalityMinConf)
	}
	if cfg.ReconcilerEVMMinConf != 1 {
		t.Fatalf("expected default evm min confirmations 1, got %d", cfg.ReconcilerEVMMinConf)
	}
	if cfg.ReconcilerEVMFinalityMinConf != 1 {
		t.Fatalf("expected default evm finality min confirmations 1, got %d", cfg.ReconcilerEVMFinalityMinConf)
	}
	if cfg.ReconcilerReorgObserveWindow <= 0 {
		t.Fatalf("expected positive default reorg observe window")
	}
	if cfg.ReconcilerStabilityCycles != 1 {
		t.Fatalf("expected default reconciler stability cycles 1, got %d", cfg.ReconcilerStabilityCycles)
	}
	if cfg.WebhookEnabled {
		t.Fatalf("expected webhook disabled by default")
	}
	if cfg.WebhookPollInterval <= 0 {
		t.Fatalf("expected positive default webhook poll interval")
	}
	if cfg.WebhookBatchSize <= 0 {
		t.Fatalf("expected positive default webhook batch size")
	}
	if cfg.WebhookLeaseDuration <= 0 {
		t.Fatalf("expected positive default webhook lease duration")
	}
	if cfg.WebhookWorkerID == "" {
		t.Fatalf("expected default webhook worker id")
	}
	if cfg.WebhookTimeout <= 0 {
		t.Fatalf("expected positive default webhook timeout")
	}
	if cfg.WebhookMaxAttempts <= 0 {
		t.Fatalf("expected positive default webhook max attempts")
	}
	if cfg.WebhookInitialBackoff <= 0 || cfg.WebhookMaxBackoff <= 0 {
		t.Fatalf("expected positive webhook backoff defaults")
	}
	if cfg.WebhookRetryJitterBPS != 0 {
		t.Fatalf("expected default webhook retry jitter bps 0, got %d", cfg.WebhookRetryJitterBPS)
	}
	if cfg.WebhookRetryBudget != 0 {
		t.Fatalf("expected default webhook retry budget 0, got %d", cfg.WebhookRetryBudget)
	}
	if cfg.WebhookAlertEnabled {
		t.Fatalf("expected webhook alert disabled by default")
	}
	if cfg.WebhookAlertCooldown.Seconds() != 300 {
		t.Fatalf("expected default webhook alert cooldown 300s, got %s", cfg.WebhookAlertCooldown)
	}
	if cfg.WebhookAlertFailedCount != 0 {
		t.Fatalf("expected default webhook alert failed count threshold 0, got %d", cfg.WebhookAlertFailedCount)
	}
	if cfg.WebhookAlertPendingReady != 0 {
		t.Fatalf("expected default webhook alert pending ready threshold 0, got %d", cfg.WebhookAlertPendingReady)
	}
	if cfg.WebhookAlertOldestAgeSec != 0 {
		t.Fatalf("expected default webhook alert oldest age threshold 0, got %d", cfg.WebhookAlertOldestAgeSec)
	}
	if len(cfg.WebhookOpsAdminKeys) != 0 {
		t.Fatalf("expected default webhook ops admin keys empty, got %d", len(cfg.WebhookOpsAdminKeys))
	}
	if len(cfg.WebhookURLAllowList) == 0 {
		t.Fatalf("expected default webhook allowlist")
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
	t.Setenv("PAYMENT_REQUEST_RECONCILER_BTC_MIN_CONFIRMATIONS", "2")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_BTC_FINALITY_MIN_CONFIRMATIONS", "6")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_EVM_MIN_CONFIRMATIONS", "3")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_EVM_FINALITY_MIN_CONFIRMATIONS", "8")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_REORG_OBSERVE_WINDOW_SECONDS", "7200")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_STABILITY_CYCLES", "2")
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
	if cfg.ReconcilerBTCMinConf != 2 {
		t.Fatalf("expected btc min confirmations 2, got %d", cfg.ReconcilerBTCMinConf)
	}
	if cfg.ReconcilerBTCFinalityMinConf != 6 {
		t.Fatalf("expected btc finality min confirmations 6, got %d", cfg.ReconcilerBTCFinalityMinConf)
	}
	if cfg.ReconcilerEVMMinConf != 3 {
		t.Fatalf("expected evm min confirmations 3, got %d", cfg.ReconcilerEVMMinConf)
	}
	if cfg.ReconcilerEVMFinalityMinConf != 8 {
		t.Fatalf("expected evm finality min confirmations 8, got %d", cfg.ReconcilerEVMFinalityMinConf)
	}
	if cfg.ReconcilerReorgObserveWindow.Seconds() != 7200 {
		t.Fatalf("expected reorg observe window 7200s, got %s", cfg.ReconcilerReorgObserveWindow)
	}
	if cfg.ReconcilerStabilityCycles != 2 {
		t.Fatalf("expected stability cycles 2, got %d", cfg.ReconcilerStabilityCycles)
	}
	if cfg.EVMRPCURLs["local"] != "http://eth-node:8545" {
		t.Fatalf("expected local evm rpc url to be parsed")
	}
}

func TestLoadConfigRejectsBTCFinalityLessThanBusiness(t *testing.T) {
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
	t.Setenv("PAYMENT_REQUEST_RECONCILER_BTC_MIN_CONFIRMATIONS", "3")
	t.Setenv("PAYMENT_REQUEST_RECONCILER_BTC_FINALITY_MIN_CONFIRMATIONS", "2")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_RECONCILER_BTC_FINALITY_MIN_CONFIRMATIONS_INVALID" {
		t.Fatalf("expected CONFIG_RECONCILER_BTC_FINALITY_MIN_CONFIRMATIONS_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidReorgObserveWindow(t *testing.T) {
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
	t.Setenv("PAYMENT_REQUEST_RECONCILER_REORG_OBSERVE_WINDOW_SECONDS", "0")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_RECONCILER_REORG_OBSERVE_WINDOW_SECONDS_INVALID" {
		t.Fatalf("expected CONFIG_RECONCILER_REORG_OBSERVE_WINDOW_SECONDS_INVALID, got %s", cfgErr.Code)
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

func TestLoadConfigRejectsInvalidBTCMinConfirmations(t *testing.T) {
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
	t.Setenv("PAYMENT_REQUEST_RECONCILER_BTC_MIN_CONFIRMATIONS", "0")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_RECONCILER_BTC_MIN_CONFIRMATIONS_INVALID" {
		t.Fatalf("expected CONFIG_RECONCILER_BTC_MIN_CONFIRMATIONS_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidEVMMinConfirmations(t *testing.T) {
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
	t.Setenv("PAYMENT_REQUEST_RECONCILER_EVM_MIN_CONFIRMATIONS", "not-a-number")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_RECONCILER_EVM_MIN_CONFIRMATIONS_INVALID" {
		t.Fatalf("expected CONFIG_RECONCILER_EVM_MIN_CONFIRMATIONS_INVALID, got %s", cfgErr.Code)
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

func TestLoadConfigParsesWebhookConfig(t *testing.T) {
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
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_ENABLED", "true")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_URL_ALLOWLIST_JSON", `["hooks.example.com","*.partners.example.com"]`)
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_HMAC_SECRET", "webhook-secret")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_POLL_INTERVAL_SECONDS", "7")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_BATCH_SIZE", "25")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_LEASE_SECONDS", "21")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_WORKER_ID", "webhook-worker-a")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_TIMEOUT_SECONDS", "4")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_MAX_ATTEMPTS", "9")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_INITIAL_BACKOFF_SECONDS", "6")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_MAX_BACKOFF_SECONDS", "120")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_RETRY_JITTER_BPS", "1800")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_RETRY_BUDGET", "4")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_ALERT_ENABLED", "true")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_ALERT_COOLDOWN_SECONDS", "90")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_ALERT_FAILED_COUNT_THRESHOLD", "11")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_ALERT_PENDING_READY_THRESHOLD", "12")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_ALERT_OLDEST_PENDING_AGE_SECONDS", "13")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_OPS_ADMIN_KEYS_JSON", `["ops-key-a","ops-key-b"]`)

	cfg, cfgErr := LoadConfig()
	if cfgErr != nil {
		t.Fatalf("expected no error, got %v", cfgErr)
	}
	if !cfg.WebhookEnabled {
		t.Fatalf("expected webhook enabled")
	}
	if len(cfg.WebhookURLAllowList) != 2 {
		t.Fatalf("expected allowlist size 2, got %d", len(cfg.WebhookURLAllowList))
	}
	if cfg.WebhookURLAllowList[0] != "hooks.example.com" {
		t.Fatalf("unexpected allowlist[0]: %s", cfg.WebhookURLAllowList[0])
	}
	if cfg.WebhookHMACSecret != "webhook-secret" {
		t.Fatalf("unexpected webhook hmac secret")
	}
	if cfg.WebhookPollInterval.Seconds() != 7 {
		t.Fatalf("expected webhook poll interval 7s, got %s", cfg.WebhookPollInterval)
	}
	if cfg.WebhookBatchSize != 25 {
		t.Fatalf("expected webhook batch size 25, got %d", cfg.WebhookBatchSize)
	}
	if cfg.WebhookLeaseDuration.Seconds() != 21 {
		t.Fatalf("expected webhook lease duration 21s, got %s", cfg.WebhookLeaseDuration)
	}
	if cfg.WebhookWorkerID != "webhook-worker-a" {
		t.Fatalf("expected webhook worker id webhook-worker-a, got %s", cfg.WebhookWorkerID)
	}
	if cfg.WebhookTimeout.Seconds() != 4 {
		t.Fatalf("expected webhook timeout 4s, got %s", cfg.WebhookTimeout)
	}
	if cfg.WebhookMaxAttempts != 9 {
		t.Fatalf("expected webhook max attempts 9, got %d", cfg.WebhookMaxAttempts)
	}
	if cfg.WebhookInitialBackoff.Seconds() != 6 {
		t.Fatalf("expected webhook initial backoff 6s, got %s", cfg.WebhookInitialBackoff)
	}
	if cfg.WebhookMaxBackoff.Seconds() != 120 {
		t.Fatalf("expected webhook max backoff 120s, got %s", cfg.WebhookMaxBackoff)
	}
	if cfg.WebhookRetryJitterBPS != 1800 {
		t.Fatalf("expected webhook retry jitter bps 1800, got %d", cfg.WebhookRetryJitterBPS)
	}
	if cfg.WebhookRetryBudget != 4 {
		t.Fatalf("expected webhook retry budget 4, got %d", cfg.WebhookRetryBudget)
	}
	if !cfg.WebhookAlertEnabled {
		t.Fatalf("expected webhook alert enabled")
	}
	if cfg.WebhookAlertCooldown.Seconds() != 90 {
		t.Fatalf("expected webhook alert cooldown 90s, got %s", cfg.WebhookAlertCooldown)
	}
	if cfg.WebhookAlertFailedCount != 11 {
		t.Fatalf("expected webhook alert failed count threshold 11, got %d", cfg.WebhookAlertFailedCount)
	}
	if cfg.WebhookAlertPendingReady != 12 {
		t.Fatalf("expected webhook alert pending ready threshold 12, got %d", cfg.WebhookAlertPendingReady)
	}
	if cfg.WebhookAlertOldestAgeSec != 13 {
		t.Fatalf("expected webhook alert oldest age threshold 13, got %d", cfg.WebhookAlertOldestAgeSec)
	}
	if len(cfg.WebhookOpsAdminKeys) != 2 {
		t.Fatalf("expected webhook ops admin keys size 2, got %d", len(cfg.WebhookOpsAdminKeys))
	}
	if cfg.WebhookOpsAdminKeys[0] != "ops-key-a" || cfg.WebhookOpsAdminKeys[1] != "ops-key-b" {
		t.Fatalf("unexpected webhook ops admin keys: %+v", cfg.WebhookOpsAdminKeys)
	}
}

func TestLoadConfigAllowsWebhookWithoutURLForNonDispatcherRuntime(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_ENABLED", "true")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_HMAC_SECRET", "webhook-secret")

	cfg, cfgErr := LoadConfig()
	if cfgErr != nil {
		t.Fatalf("expected no error, got %v", cfgErr)
	}
	if !cfg.WebhookEnabled {
		t.Fatalf("expected webhook enabled")
	}
}

func TestLoadConfigRejectsWebhookMaxBackoffLessThanInitial(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_ENABLED", "true")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_HMAC_SECRET", "webhook-secret")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_INITIAL_BACKOFF_SECONDS", "60")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_MAX_BACKOFF_SECONDS", "30")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_WEBHOOK_MAX_BACKOFF_INVALID" {
		t.Fatalf("expected CONFIG_WEBHOOK_MAX_BACKOFF_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidWebhookRetryJitterBPS(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_RETRY_JITTER_BPS", "10001")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_WEBHOOK_RETRY_JITTER_BPS_INVALID" {
		t.Fatalf("expected CONFIG_WEBHOOK_RETRY_JITTER_BPS_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidWebhookRetryBudget(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_RETRY_BUDGET", "-1")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_WEBHOOK_RETRY_BUDGET_INVALID" {
		t.Fatalf("expected CONFIG_WEBHOOK_RETRY_BUDGET_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidWebhookAlertEnabled(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_ALERT_ENABLED", "not-a-bool")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_WEBHOOK_ALERT_ENABLED_INVALID" {
		t.Fatalf("expected CONFIG_WEBHOOK_ALERT_ENABLED_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidWebhookAlertCooldown(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_ALERT_COOLDOWN_SECONDS", "0")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_WEBHOOK_ALERT_COOLDOWN_SECONDS_INVALID" {
		t.Fatalf("expected CONFIG_WEBHOOK_ALERT_COOLDOWN_SECONDS_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidWebhookAlertThreshold(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_ALERT_PENDING_READY_THRESHOLD", "-1")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_WEBHOOK_ALERT_PENDING_READY_THRESHOLD_INVALID" {
		t.Fatalf("expected CONFIG_WEBHOOK_ALERT_PENDING_READY_THRESHOLD_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsWebhookAlertEnabledWithoutThreshold(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_ALERT_ENABLED", "true")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_WEBHOOK_ALERT_THRESHOLD_REQUIRED" {
		t.Fatalf("expected CONFIG_WEBHOOK_ALERT_THRESHOLD_REQUIRED, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidWebhookOpsAdminKeys(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_OPS_ADMIN_KEYS_JSON", `{bad-json`)

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_WEBHOOK_OPS_ADMIN_KEYS_INVALID" {
		t.Fatalf("expected CONFIG_WEBHOOK_OPS_ADMIN_KEYS_INVALID, got %s", cfgErr.Code)
	}
}

func TestLoadConfigRejectsInvalidWebhookURLAllowlist(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON", `{"ks_btc_testnet":"tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"}`)
	t.Setenv("PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET", "active-secret")
	t.Setenv("PAYMENT_REQUEST_WEBHOOK_URL_ALLOWLIST_JSON", `["https://example.com"]`)

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}
	if cfgErr.Code != "CONFIG_WEBHOOK_URL_ALLOWLIST_INVALID" {
		t.Fatalf("expected CONFIG_WEBHOOK_URL_ALLOWLIST_INVALID, got %s", cfgErr.Code)
	}
}
