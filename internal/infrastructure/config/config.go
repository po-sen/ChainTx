package config

import (
	"encoding/json"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort                     = "8080"
	defaultOpenAPISpec              = "api/openapi.yaml"
	defaultShutdownTimeout          = 10 * time.Second
	defaultDBReadinessTimeout       = 30 * time.Second
	defaultDBReadinessRetryInterval = 2 * time.Second
	defaultMigrationsPath           = "internal/adapters/outbound/persistence/postgresql/migrations"
	defaultAllocationMode           = "devtest"
	defaultKeysetHashAlgo           = "hmac-sha256"
	defaultReconcilerPollInterval   = 15 * time.Second
	defaultReconcilerBatchSize      = 100
	defaultReconcilerLeaseDuration  = 30 * time.Second
	defaultDetectedThresholdBPS     = 10000
	defaultConfirmedThresholdBPS    = 10000
	defaultBTCMinConfirmations      = 1
	defaultEVMMinConfirmations      = 1
)

const addressSchemeAllowListEnv = "PAYMENT_REQUEST_ADDRESS_SCHEME_ALLOW_LIST_JSON"
const devtestKeysetsEnv = "PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON"
const keysetHashHMACSecretEnv = "PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET"
const keysetHashHMACPreviousSecretsEnv = "PAYMENT_REQUEST_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_JSON"
const reconcilerEnabledEnv = "PAYMENT_REQUEST_RECONCILER_ENABLED"
const reconcilerPollIntervalEnv = "PAYMENT_REQUEST_RECONCILER_POLL_INTERVAL_SECONDS"
const reconcilerBatchSizeEnv = "PAYMENT_REQUEST_RECONCILER_BATCH_SIZE"
const reconcilerLeaseSecondsEnv = "PAYMENT_REQUEST_RECONCILER_LEASE_SECONDS"
const reconcilerWorkerIDEnv = "PAYMENT_REQUEST_RECONCILER_WORKER_ID"
const reconcilerDetectedThresholdBPSEnv = "PAYMENT_REQUEST_RECONCILER_DETECTED_THRESHOLD_BPS"
const reconcilerConfirmedThresholdBPSEnv = "PAYMENT_REQUEST_RECONCILER_CONFIRMED_THRESHOLD_BPS"
const reconcilerBTCMinConfirmationsEnv = "PAYMENT_REQUEST_RECONCILER_BTC_MIN_CONFIRMATIONS"
const reconcilerEVMMinConfirmationsEnv = "PAYMENT_REQUEST_RECONCILER_EVM_MIN_CONFIRMATIONS"
const btcExploraBaseURLEnv = "PAYMENT_REQUEST_BTC_ESPLORA_BASE_URL"
const evmRPCURLsEnv = "PAYMENT_REQUEST_EVM_RPC_URLS_JSON"

type ConfigError struct {
	Code     string
	Message  string
	Metadata map[string]string
}

func (e *ConfigError) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

type Config struct {
	Port                     string
	OpenAPISpecPath          string
	ShutdownTimeout          time.Duration
	DatabaseURL              string
	DatabaseTarget           string
	DBReadinessTimeout       time.Duration
	DBReadinessRetryInterval time.Duration
	MigrationsPath           string
	AllocationMode           string
	DevtestAllowMainnet      bool
	DevtestKeysets           map[string]string
	DevtestKeysetPreflights  []DevtestKeysetPreflightEntry
	KeysetHashAlgorithm      string
	KeysetHashHMACSecret     string
	KeysetHashHMACLegacyKeys []string
	ReconcilerEnabled        bool
	ReconcilerPollInterval   time.Duration
	ReconcilerBatchSize      int
	ReconcilerLeaseDuration  time.Duration
	ReconcilerWorkerID       string
	ReconcilerDetectedBPS    int
	ReconcilerConfirmedBPS   int
	ReconcilerBTCMinConf     int
	ReconcilerEVMMinConf     int
	BTCExploraBaseURL        string
	EVMRPCURLs               map[string]string
	AddressSchemeAllowList   map[string]map[string]struct{}
}

func LoadConfig() (Config, *ConfigError) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, &ConfigError{
			Code:    "CONFIG_DATABASE_URL_REQUIRED",
			Message: "DATABASE_URL is required",
		}
	}

	databaseTarget, parseErr := parseDatabaseTarget(databaseURL)
	if parseErr != nil {
		return Config{}, parseErr
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	openAPISpecPath := os.Getenv("OPENAPI_SPEC_PATH")
	if openAPISpecPath == "" {
		openAPISpecPath = defaultOpenAPISpec
	}

	allocationMode := strings.ToLower(strings.TrimSpace(os.Getenv("PAYMENT_REQUEST_ALLOCATION_MODE")))
	if allocationMode == "" {
		allocationMode = defaultAllocationMode
	}

	allowMainnet := false
	rawAllowMainnet := strings.TrimSpace(os.Getenv("PAYMENT_REQUEST_DEVTEST_ALLOW_MAINNET"))
	if rawAllowMainnet != "" {
		parsedAllowMainnet, err := strconv.ParseBool(rawAllowMainnet)
		if err != nil {
			return Config{}, &ConfigError{
				Code:    "CONFIG_DEVTEST_ALLOW_MAINNET_INVALID",
				Message: "PAYMENT_REQUEST_DEVTEST_ALLOW_MAINNET must be a boolean",
			}
		}
		allowMainnet = parsedAllowMainnet
	}

	rawDevtestKeysets := strings.TrimSpace(os.Getenv(devtestKeysetsEnv))
	devtestKeysets, devtestKeysetPreflights, keysetErr := parseDevtestKeysets(rawDevtestKeysets)
	if keysetErr != nil {
		return Config{}, keysetErr
	}
	if allocationMode == "devtest" && len(devtestKeysets) == 0 {
		return Config{}, &ConfigError{
			Code:    "CONFIG_DEVTEST_KEYSETS_REQUIRED",
			Message: devtestKeysetsEnv + " is required for devtest allocation mode",
		}
	}
	keysetHashHMACSecret := strings.TrimSpace(os.Getenv(keysetHashHMACSecretEnv))
	if allocationMode == "devtest" && keysetHashHMACSecret == "" {
		return Config{}, &ConfigError{
			Code:    "CONFIG_KEYSET_HASH_HMAC_SECRET_REQUIRED",
			Message: keysetHashHMACSecretEnv + " is required for devtest allocation mode",
		}
	}
	keysetHashLegacyKeys, legacySecretErr := parseLegacyHMACSecrets(
		strings.TrimSpace(os.Getenv(keysetHashHMACPreviousSecretsEnv)),
	)
	if legacySecretErr != nil {
		return Config{}, legacySecretErr
	}
	reconcilerCfg, reconcilerErr := parseReconcilerConfig()
	if reconcilerErr != nil {
		return Config{}, reconcilerErr
	}
	reconcilerWorkerID := strings.TrimSpace(os.Getenv(reconcilerWorkerIDEnv))
	if reconcilerWorkerID == "" {
		reconcilerWorkerID = defaultReconcilerWorkerID()
	}
	btcExploraBaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv(btcExploraBaseURLEnv)), "/")
	evmRPCURLs, evmRPCURLsErr := parseEVMRPCURLs(strings.TrimSpace(os.Getenv(evmRPCURLsEnv)))
	if evmRPCURLsErr != nil {
		return Config{}, evmRPCURLsErr
	}
	if reconcilerCfg.Enabled && btcExploraBaseURL == "" && len(evmRPCURLs) == 0 {
		return Config{}, &ConfigError{
			Code:    "CONFIG_RECONCILER_ENDPOINTS_REQUIRED",
			Message: "at least one observer endpoint is required when reconciler is enabled",
		}
	}

	addressSchemeAllowList, allowListErr := loadAddressSchemeAllowList()
	if allowListErr != nil {
		return Config{}, allowListErr
	}

	return Config{
		Port:                     port,
		OpenAPISpecPath:          openAPISpecPath,
		ShutdownTimeout:          defaultShutdownTimeout,
		DatabaseURL:              databaseURL,
		DatabaseTarget:           databaseTarget,
		DBReadinessTimeout:       defaultDBReadinessTimeout,
		DBReadinessRetryInterval: defaultDBReadinessRetryInterval,
		MigrationsPath:           defaultMigrationsPath,
		AllocationMode:           allocationMode,
		DevtestAllowMainnet:      allowMainnet,
		DevtestKeysets:           devtestKeysets,
		DevtestKeysetPreflights:  devtestKeysetPreflights,
		KeysetHashAlgorithm:      defaultKeysetHashAlgo,
		KeysetHashHMACSecret:     keysetHashHMACSecret,
		KeysetHashHMACLegacyKeys: keysetHashLegacyKeys,
		ReconcilerEnabled:        reconcilerCfg.Enabled,
		ReconcilerPollInterval:   reconcilerCfg.PollInterval,
		ReconcilerBatchSize:      reconcilerCfg.BatchSize,
		ReconcilerLeaseDuration:  reconcilerCfg.LeaseDuration,
		ReconcilerWorkerID:       reconcilerWorkerID,
		ReconcilerDetectedBPS:    reconcilerCfg.DetectedBPS,
		ReconcilerConfirmedBPS:   reconcilerCfg.ConfirmedBPS,
		ReconcilerBTCMinConf:     reconcilerCfg.BTCMinConfirmations,
		ReconcilerEVMMinConf:     reconcilerCfg.EVMMinConfirmations,
		BTCExploraBaseURL:        btcExploraBaseURL,
		EVMRPCURLs:               evmRPCURLs,
		AddressSchemeAllowList:   addressSchemeAllowList,
	}, nil
}

func (c Config) Address() string {
	return ":" + c.Port
}

func parseDatabaseTarget(databaseURL string) (string, *ConfigError) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", &ConfigError{
			Code:    "CONFIG_DATABASE_URL_INVALID",
			Message: "DATABASE_URL is invalid",
		}
	}

	switch parsed.Scheme {
	case "postgres", "postgresql":
	default:
		return "", &ConfigError{
			Code:    "CONFIG_DATABASE_URL_SCHEME_INVALID",
			Message: "DATABASE_URL must use postgres or postgresql scheme",
		}
	}

	if parsed.Host == "" {
		return "", &ConfigError{
			Code:    "CONFIG_DATABASE_URL_HOST_MISSING",
			Message: "DATABASE_URL host is required",
		}
	}

	databaseName := strings.TrimPrefix(parsed.Path, "/")
	if databaseName == "" {
		return "", &ConfigError{
			Code:    "CONFIG_DATABASE_NAME_MISSING",
			Message: "DATABASE_URL database name is required",
		}
	}

	return parsed.Host + "/" + databaseName, nil
}

func loadAddressSchemeAllowList() (map[string]map[string]struct{}, *ConfigError) {
	raw := strings.TrimSpace(os.Getenv(addressSchemeAllowListEnv))
	if raw == "" {
		return defaultAddressSchemeAllowList(), nil
	}

	decoded := map[string][]string{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, &ConfigError{
			Code:    "CONFIG_ADDRESS_SCHEME_ALLOW_LIST_INVALID",
			Message: "PAYMENT_REQUEST_ADDRESS_SCHEME_ALLOW_LIST_JSON must be a JSON object of string arrays",
		}
	}

	allowList := map[string]map[string]struct{}{}
	for chain, schemes := range decoded {
		normalizedChain := strings.ToLower(strings.TrimSpace(chain))
		if normalizedChain == "" {
			continue
		}

		if _, exists := allowList[normalizedChain]; !exists {
			allowList[normalizedChain] = map[string]struct{}{}
		}

		for _, scheme := range schemes {
			normalizedScheme := strings.ToLower(strings.TrimSpace(scheme))
			if normalizedScheme == "" {
				continue
			}
			allowList[normalizedChain][normalizedScheme] = struct{}{}
		}
	}

	if len(allowList) == 0 {
		return nil, &ConfigError{
			Code:    "CONFIG_ADDRESS_SCHEME_ALLOW_LIST_EMPTY",
			Message: "PAYMENT_REQUEST_ADDRESS_SCHEME_ALLOW_LIST_JSON must define at least one chain/scheme pair",
		}
	}

	return allowList, nil
}

func defaultAddressSchemeAllowList() map[string]map[string]struct{} {
	return map[string]map[string]struct{}{
		"bitcoin": {
			"bip84_p2wpkh": {},
		},
		"ethereum": {
			"evm_bip44": {},
		},
	}
}

type devtestKeysetEnvelope struct {
	ExtendedPublicKey string `json:"extended_public_key"`
	KeyMaterial       string `json:"key_material"`
	XPub              string `json:"xpub"`
}

type devtestScopedKeysetEnvelope struct {
	KeysetID          string `json:"keyset_id"`
	ExtendedPublicKey string `json:"extended_public_key"`
	KeyMaterial       string `json:"key_material"`
	XPub              string `json:"xpub"`
	ExpectedAddress   string `json:"expected_index0_address"`
}

type DevtestKeysetPreflightEntry struct {
	Chain                 string
	Network               string
	KeysetID              string
	ExtendedPublicKey     string
	ExpectedIndex0Address string
}

func parseDevtestKeysets(raw string) (map[string]string, []DevtestKeysetPreflightEntry, *ConfigError) {
	if raw == "" {
		return map[string]string{}, []DevtestKeysetPreflightEntry{}, nil
	}

	entries := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, nil, &ConfigError{
			Code:    "CONFIG_DEVTEST_KEYSETS_INVALID",
			Message: devtestKeysetsEnv + " must be a JSON object",
		}
	}

	if parsedFlat, isFlat, cfgErr := parseFlatDevtestKeysets(entries); cfgErr != nil {
		return nil, nil, cfgErr
	} else if isFlat {
		return parsedFlat, []DevtestKeysetPreflightEntry{}, nil
	}

	return parseNestedDevtestKeysets(entries)
}

func parseFlatDevtestKeysets(entries map[string]json.RawMessage) (map[string]string, bool, *ConfigError) {
	keysets := map[string]string{}
	for keysetID, payload := range entries {
		normalizedKeysetID := strings.TrimSpace(keysetID)
		if normalizedKeysetID == "" {
			continue
		}

		var keyAsString string
		if err := json.Unmarshal(payload, &keyAsString); err == nil {
			trimmed := strings.TrimSpace(keyAsString)
			if trimmed == "" {
				return nil, false, &ConfigError{
					Code:    "CONFIG_DEVTEST_KEYSETS_INVALID",
					Message: devtestKeysetsEnv + " contains empty key material",
					Metadata: map[string]string{
						"keyset_id": normalizedKeysetID,
					},
				}
			}
			keysets[normalizedKeysetID] = trimmed
			continue
		}

		envelope := devtestKeysetEnvelope{}
		if err := json.Unmarshal(payload, &envelope); err != nil {
			// Not flat format; likely nested chain/network format.
			return nil, false, nil
		}

		resolvedKey := resolveEnvelopeKeyMaterial(envelope.ExtendedPublicKey, envelope.KeyMaterial, envelope.XPub)
		if resolvedKey == "" {
			// Object without key material likely means nested chain/network format.
			return nil, false, nil
		}
		keysets[normalizedKeysetID] = resolvedKey
	}

	return keysets, true, nil
}

func parseNestedDevtestKeysets(entries map[string]json.RawMessage) (map[string]string, []DevtestKeysetPreflightEntry, *ConfigError) {
	keysets := map[string]string{}
	preflights := []DevtestKeysetPreflightEntry{}

	for chain, chainPayload := range entries {
		normalizedChain := strings.ToLower(strings.TrimSpace(chain))
		if normalizedChain == "" {
			continue
		}

		networkEntries := map[string]json.RawMessage{}
		if err := json.Unmarshal(chainPayload, &networkEntries); err != nil {
			return nil, nil, &ConfigError{
				Code:    "CONFIG_DEVTEST_KEYSETS_INVALID",
				Message: devtestKeysetsEnv + " nested format must be chain->network objects",
				Metadata: map[string]string{
					"chain": normalizedChain,
				},
			}
		}

		for network, networkPayload := range networkEntries {
			normalizedNetwork := strings.ToLower(strings.TrimSpace(network))
			if normalizedNetwork == "" {
				continue
			}

			envelope := devtestScopedKeysetEnvelope{}
			if err := json.Unmarshal(networkPayload, &envelope); err != nil {
				return nil, nil, &ConfigError{
					Code:    "CONFIG_DEVTEST_KEYSETS_INVALID",
					Message: devtestKeysetsEnv + " nested entries must be objects with keyset_id and extended_public_key",
					Metadata: map[string]string{
						"chain":   normalizedChain,
						"network": normalizedNetwork,
					},
				}
			}

			keysetID := strings.TrimSpace(envelope.KeysetID)
			if keysetID == "" {
				return nil, nil, &ConfigError{
					Code:    "CONFIG_DEVTEST_KEYSETS_INVALID",
					Message: devtestKeysetsEnv + " nested entry is missing keyset_id",
					Metadata: map[string]string{
						"chain":   normalizedChain,
						"network": normalizedNetwork,
					},
				}
			}

			keyMaterial := resolveEnvelopeKeyMaterial(envelope.ExtendedPublicKey, envelope.KeyMaterial, envelope.XPub)
			if keyMaterial == "" {
				return nil, nil, &ConfigError{
					Code:    "CONFIG_DEVTEST_KEYSETS_INVALID",
					Message: devtestKeysetsEnv + " nested entry is missing extended_public_key",
					Metadata: map[string]string{
						"chain":     normalizedChain,
						"network":   normalizedNetwork,
						"keyset_id": keysetID,
					},
				}
			}
			expectedAddress := strings.TrimSpace(envelope.ExpectedAddress)
			if expectedAddress == "" {
				return nil, nil, &ConfigError{
					Code:    "CONFIG_DEVTEST_KEYSETS_INVALID",
					Message: devtestKeysetsEnv + " nested entry is missing expected_index0_address",
					Metadata: map[string]string{
						"chain":     normalizedChain,
						"network":   normalizedNetwork,
						"keyset_id": keysetID,
					},
				}
			}

			if existing, exists := keysets[keysetID]; exists && existing != keyMaterial {
				return nil, nil, &ConfigError{
					Code:    "CONFIG_DEVTEST_KEYSETS_INVALID",
					Message: devtestKeysetsEnv + " defines conflicting key material for keyset_id",
					Metadata: map[string]string{
						"keyset_id": keysetID,
					},
				}
			}
			keysets[keysetID] = keyMaterial
			preflights = append(preflights, DevtestKeysetPreflightEntry{
				Chain:                 normalizedChain,
				Network:               normalizedNetwork,
				KeysetID:              keysetID,
				ExtendedPublicKey:     keyMaterial,
				ExpectedIndex0Address: expectedAddress,
			})
		}
	}

	return keysets, preflights, nil
}

func parseLegacyHMACSecrets(raw string) ([]string, *ConfigError) {
	if raw == "" {
		return []string{}, nil
	}

	rawSecrets := []string{}
	if err := json.Unmarshal([]byte(raw), &rawSecrets); err != nil {
		return nil, &ConfigError{
			Code:    "CONFIG_KEYSET_HASH_HMAC_PREVIOUS_SECRETS_INVALID",
			Message: keysetHashHMACPreviousSecretsEnv + " must be a JSON array of strings",
		}
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(rawSecrets))
	for _, secret := range rawSecrets {
		trimmed := strings.TrimSpace(secret)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}

	return out, nil
}

type reconcilerRuntimeConfig struct {
	Enabled             bool
	PollInterval        time.Duration
	BatchSize           int
	LeaseDuration       time.Duration
	DetectedBPS         int
	ConfirmedBPS        int
	BTCMinConfirmations int
	EVMMinConfirmations int
}

func parseReconcilerConfig() (reconcilerRuntimeConfig, *ConfigError) {
	enabled := false
	rawEnabled := strings.TrimSpace(os.Getenv(reconcilerEnabledEnv))
	if rawEnabled != "" {
		parsed, err := strconv.ParseBool(rawEnabled)
		if err != nil {
			return reconcilerRuntimeConfig{}, &ConfigError{
				Code:    "CONFIG_RECONCILER_ENABLED_INVALID",
				Message: reconcilerEnabledEnv + " must be a boolean",
			}
		}
		enabled = parsed
	}

	pollInterval := defaultReconcilerPollInterval
	rawPollInterval := strings.TrimSpace(os.Getenv(reconcilerPollIntervalEnv))
	if rawPollInterval != "" {
		seconds, err := strconv.Atoi(rawPollInterval)
		if err != nil || seconds <= 0 {
			return reconcilerRuntimeConfig{}, &ConfigError{
				Code:    "CONFIG_RECONCILER_POLL_INTERVAL_INVALID",
				Message: reconcilerPollIntervalEnv + " must be a positive integer in seconds",
			}
		}
		pollInterval = time.Duration(seconds) * time.Second
	}

	batchSize := defaultReconcilerBatchSize
	rawBatchSize := strings.TrimSpace(os.Getenv(reconcilerBatchSizeEnv))
	if rawBatchSize != "" {
		parsed, err := strconv.Atoi(rawBatchSize)
		if err != nil || parsed <= 0 {
			return reconcilerRuntimeConfig{}, &ConfigError{
				Code:    "CONFIG_RECONCILER_BATCH_SIZE_INVALID",
				Message: reconcilerBatchSizeEnv + " must be a positive integer",
			}
		}
		batchSize = parsed
	}

	leaseDuration := defaultReconcilerLeaseDuration
	rawLeaseSeconds := strings.TrimSpace(os.Getenv(reconcilerLeaseSecondsEnv))
	if rawLeaseSeconds != "" {
		seconds, err := strconv.Atoi(rawLeaseSeconds)
		if err != nil || seconds <= 0 {
			return reconcilerRuntimeConfig{}, &ConfigError{
				Code:    "CONFIG_RECONCILER_LEASE_SECONDS_INVALID",
				Message: reconcilerLeaseSecondsEnv + " must be a positive integer in seconds",
			}
		}
		leaseDuration = time.Duration(seconds) * time.Second
	}

	detectedBPS := defaultDetectedThresholdBPS
	rawDetectedBPS := strings.TrimSpace(os.Getenv(reconcilerDetectedThresholdBPSEnv))
	if rawDetectedBPS != "" {
		parsed, err := strconv.Atoi(rawDetectedBPS)
		if err != nil {
			return reconcilerRuntimeConfig{}, &ConfigError{
				Code:    "CONFIG_RECONCILER_DETECTED_THRESHOLD_BPS_INVALID",
				Message: reconcilerDetectedThresholdBPSEnv + " must be an integer",
			}
		}
		detectedBPS = parsed
	}

	confirmedBPS := defaultConfirmedThresholdBPS
	rawConfirmedBPS := strings.TrimSpace(os.Getenv(reconcilerConfirmedThresholdBPSEnv))
	if rawConfirmedBPS != "" {
		parsed, err := strconv.Atoi(rawConfirmedBPS)
		if err != nil {
			return reconcilerRuntimeConfig{}, &ConfigError{
				Code:    "CONFIG_RECONCILER_CONFIRMED_THRESHOLD_BPS_INVALID",
				Message: reconcilerConfirmedThresholdBPSEnv + " must be an integer",
			}
		}
		confirmedBPS = parsed
	}

	if confirmedBPS < 1 || confirmedBPS > 10000 {
		return reconcilerRuntimeConfig{}, &ConfigError{
			Code:    "CONFIG_RECONCILER_CONFIRMED_THRESHOLD_BPS_INVALID",
			Message: reconcilerConfirmedThresholdBPSEnv + " must be between 1 and 10000",
		}
	}
	if detectedBPS < 1 || detectedBPS > confirmedBPS {
		return reconcilerRuntimeConfig{}, &ConfigError{
			Code:    "CONFIG_RECONCILER_DETECTED_THRESHOLD_BPS_INVALID",
			Message: reconcilerDetectedThresholdBPSEnv + " must be between 1 and " + strconv.Itoa(confirmedBPS),
		}
	}

	btcMinConfirmations := defaultBTCMinConfirmations
	rawBTCMinConfirmations := strings.TrimSpace(os.Getenv(reconcilerBTCMinConfirmationsEnv))
	if rawBTCMinConfirmations != "" {
		parsed, err := strconv.Atoi(rawBTCMinConfirmations)
		if err != nil || parsed <= 0 {
			return reconcilerRuntimeConfig{}, &ConfigError{
				Code:    "CONFIG_RECONCILER_BTC_MIN_CONFIRMATIONS_INVALID",
				Message: reconcilerBTCMinConfirmationsEnv + " must be a positive integer",
			}
		}
		btcMinConfirmations = parsed
	}

	evmMinConfirmations := defaultEVMMinConfirmations
	rawEVMMinConfirmations := strings.TrimSpace(os.Getenv(reconcilerEVMMinConfirmationsEnv))
	if rawEVMMinConfirmations != "" {
		parsed, err := strconv.Atoi(rawEVMMinConfirmations)
		if err != nil || parsed <= 0 {
			return reconcilerRuntimeConfig{}, &ConfigError{
				Code:    "CONFIG_RECONCILER_EVM_MIN_CONFIRMATIONS_INVALID",
				Message: reconcilerEVMMinConfirmationsEnv + " must be a positive integer",
			}
		}
		evmMinConfirmations = parsed
	}

	return reconcilerRuntimeConfig{
		Enabled:             enabled,
		PollInterval:        pollInterval,
		BatchSize:           batchSize,
		LeaseDuration:       leaseDuration,
		DetectedBPS:         detectedBPS,
		ConfirmedBPS:        confirmedBPS,
		BTCMinConfirmations: btcMinConfirmations,
		EVMMinConfirmations: evmMinConfirmations,
	}, nil
}

func defaultReconcilerWorkerID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		hostname = "unknown"
	}
	return hostname + ":" + strconv.Itoa(os.Getpid())
}

func parseEVMRPCURLs(raw string) (map[string]string, *ConfigError) {
	if raw == "" {
		return map[string]string{}, nil
	}

	decoded := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, &ConfigError{
			Code:    "CONFIG_EVM_RPC_URLS_INVALID",
			Message: evmRPCURLsEnv + " must be a JSON object",
		}
	}

	out := map[string]string{}
	for network, rpcURL := range decoded {
		normalizedNetwork := strings.ToLower(strings.TrimSpace(network))
		normalizedRPCURL := strings.TrimSpace(rpcURL)
		if normalizedNetwork == "" || normalizedRPCURL == "" {
			continue
		}
		out[normalizedNetwork] = normalizedRPCURL
	}

	return out, nil
}

func resolveEnvelopeKeyMaterial(extendedPublicKey string, keyMaterial string, xpub string) string {
	resolved := strings.TrimSpace(extendedPublicKey)
	if resolved == "" {
		resolved = strings.TrimSpace(keyMaterial)
	}
	if resolved == "" {
		resolved = strings.TrimSpace(xpub)
	}
	return resolved
}
