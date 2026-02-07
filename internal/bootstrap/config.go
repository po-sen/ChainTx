package bootstrap

import (
	"net/url"
	"os"
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
)

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

	return Config{
		Port:                     port,
		OpenAPISpecPath:          openAPISpecPath,
		ShutdownTimeout:          defaultShutdownTimeout,
		DatabaseURL:              databaseURL,
		DatabaseTarget:           databaseTarget,
		DBReadinessTimeout:       defaultDBReadinessTimeout,
		DBReadinessRetryInterval: defaultDBReadinessRetryInterval,
		MigrationsPath:           defaultMigrationsPath,
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
