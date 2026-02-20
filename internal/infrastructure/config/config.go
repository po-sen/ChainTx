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
)

const addressSchemeAllowListEnv = "PAYMENT_REQUEST_ADDRESS_SCHEME_ALLOW_LIST_JSON"
const devtestKeysetsEnv = "PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON"
const keysetHashHMACSecretEnv = "PAYMENT_REQUEST_KEYSET_HASH_HMAC_SECRET"

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
	KeysetHashAlgorithm      string
	KeysetHashHMACSecret     string
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
	devtestKeysets, keysetErr := parseDevtestKeysets(rawDevtestKeysets)
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
		KeysetHashAlgorithm:      defaultKeysetHashAlgo,
		KeysetHashHMACSecret:     keysetHashHMACSecret,
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
}

func parseDevtestKeysets(raw string) (map[string]string, *ConfigError) {
	if raw == "" {
		return map[string]string{}, nil
	}

	entries := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, &ConfigError{
			Code:    "CONFIG_DEVTEST_KEYSETS_INVALID",
			Message: devtestKeysetsEnv + " must be a JSON object",
		}
	}

	if parsedFlat, isFlat, cfgErr := parseFlatDevtestKeysets(entries); cfgErr != nil {
		return nil, cfgErr
	} else if isFlat {
		return parsedFlat, nil
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

func parseNestedDevtestKeysets(entries map[string]json.RawMessage) (map[string]string, *ConfigError) {
	keysets := map[string]string{}

	for chain, chainPayload := range entries {
		normalizedChain := strings.ToLower(strings.TrimSpace(chain))
		if normalizedChain == "" {
			continue
		}

		networkEntries := map[string]json.RawMessage{}
		if err := json.Unmarshal(chainPayload, &networkEntries); err != nil {
			return nil, &ConfigError{
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
				return nil, &ConfigError{
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
				return nil, &ConfigError{
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
				return nil, &ConfigError{
					Code:    "CONFIG_DEVTEST_KEYSETS_INVALID",
					Message: devtestKeysetsEnv + " nested entry is missing extended_public_key",
					Metadata: map[string]string{
						"chain":     normalizedChain,
						"network":   normalizedNetwork,
						"keyset_id": keysetID,
					},
				}
			}

			if existing, exists := keysets[keysetID]; exists && existing != keyMaterial {
				return nil, &ConfigError{
					Code:    "CONFIG_DEVTEST_KEYSETS_INVALID",
					Message: devtestKeysetsEnv + " defines conflicting key material for keyset_id",
					Metadata: map[string]string{
						"keyset_id": keysetID,
					},
				}
			}
			keysets[keysetID] = keyMaterial
		}
	}

	return keysets, nil
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
