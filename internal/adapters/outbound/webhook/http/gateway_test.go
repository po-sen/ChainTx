//go:build !integration

package http

import (
	"context"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chaintx/internal/application/dto"
)

func TestSendWebhookEventSuccess(t *testing.T) {
	const secret = "webhook-secret"
	payload := []byte(`{"event_id":"evt_1","event_type":"payment_request.status_changed"}`)

	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("X-ChainTx-Event-Id"); got != "evt_1" {
			t.Fatalf("expected event id header evt_1, got %s", got)
		}
		if got := r.Header.Get("X-ChainTx-Event-Type"); got != "payment_request.status_changed" {
			t.Fatalf("expected event type header, got %s", got)
		}
		timestamp := strings.TrimSpace(r.Header.Get("X-ChainTx-Timestamp"))
		if timestamp == "" {
			t.Fatalf("expected timestamp header")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		expectedSignature := BuildExpectedSignatureHeader(secret, timestamp, body)
		if got := r.Header.Get("X-ChainTx-Signature"); got != expectedSignature {
			t.Fatalf("expected signature %s, got %s", expectedSignature, got)
		}
		w.WriteHeader(nethttp.StatusNoContent)
	}))
	defer server.Close()

	gateway := NewGateway(Config{
		HMACSecret: secret,
	})
	output, appErr := gateway.SendWebhookEvent(context.Background(), dto.SendWebhookEventInput{
		EventID:        "evt_1",
		EventType:      "payment_request.status_changed",
		DestinationURL: server.URL,
		Payload:        payload,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if output.StatusCode != nethttp.StatusNoContent {
		t.Fatalf("expected status %d, got %d", nethttp.StatusNoContent, output.StatusCode)
	}
}

func TestSendWebhookEventNon2xxReturnsError(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(nethttp.StatusBadGateway)
		_, _ = w.Write([]byte("upstream unavailable"))
	}))
	defer server.Close()

	gateway := NewGateway(Config{
		HMACSecret: "webhook-secret",
	})
	output, appErr := gateway.SendWebhookEvent(context.Background(), dto.SendWebhookEventInput{
		EventID:        "evt_2",
		EventType:      "payment_request.status_changed",
		DestinationURL: server.URL,
		Payload:        []byte(`{"event_id":"evt_2"}`),
	})
	if appErr == nil {
		t.Fatalf("expected error")
	}
	if appErr.Code != "webhook_delivery_failed" {
		t.Fatalf("expected webhook_delivery_failed, got %s", appErr.Code)
	}
	if output.StatusCode != nethttp.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", nethttp.StatusBadGateway, output.StatusCode)
	}
}

func TestSendWebhookEventRequiresDestinationURL(t *testing.T) {
	gateway := NewGateway(Config{
		HMACSecret: "webhook-secret",
	})
	_, appErr := gateway.SendWebhookEvent(context.Background(), dto.SendWebhookEventInput{
		EventID:   "evt_3",
		EventType: "payment_request.status_changed",
		Payload:   []byte(`{"event_id":"evt_3"}`),
	})
	if appErr == nil {
		t.Fatalf("expected error")
	}
	if appErr.Code != "webhook_destination_missing" {
		t.Fatalf("expected webhook_destination_missing, got %s", appErr.Code)
	}
}
