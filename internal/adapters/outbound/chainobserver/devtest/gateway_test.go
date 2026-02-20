//go:build !integration

package devtest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chaintx/internal/application/dto"
)

func TestObservePaymentRequestBitcoinConfirmed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"chain_stats":   map[string]any{"funded_txo_sum": 50000},
			"mempool_stats": map[string]any{"funded_txo_sum": 0},
		})
	}))
	defer server.Close()

	gateway := NewGateway(Config{BTCExploraBaseURL: server.URL})
	expected := "40000"
	output, appErr := gateway.ObservePaymentRequest(context.Background(), dto.ObservePaymentRequestInput{
		RequestID:           "pr1",
		Chain:               "bitcoin",
		Network:             "regtest",
		Asset:               "BTC",
		ExpectedAmountMinor: &expected,
		AddressCanonical:    "bcrt1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq",
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if !output.Supported || !output.Confirmed {
		t.Fatalf("expected supported+confirmed, got %+v", output)
	}
}

func TestObservePaymentRequestBitcoinDetectedFromMempool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"chain_stats":   map[string]any{"funded_txo_sum": 1000},
			"mempool_stats": map[string]any{"funded_txo_sum": 1000},
		})
	}))
	defer server.Close()

	gateway := NewGateway(Config{BTCExploraBaseURL: server.URL})
	expected := "1500"
	output, appErr := gateway.ObservePaymentRequest(context.Background(), dto.ObservePaymentRequestInput{
		RequestID:           "pr1",
		Chain:               "bitcoin",
		Network:             "regtest",
		Asset:               "BTC",
		ExpectedAmountMinor: &expected,
		AddressCanonical:    "bcrt1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq",
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if !output.Supported || !output.Detected || output.Confirmed {
		t.Fatalf("expected supported+detected only, got %+v", output)
	}
}

func TestObservePaymentRequestBitcoinThresholdsAreConfigurable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"chain_stats":   map[string]any{"funded_txo_sum": 900},
			"mempool_stats": map[string]any{"funded_txo_sum": 0},
		})
	}))
	defer server.Close()

	gateway := NewGateway(Config{
		BTCExploraBaseURL: server.URL,
		DetectedBPS:       8000,
		ConfirmedBPS:      9000,
	})
	expected := "1000"
	output, appErr := gateway.ObservePaymentRequest(context.Background(), dto.ObservePaymentRequestInput{
		RequestID:           "pr1",
		Chain:               "bitcoin",
		Network:             "regtest",
		Asset:               "BTC",
		ExpectedAmountMinor: &expected,
		AddressCanonical:    "bcrt1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq",
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if !output.Supported || output.Detected || !output.Confirmed {
		t.Fatalf("expected supported+confirmed with custom threshold, got %+v", output)
	}
}

func TestObservePaymentRequestEVMConfirmed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}
		if payload["method"] != "eth_getBalance" {
			t.Fatalf("expected eth_getBalance, got %v", payload["method"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0x64",
		})
	}))
	defer server.Close()

	gateway := NewGateway(Config{EVMRPCURLs: map[string]string{"local": server.URL}})
	expected := "100"
	output, appErr := gateway.ObservePaymentRequest(context.Background(), dto.ObservePaymentRequestInput{
		RequestID:           "pr1",
		Chain:               "ethereum",
		Network:             "local",
		Asset:               "ETH",
		ExpectedAmountMinor: &expected,
		AddressCanonical:    "0x61ed32e69db70c5abab0522d80e8f5db215965de",
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if !output.Supported || !output.Confirmed {
		t.Fatalf("expected supported+confirmed, got %+v", output)
	}
}

func TestObservePaymentRequestEVMDetectedWhenBelowConfirmedThreshold(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0x50",
		})
	}))
	defer server.Close()

	gateway := NewGateway(Config{
		EVMRPCURLs:   map[string]string{"local": server.URL},
		DetectedBPS:  8000,
		ConfirmedBPS: 10000,
	})
	expected := "100"
	output, appErr := gateway.ObservePaymentRequest(context.Background(), dto.ObservePaymentRequestInput{
		RequestID:           "pr1",
		Chain:               "ethereum",
		Network:             "local",
		Asset:               "ETH",
		ExpectedAmountMinor: &expected,
		AddressCanonical:    "0x61ed32e69db70c5abab0522d80e8f5db215965de",
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if !output.Supported || !output.Detected || output.Confirmed {
		t.Fatalf("expected supported+detected only, got %+v", output)
	}
}

func TestObservePaymentRequestUnsupportedWhenMissingEndpoint(t *testing.T) {
	gateway := NewGateway(Config{})
	output, appErr := gateway.ObservePaymentRequest(context.Background(), dto.ObservePaymentRequestInput{
		RequestID:        "pr1",
		Chain:            "ethereum",
		Network:          "local",
		Asset:            "ETH",
		AddressCanonical: "0x61ed32e69db70c5abab0522d80e8f5db215965de",
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if output.Supported {
		t.Fatalf("expected unsupported output")
	}
}
