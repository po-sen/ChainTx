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
)

const addressSchemeAllowListEnv = "PAYMENT_REQUEST_ADDRESS_SCHEME_ALLOW_LIST_JSON"

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

	devtestKeysets := map[string]string{}
	rawDevtestKeysets := strings.TrimSpace(os.Getenv("PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON"))
	if rawDevtestKeysets != "" {
		if err := json.Unmarshal([]byte(rawDevtestKeysets), &devtestKeysets); err != nil {
			return Config{}, &ConfigError{
				Code:    "CONFIG_DEVTEST_KEYSETS_INVALID",
				Message: "PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON must be a JSON object",
			}
		}
	}
	if allocationMode == "devtest" && len(devtestKeysets) == 0 {
		return Config{}, &ConfigError{
			Code:    "CONFIG_DEVTEST_KEYSETS_REQUIRED",
			Message: "PAYMENT_REQUEST_DEVTEST_KEYSETS_JSON is required for devtest allocation mode",
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
