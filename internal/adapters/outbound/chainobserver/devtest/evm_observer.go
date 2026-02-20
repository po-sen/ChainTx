package devtest

import (
	"context"
	"math/big"
	"strings"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type evmObserver struct {
	rpcURLs    map[string]string
	rpcClient  *jsonRPCClient
	thresholds thresholdPolicy
}

func newEVMObserver(
	rawRPCURLs map[string]string,
	rpcClient *jsonRPCClient,
	thresholds thresholdPolicy,
) *evmObserver {
	normalized := map[string]string{}
	for network, rawURL := range rawRPCURLs {
		normalizedNetwork := strings.ToLower(strings.TrimSpace(network))
		normalizedURL := strings.TrimSpace(rawURL)
		if normalizedNetwork == "" || normalizedURL == "" {
			continue
		}
		normalized[normalizedNetwork] = normalizedURL
	}

	return &evmObserver{
		rpcURLs:    normalized,
		rpcClient:  rpcClient,
		thresholds: thresholds,
	}
}

func (o *evmObserver) Observe(
	ctx context.Context,
	input dto.ObservePaymentRequestInput,
	expected *big.Int,
) (dto.ObservePaymentRequestOutput, *apperrors.AppError) {
	if o == nil || o.rpcClient == nil {
		return dto.ObservePaymentRequestOutput{Supported: false}, nil
	}

	network := strings.ToLower(strings.TrimSpace(input.Network))
	rpcURL, exists := o.rpcURLs[network]
	if !exists || strings.TrimSpace(rpcURL) == "" {
		return dto.ObservePaymentRequestOutput{Supported: false}, nil
	}

	asset := strings.ToUpper(strings.TrimSpace(input.Asset))
	normalizedAddress := normalizeHexAddress(input.AddressCanonical)

	var (
		amount *big.Int
		appErr *apperrors.AppError
	)

	if asset == "ETH" {
		amount, appErr = o.ethGetBalance(ctx, rpcURL, normalizedAddress)
	} else {
		if input.TokenContract == nil || strings.TrimSpace(*input.TokenContract) == "" {
			return dto.ObservePaymentRequestOutput{Supported: false}, nil
		}
		amount, appErr = o.erc20BalanceOf(ctx, rpcURL, strings.TrimSpace(*input.TokenContract), normalizedAddress)
	}
	if appErr != nil {
		return dto.ObservePaymentRequestOutput{}, appErr
	}

	confirmedRequired := o.thresholds.confirmedRequired(expected)
	detectedRequired := o.thresholds.detectedRequired(expected)
	confirmed := amount.Cmp(confirmedRequired) >= 0
	detected := !confirmed && amount.Cmp(detectedRequired) >= 0

	return dto.ObservePaymentRequestOutput{
		Supported:         true,
		ObservedAmount:    amount.String(),
		Detected:          detected,
		Confirmed:         confirmed,
		ObservationSource: "evm_rpc",
		ObservationDetails: map[string]any{
			"network":                  network,
			"asset":                    asset,
			"detected_threshold_bps":   o.thresholds.detectedBPS,
			"confirmed_threshold_bps":  o.thresholds.confirmedBPS,
			"detected_required_minor":  detectedRequired.String(),
			"confirmed_required_minor": confirmedRequired.String(),
		},
	}, nil
}

func (o *evmObserver) ethGetBalance(
	ctx context.Context,
	rpcURL string,
	address string,
) (*big.Int, *apperrors.AppError) {
	result, appErr := o.rpcClient.Call(ctx, rpcURL, "eth_getBalance", []any{address, "latest"})
	if appErr != nil {
		return nil, appErr
	}
	return parseHexQuantity(result)
}

func (o *evmObserver) erc20BalanceOf(
	ctx context.Context,
	rpcURL string,
	contract string,
	address string,
) (*big.Int, *apperrors.AppError) {
	callData, appErr := buildERC20BalanceOfData(address)
	if appErr != nil {
		return nil, appErr
	}

	result, rpcErr := o.rpcClient.Call(ctx, rpcURL, "eth_call", []any{
		map[string]any{
			"to":   normalizeHexAddress(contract),
			"data": callData,
		},
		"latest",
	})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return parseHexQuantity(result)
}
