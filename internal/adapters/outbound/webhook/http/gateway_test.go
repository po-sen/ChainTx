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
		if got := r.Header.Get("Idempotency-Key"); got != "evt_1" {
			t.Fatalf("expected idempotency key evt_1, got %s", got)
		}
		if got := r.Header.Get("X-ChainTx-Event-Type"); got != "payment_request.status_changed" {
			t.Fatalf("expected event type header, got %s", got)
		}
		if got := r.Header.Get("X-ChainTx-Delivery-Attempt"); got != "3" {
			t.Fatalf("expected attempt header 3, got %s", got)
		}
		timestamp := strings.TrimSpace(r.Header.Get("X-ChainTx-Timestamp"))
		if timestamp == "" {
			t.Fatalf("expected timestamp header")
		}
		nonce := strings.TrimSpace(r.Header.Get("X-ChainTx-Nonce"))
		if nonce == "" {
			t.Fatalf("expected nonce header")
		}
		if got := strings.TrimSpace(r.Header.Get("X-ChainTx-Signature-Version")); got != "v1" {
			t.Fatalf("expected signature version v1, got %s", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		expectedLegacySignature := BuildExpectedSignatureHeader(secret, timestamp, body)
		if got := r.Header.Get("X-ChainTx-Signature"); got != expectedLegacySignature {
			t.Fatalf("expected legacy signature %s, got %s", expectedLegacySignature, got)
		}
		expectedV1Signature := BuildExpectedSignatureV1Header(
			secret,
			timestamp,
			nonce,
			"evt_1",
			"payment_request.status_changed",
			body,
		)
		if got := r.Header.Get("X-ChainTx-Signature-V1"); got != expectedV1Signature {
			t.Fatalf("expected signature v1 %s, got %s", expectedV1Signature, got)
		}
		w.WriteHeader(nethttp.StatusNoContent)
	}))
	defer server.Close()

	gateway := NewGateway(Config{
		HMACSecret: secret,
	})
	output, appErr := gateway.SendWebhookEvent(context.Background(), dto.SendWebhookEventInput{
		EventID:         "evt_1",
		EventType:       "payment_request.status_changed",
		DeliveryAttempt: 3,
		DestinationURL:  server.URL,
		Payload:         payload,
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

func TestSendWebhookEventUsesDefaultAttemptHeader(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if got := r.Header.Get("X-ChainTx-Delivery-Attempt"); got != "1" {
			t.Fatalf("expected default attempt header 1, got %s", got)
		}
		w.WriteHeader(nethttp.StatusNoContent)
	}))
	defer server.Close()

	gateway := NewGateway(Config{
		HMACSecret: "webhook-secret",
	})
	_, appErr := gateway.SendWebhookEvent(context.Background(), dto.SendWebhookEventInput{
		EventID:        "evt_4",
		EventType:      "payment_request.status_changed",
		DestinationURL: server.URL,
		Payload:        []byte(`{"event_id":"evt_4"}`),
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
}

func TestSendWebhookEventRequiresEventID(t *testing.T) {
	gateway := NewGateway(Config{
		HMACSecret: "webhook-secret",
	})
	_, appErr := gateway.SendWebhookEvent(context.Background(), dto.SendWebhookEventInput{
		EventType:      "payment_request.status_changed",
		DestinationURL: "https://hooks.example.com/evt",
		Payload:        []byte(`{"event_id":"evt"}`),
	})
	if appErr == nil {
		t.Fatalf("expected error")
	}
	if appErr.Code != "webhook_event_id_missing" {
		t.Fatalf("expected webhook_event_id_missing, got %s", appErr.Code)
	}
}

func TestSendWebhookEventRequiresEventType(t *testing.T) {
	gateway := NewGateway(Config{
		HMACSecret: "webhook-secret",
	})
	_, appErr := gateway.SendWebhookEvent(context.Background(), dto.SendWebhookEventInput{
		EventID:        "evt",
		DestinationURL: "https://hooks.example.com/evt",
		Payload:        []byte(`{"event_id":"evt"}`),
	})
	if appErr == nil {
		t.Fatalf("expected error")
	}
	if appErr.Code != "webhook_event_type_missing" {
		t.Fatalf("expected webhook_event_type_missing, got %s", appErr.Code)
	}
}
