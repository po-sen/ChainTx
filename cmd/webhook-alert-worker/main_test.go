package main

import (
	"testing"

	"chaintx/internal/infrastructure/config"
)

func TestValidateWebhookAlertWorkerConfig(t *testing.T) {
	testCases := []struct {
		name         string
		cfg          config.Config
		expectedCode string
	}{
		{
			name: "disabled",
			cfg: config.Config{
				WebhookAlertEnabled: false,
			},
			expectedCode: "CONFIG_WEBHOOK_ALERT_DISABLED",
		},
		{
			name: "valid",
			cfg: config.Config{
				WebhookAlertEnabled: true,
			},
			expectedCode: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateWebhookAlertWorkerConfig(tc.cfg)
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
