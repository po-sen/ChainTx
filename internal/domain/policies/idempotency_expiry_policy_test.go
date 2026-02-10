package policies

import (
	"testing"
	"time"
)

func TestResolveIdempotencyExpiryUsesBaseline(t *testing.T) {
	createdAt := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	requestExpiresAt := createdAt.Add(1 * time.Hour)

	resolved := ResolveIdempotencyExpiry(createdAt, requestExpiresAt)
	if resolved.Sub(createdAt) != 7*24*time.Hour {
		t.Fatalf("expected 7d baseline, got %s", resolved.Sub(createdAt))
	}
}

func TestResolveIdempotencyExpiryUsesRequestExpiryWhenLonger(t *testing.T) {
	createdAt := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	requestExpiresAt := createdAt.Add(10 * 24 * time.Hour)

	resolved := ResolveIdempotencyExpiry(createdAt, requestExpiresAt)
	if !resolved.Equal(requestExpiresAt) {
		t.Fatalf("expected request expiry %s, got %s", requestExpiresAt, resolved)
	}
}
