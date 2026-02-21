package devtest

import (
	"context"
	"math/big"
	"strings"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type evmObserver struct {
	rpcURLs       map[string]string
	rpcClient     *jsonRPCClient
	thresholds    thresholdPolicy
	confirmations confirmationPolicy
}

func newEVMObserver(
	rawRPCURLs map[string]string,
	rpcClient *jsonRPCClient,
	thresholds thresholdPolicy,
	confirmations confirmationPolicy,
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
		rpcURLs:       normalized,
		rpcClient:     rpcClient,
		thresholds:    thresholds,
		confirmations: confirmations,
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
		latestAmount    *big.Int
		confirmedAmount *big.Int
		appErr          *apperrors.AppError
	)

	confirmedBlockTag := "latest"
	if o.confirmations.evmMin > 1 {
		confirmedBlockTag, appErr = o.confirmedBlockTag(ctx, rpcURL)
		if appErr != nil {
			return dto.ObservePaymentRequestOutput{}, appErr
		}
	}

	if asset == "ETH" {
		latestAmount, appErr = o.ethGetBalance(ctx, rpcURL, normalizedAddress, "latest")
	} else {
		if input.TokenContract == nil || strings.TrimSpace(*input.TokenContract) == "" {
			return dto.ObservePaymentRequestOutput{Supported: false}, nil
		}
		latestAmount, appErr = o.erc20BalanceOf(
			ctx,
			rpcURL,
			strings.TrimSpace(*input.TokenContract),
			normalizedAddress,
			"latest",
		)
	}
	if appErr != nil {
		return dto.ObservePaymentRequestOutput{}, appErr
	}

	if confirmedBlockTag == "latest" {
		confirmedAmount = latestAmount
	} else if asset == "ETH" {
		confirmedAmount, appErr = o.ethGetBalance(ctx, rpcURL, normalizedAddress, confirmedBlockTag)
		if appErr != nil {
			return dto.ObservePaymentRequestOutput{}, appErr
		}
	} else {
		confirmedAmount, appErr = o.erc20BalanceOf(
			ctx,
			rpcURL,
			strings.TrimSpace(*input.TokenContract),
			normalizedAddress,
			confirmedBlockTag,
		)
		if appErr != nil {
			return dto.ObservePaymentRequestOutput{}, appErr
		}
	}

	confirmedRequired := o.thresholds.confirmedRequired(expected)
	detectedRequired := o.thresholds.detectedRequired(expected)
	confirmed := confirmedAmount.Cmp(confirmedRequired) >= 0
	detected := !confirmed && latestAmount.Cmp(detectedRequired) >= 0

	return dto.ObservePaymentRequestOutput{
		Supported:         true,
		ObservedAmount:    latestAmount.String(),
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
			"evm_min_confirmations":    o.confirmations.evmMin,
			"confirmed_block_tag":      confirmedBlockTag,
			"latest_amount_minor":      latestAmount.String(),
			"confirmed_amount_minor":   confirmedAmount.String(),
		},
	}, nil
}

func (o *evmObserver) ethGetBalance(
	ctx context.Context,
	rpcURL string,
	address string,
	blockTag string,
) (*big.Int, *apperrors.AppError) {
	result, appErr := o.rpcClient.Call(ctx, rpcURL, "eth_getBalance", []any{address, blockTag})
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
	blockTag string,
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
		blockTag,
	})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return parseHexQuantity(result)
}

func (o *evmObserver) confirmedBlockTag(
	ctx context.Context,
	rpcURL string,
) (string, *apperrors.AppError) {
	result, appErr := o.rpcClient.Call(ctx, rpcURL, "eth_blockNumber", []any{})
	if appErr != nil {
		return "", appErr
	}
	latestBlock, parseErr := parseHexQuantity(result)
	if parseErr != nil {
		return "", parseErr
	}

	minus := big.NewInt(int64(o.confirmations.evmMin - 1))
	target := new(big.Int).Sub(latestBlock, minus)
	if target.Sign() < 0 {
		target = big.NewInt(0)
	}
	return "0x" + target.Text(16), nil
}
