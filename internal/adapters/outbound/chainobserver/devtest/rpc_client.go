package devtest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	apperrors "chaintx/internal/shared_kernel/errors"
)

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonRPCClient struct {
	httpClient  *http.Client
	httpTimeout time.Duration
}

func newJSONRPCClient(httpClient *http.Client, httpTimeout time.Duration) *jsonRPCClient {
	return &jsonRPCClient{
		httpClient:  httpClient,
		httpTimeout: httpTimeout,
	}
}

func (c *jsonRPCClient) Call(
	ctx context.Context,
	rpcURL string,
	method string,
	params any,
) (json.RawMessage, *apperrors.AppError) {
	payload := rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to encode rpc request",
			map[string]any{"error": err.Error(), "method": method},
		)
	}

	requestCtx, cancel := context.WithTimeout(ctx, c.httpTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, rpcURL, bytes.NewReader(encoded))
	if err != nil {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to build rpc request",
			map[string]any{"error": err.Error(), "method": method},
		)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to call rpc endpoint",
			map[string]any{"error": err.Error(), "method": method},
		)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"rpc endpoint returned non-200 status",
			map[string]any{"status_code": response.StatusCode, "method": method},
		)
	}

	rpcResp := rpcResponse{}
	if err := json.NewDecoder(response.Body).Decode(&rpcResp); err != nil {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to decode rpc response",
			map[string]any{"error": err.Error(), "method": method},
		)
	}
	if rpcResp.Error != nil {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"rpc endpoint returned error",
			map[string]any{
				"method":    method,
				"rpc_error": rpcResp.Error.Message,
				"rpc_code":  rpcResp.Error.Code,
			},
		)
	}

	return rpcResp.Result, nil
}
