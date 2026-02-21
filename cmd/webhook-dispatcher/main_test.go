package main

import (
	"testing"

	"chaintx/internal/infrastructure/config"
)

func TestValidateWebhookDispatcherConfig(t *testing.T) {
	testCases := []struct {
		name         string
		cfg          config.Config
		expectedCode string
	}{
		{
			name: "disabled",
			cfg: config.Config{
				WebhookEnabled: false,
			},
			expectedCode: "CONFIG_WEBHOOK_DISABLED",
		},
		{
			name: "missing hmac",
			cfg: config.Config{
				WebhookEnabled: true,
			},
			expectedCode: "CONFIG_WEBHOOK_HMAC_SECRET_REQUIRED",
		},
		{
			name: "valid",
			cfg: config.Config{
				WebhookEnabled:    true,
				WebhookHMACSecret: "secret",
			},
			expectedCode: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateWebhookDispatcherConfig(tc.cfg)
			if tc.expectedCode == "" {
				if err != nil {
					t.Fatalf("expected nil error, got %+v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error code %s", tc.expectedCode)
			}
			if err.Code != tc.expectedCode {
				t.Fatalf("expected error code %s, got %s", tc.expectedCode, err.Code)
			}
		})
	}
}
