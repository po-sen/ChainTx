package bootstrap

import "testing"

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://chaintx:chaintx@localhost:5432/chaintx?sslmode=disable")
	t.Setenv("PORT", "")
	t.Setenv("OPENAPI_SPEC_PATH", "")

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
}

func TestLoadConfig_RequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}

	if cfgErr.Code != "CONFIG_DATABASE_URL_REQUIRED" {
		t.Fatalf("expected CONFIG_DATABASE_URL_REQUIRED, got %s", cfgErr.Code)
	}
}

func TestLoadConfig_RejectsInvalidScheme(t *testing.T) {
	t.Setenv("DATABASE_URL", "mysql://localhost:3306/chaintx")

	_, cfgErr := LoadConfig()
	if cfgErr == nil {
		t.Fatalf("expected error")
	}

	if cfgErr.Code != "CONFIG_DATABASE_URL_SCHEME_INVALID" {
		t.Fatalf("expected CONFIG_DATABASE_URL_SCHEME_INVALID, got %s", cfgErr.Code)
	}
}
