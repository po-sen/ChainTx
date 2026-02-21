//go:build !integration

package valueobjects

import "testing"

func TestNormalizeWebhookURLValid(t *testing.T) {
	canonical, host, appErr := NormalizeWebhookURL("HTTPS://Webhook.Example.com:8443/path?a=1")
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if canonical != "https://webhook.example.com:8443/path?a=1" {
		t.Fatalf("unexpected canonical url: %s", canonical)
	}
	if host != "webhook.example.com" {
		t.Fatalf("unexpected host: %s", host)
	}
}

func TestNormalizeWebhookURLRejectsInvalidInput(t *testing.T) {
	testCases := []string{
		"",
		"not-a-url",
		"ftp://example.com/hook",
		"https://user:pass@example.com/hook",
	}

	for _, testCase := range testCases {
		_, _, appErr := NormalizeWebhookURL(testCase)
		if appErr == nil {
			t.Fatalf("expected validation error for %q", testCase)
		}
		if appErr.Code != "invalid_request" {
			t.Fatalf("expected invalid_request for %q, got %s", testCase, appErr.Code)
		}
	}
}

func TestNormalizeWebhookHostPattern(t *testing.T) {
	pattern, appErr := NormalizeWebhookHostPattern(" *.Example.com ")
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if pattern != "*.example.com" {
		t.Fatalf("unexpected pattern: %s", pattern)
	}

	_, appErr = NormalizeWebhookHostPattern("https://example.com")
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
}

func TestIsWebhookHostAllowed(t *testing.T) {
	allowlist := []string{"hooks.example.com", "*.partner.example"}

	if !IsWebhookHostAllowed("hooks.example.com", allowlist) {
		t.Fatalf("expected exact host match")
	}
	if !IsWebhookHostAllowed("a.partner.example", allowlist) {
		t.Fatalf("expected wildcard host match")
	}
	if IsWebhookHostAllowed("partner.example", allowlist) {
		t.Fatalf("did not expect wildcard to match apex domain")
	}
	if IsWebhookHostAllowed("evil.example", allowlist) {
		t.Fatalf("did not expect non-allowlisted host to match")
	}
}
