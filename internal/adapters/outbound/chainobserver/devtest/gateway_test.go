//go:build !integration

package devtest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestObservePaymentRequestBitcoinConfirmedRespectsMinConfirmations(t *testing.T) {
	const address = "bcrt1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/utxo"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"value": 1000,
					"status": map[string]any{
						"confirmed":    true,
						"block_height": 101,
					},
				},
			})
		case r.URL.Path == "/blocks/tip/height":
			_, _ = w.Write([]byte("101"))
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"chain_stats":   map[string]any{"funded_txo_sum": 1000},
				"mempool_stats": map[string]any{"funded_txo_sum": 0},
			})
		}
	}))
	defer server.Close()

	gateway := NewGateway(Config{
		BTCExploraBaseURL: server.URL,
		BTCMinConf:        2,
	})
	expected := "1000"
	output, appErr := gateway.ObservePaymentRequest(context.Background(), dto.ObservePaymentRequestInput{
		RequestID:           "pr1",
		Chain:               "bitcoin",
		Network:             "regtest",
		Asset:               "BTC",
		ExpectedAmountMinor: &expected,
		AddressCanonical:    address,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if !output.Supported || !output.Detected || output.Confirmed {
		t.Fatalf("expected depth-gated detected only, got %+v", output)
	}
}

func TestObservePaymentRequestBitcoinConfirmedAfterMinConfirmationsReached(t *testing.T) {
	const address = "bcrt1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/utxo"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"value": 1000,
					"status": map[string]any{
						"confirmed":    true,
						"block_height": 101,
					},
				},
			})
		case r.URL.Path == "/blocks/tip/height":
			_, _ = w.Write([]byte("102"))
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"chain_stats":   map[string]any{"funded_txo_sum": 1000},
				"mempool_stats": map[string]any{"funded_txo_sum": 0},
			})
		}
	}))
	defer server.Close()

	gateway := NewGateway(Config{
		BTCExploraBaseURL: server.URL,
		BTCMinConf:        2,
	})
	expected := "1000"
	output, appErr := gateway.ObservePaymentRequest(context.Background(), dto.ObservePaymentRequestInput{
		RequestID:           "pr1",
		Chain:               "bitcoin",
		Network:             "regtest",
		Asset:               "BTC",
		ExpectedAmountMinor: &expected,
		AddressCanonical:    address,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if !output.Supported || output.Detected || !output.Confirmed {
		t.Fatalf("expected confirmed after depth reached, got %+v", output)
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

func TestObservePaymentRequestEVMConfirmedRespectsMinConfirmations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}

		method := payload["method"]
		switch method {
		case "eth_blockNumber":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  "0x64",
			})
		case "eth_getBalance":
			params, ok := payload["params"].([]any)
			if !ok || len(params) != 2 {
				t.Fatalf("unexpected params: %+v", payload["params"])
			}
			blockTag, _ := params[1].(string)
			result := "0x64"
			if blockTag == "0x62" {
				result = "0x4f"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  result,
			})
		default:
			t.Fatalf("unexpected method: %v", method)
		}
	}))
	defer server.Close()

	gateway := NewGateway(Config{
		EVMRPCURLs:   map[string]string{"local": server.URL},
		DetectedBPS:  8000,
		ConfirmedBPS: 10000,
		EVMMinConf:   3,
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
		t.Fatalf("expected detected only when depth not reached, got %+v", output)
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
