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

func TestObservePaymentRequestBitcoinConfirmedUsesUTXOSettlements(t *testing.T) {
	const address = "bcrt1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/utxo"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"txid":  "tx-a",
					"vout":  0,
					"value": 600,
					"status": map[string]any{
						"confirmed":    true,
						"block_height": 101,
						"block_hash":   "hash-101",
					},
				},
				{
					"txid":  "tx-b",
					"vout":  1,
					"value": 400,
					"status": map[string]any{
						"confirmed":    true,
						"block_height": 102,
						"block_hash":   "hash-102",
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

	gateway := NewGateway(Config{BTCExploraBaseURL: server.URL})
	expected := "1000"
	output, appErr := gateway.ObservePaymentRequest(context.Background(), dto.ObservePaymentRequestInput{
		RequestID:           "pr_btc_split",
		Chain:               "bitcoin",
		Network:             "regtest",
		Asset:               "BTC",
		ExpectedAmountMinor: &expected,
		AddressCanonical:    address,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if !output.Supported || !output.Confirmed {
		t.Fatalf("expected supported+confirmed, got %+v", output)
	}
	if len(output.Settlements) != 2 {
		t.Fatalf("expected 2 settlement items, got %d", len(output.Settlements))
	}

	refs := map[string]bool{}
	for _, settlement := range output.Settlements {
		refs[settlement.EvidenceRef] = true
	}
	if !refs["tx-a:0"] || !refs["tx-b:1"] {
		t.Fatalf("expected tx-level evidence refs, got %+v", output.Settlements)
	}
}

func TestObservePaymentRequestBitcoinDepthGatedDetected(t *testing.T) {
	const address = "bcrt1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/utxo"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"txid":  "tx-c",
					"vout":  0,
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
		RequestID:           "pr_btc_depth",
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
	if len(output.Settlements) != 1 {
		t.Fatalf("expected one settlement item, got %d", len(output.Settlements))
	}
	if output.Settlements[0].Confirmations != 1 {
		t.Fatalf("expected 1 confirmation, got %d", output.Settlements[0].Confirmations)
	}
}

func TestObservePaymentRequestETHConfirmedUsesTransactionSettlements(t *testing.T) {
	recipient := "0x61ed32e69db70c5abab0522d80e8f5db215965de"
	txByBlock := map[string][]map[string]any{
		"0x3": {
			{
				"hash":        "0xaaa",
				"to":          recipient,
				"value":       "0x32",
				"blockNumber": "0x3",
				"blockHash":   "0xblock3",
			},
		},
		"0x4": {
			{
				"hash":        "0xbbb",
				"to":          recipient,
				"value":       "0x32",
				"blockNumber": "0x4",
				"blockHash":   "0xblock4",
			},
		},
	}
	receiptStatus := map[string]string{
		"0xaaa": "0x1",
		"0xbbb": "0x1",
	}

	server := newRPCServer(t, func(method string, params []json.RawMessage) any {
		switch method {
		case "eth_blockNumber":
			return "0x4"
		case "eth_getBlockByNumber":
			var blockTag string
			_ = json.Unmarshal(params[0], &blockTag)
			transactions := txByBlock[strings.ToLower(blockTag)]
			if transactions == nil {
				transactions = []map[string]any{}
			}
			return map[string]any{
				"number":       blockTag,
				"hash":         "0x" + strings.TrimPrefix(strings.ToLower(blockTag), "0x"),
				"transactions": transactions,
			}
		case "eth_getTransactionReceipt":
			var txHash string
			_ = json.Unmarshal(params[0], &txHash)
			status := receiptStatus[strings.ToLower(txHash)]
			if status == "" {
				status = "0x0"
			}
			return map[string]any{"status": status}
		default:
			t.Fatalf("unexpected method: %s", method)
			return nil
		}
	})
	defer server.Close()

	gateway := NewGateway(Config{EVMRPCURLs: map[string]string{"local": server.URL}})
	expected := "100"
	output, appErr := gateway.ObservePaymentRequest(context.Background(), dto.ObservePaymentRequestInput{
		RequestID:           "pr_eth_split",
		Chain:               "ethereum",
		Network:             "local",
		Asset:               "ETH",
		ExpectedAmountMinor: &expected,
		AddressCanonical:    recipient,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if !output.Supported || !output.Confirmed {
		t.Fatalf("expected supported+confirmed, got %+v", output)
	}
	if len(output.Settlements) != 2 {
		t.Fatalf("expected 2 settlement items, got %d", len(output.Settlements))
	}
	if output.Settlements[0].EvidenceRef == output.Settlements[1].EvidenceRef {
		t.Fatalf("expected distinct tx evidence refs, got %+v", output.Settlements)
	}
}

func TestObservePaymentRequestETHDetectedWhenMinConfirmationsNotMet(t *testing.T) {
	recipient := "0x61ed32e69db70c5abab0522d80e8f5db215965de"

	server := newRPCServer(t, func(method string, params []json.RawMessage) any {
		switch method {
		case "eth_blockNumber":
			return "0x5"
		case "eth_getBlockByNumber":
			var blockTag string
			_ = json.Unmarshal(params[0], &blockTag)
			transactions := []map[string]any{}
			if strings.EqualFold(blockTag, "0x5") {
				transactions = append(transactions, map[string]any{
					"hash":        "0xccc",
					"to":          recipient,
					"value":       "0x64",
					"blockNumber": "0x5",
					"blockHash":   "0xblock5",
				})
			}
			return map[string]any{
				"number":       blockTag,
				"hash":         "0x" + strings.TrimPrefix(strings.ToLower(blockTag), "0x"),
				"transactions": transactions,
			}
		case "eth_getTransactionReceipt":
			return map[string]any{"status": "0x1"}
		default:
			t.Fatalf("unexpected method: %s", method)
			return nil
		}
	})
	defer server.Close()

	gateway := NewGateway(Config{
		EVMRPCURLs: map[string]string{"local": server.URL},
		EVMMinConf: 2,
	})
	expected := "100"
	output, appErr := gateway.ObservePaymentRequest(context.Background(), dto.ObservePaymentRequestInput{
		RequestID:           "pr_eth_depth",
		Chain:               "ethereum",
		Network:             "local",
		Asset:               "ETH",
		ExpectedAmountMinor: &expected,
		AddressCanonical:    recipient,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if !output.Supported || !output.Detected || output.Confirmed {
		t.Fatalf("expected detected-only output, got %+v", output)
	}
	if len(output.Settlements) != 1 {
		t.Fatalf("expected one settlement item, got %d", len(output.Settlements))
	}
	if output.Settlements[0].Confirmations != 1 {
		t.Fatalf("expected one confirmation, got %d", output.Settlements[0].Confirmations)
	}
}

func TestObservePaymentRequestERC20ConfirmedUsesLogSettlements(t *testing.T) {
	recipient := "0x61ed32e69db70c5abab0522d80e8f5db215965de"
	tokenContract := "0x1234567890abcdef1234567890abcdef12345678"

	server := newRPCServer(t, func(method string, _ []json.RawMessage) any {
		switch method {
		case "eth_blockNumber":
			return "0x64"
		case "eth_getLogs":
			return []map[string]any{
				{
					"removed":         false,
					"transactionHash": "0xlogtx1",
					"logIndex":        "0x0",
					"blockNumber":     "0x63",
					"blockHash":       "0xblock63",
					"data":            "0x0000000000000000000000000000000000000000000000000000000000000032",
				},
				{
					"removed":         false,
					"transactionHash": "0xlogtx2",
					"logIndex":        "0x1",
					"blockNumber":     "0x64",
					"blockHash":       "0xblock64",
					"data":            "0x0000000000000000000000000000000000000000000000000000000000000032",
				},
			}
		default:
			t.Fatalf("unexpected method: %s", method)
			return nil
		}
	})
	defer server.Close()

	gateway := NewGateway(Config{EVMRPCURLs: map[string]string{"local": server.URL}})
	expected := "100"
	output, appErr := gateway.ObservePaymentRequest(context.Background(), dto.ObservePaymentRequestInput{
		RequestID:           "pr_usdt_split",
		Chain:               "ethereum",
		Network:             "local",
		Asset:               "USDT",
		ExpectedAmountMinor: &expected,
		AddressCanonical:    recipient,
		TokenContract:       &tokenContract,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if !output.Supported || !output.Confirmed {
		t.Fatalf("expected supported+confirmed, got %+v", output)
	}
	if len(output.Settlements) != 2 {
		t.Fatalf("expected 2 settlement items, got %d", len(output.Settlements))
	}

	refs := map[string]bool{}
	for _, settlement := range output.Settlements {
		refs[settlement.EvidenceRef] = true
	}
	if !refs["0xlogtx1:0"] || !refs["0xlogtx2:1"] {
		t.Fatalf("expected log-level evidence refs, got %+v", output.Settlements)
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

type rpcHandlerFunc func(method string, params []json.RawMessage) any

func newRPCServer(t *testing.T, handler rpcHandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}{
			Params: []json.RawMessage{},
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		result := handler(request.Method, request.Params)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  result,
		})
	}))
}
