package bootstrap

import (
	"os"
	"time"
)

const (
	defaultPort            = "8080"
	defaultOpenAPISpec     = "api/openapi.yaml"
	defaultShutdownTimeout = 10 * time.Second
)

type Config struct {
	Port            string
	OpenAPISpecPath string
	ShutdownTimeout time.Duration
}

func LoadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	openAPISpecPath := os.Getenv("OPENAPI_SPEC_PATH")
	if openAPISpecPath == "" {
		openAPISpecPath = defaultOpenAPISpec
	}

	return Config{
		Port:            port,
		OpenAPISpecPath: openAPISpecPath,
		ShutdownTimeout: defaultShutdownTimeout,
	}
}

func (c Config) Address() string {
	return ":" + c.Port
}
