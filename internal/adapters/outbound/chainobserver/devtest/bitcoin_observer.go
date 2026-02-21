package devtest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type bitcoinObserver struct {
	baseURL       string
	httpClient    *http.Client
	httpTimeout   time.Duration
	thresholds    thresholdPolicy
	confirmations confirmationPolicy
}

type esploraAddressStats struct {
	ChainStats struct {
		FundedTXOSum int64 `json:"funded_txo_sum"`
	} `json:"chain_stats"`
	MempoolStats struct {
		FundedTXOSum int64 `json:"funded_txo_sum"`
	} `json:"mempool_stats"`
}

type esploraAddressUTXO struct {
	Value  int64 `json:"value"`
	Status struct {
		Confirmed   bool  `json:"confirmed"`
		BlockHeight int64 `json:"block_height"`
	} `json:"status"`
}

func newBitcoinObserver(
	baseURL string,
	httpClient *http.Client,
	httpTimeout time.Duration,
	thresholds thresholdPolicy,
	confirmations confirmationPolicy,
) *bitcoinObserver {
	return &bitcoinObserver{
		baseURL:       strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient:    httpClient,
		httpTimeout:   httpTimeout,
		thresholds:    thresholds,
		confirmations: confirmations,
	}
}

func (o *bitcoinObserver) Observe(
	ctx context.Context,
	input dto.ObservePaymentRequestInput,
	expected *big.Int,
) (dto.ObservePaymentRequestOutput, *apperrors.AppError) {
	if o == nil || o.baseURL == "" {
		return dto.ObservePaymentRequestOutput{Supported: false}, nil
	}

	network := strings.ToLower(strings.TrimSpace(input.Network))
	address := strings.TrimSpace(input.AddressCanonical)
	requestCtx, cancel := context.WithTimeout(ctx, o.httpTimeout)
	defer cancel()

	stats, statsErr := o.fetchAddressStats(requestCtx, address, network)
	if statsErr != nil {
		return dto.ObservePaymentRequestOutput{}, statsErr
	}

	confirmedAmount := big.NewInt(stats.ChainStats.FundedTXOSum)
	detectedAmount := big.NewInt(stats.ChainStats.FundedTXOSum + stats.MempoolStats.FundedTXOSum)
	tipHeight := int64(0)
	if o.confirmations.btcMin > 1 {
		byDepth, tip, depthErr := o.confirmedAmountByMinConfirmations(requestCtx, address, network)
		if depthErr != nil {
			return dto.ObservePaymentRequestOutput{}, depthErr
		}
		confirmedAmount = byDepth
		tipHeight = tip
	}

	confirmedRequired := o.thresholds.confirmedRequired(expected)
	detectedRequired := o.thresholds.detectedRequired(expected)

	confirmed := confirmedAmount.Cmp(confirmedRequired) >= 0
	detected := !confirmed && detectedAmount.Cmp(detectedRequired) >= 0

	return dto.ObservePaymentRequestOutput{
		Supported:         true,
		ObservedAmount:    detectedAmount.String(),
		Detected:          detected,
		Confirmed:         confirmed,
		ObservationSource: "btc_esplora",
		ObservationDetails: map[string]any{
			"confirmed_amount_minor":   confirmedAmount.String(),
			"mempool_amount_minor":     fmt.Sprintf("%d", stats.MempoolStats.FundedTXOSum),
			"network":                  network,
			"detected_threshold_bps":   o.thresholds.detectedBPS,
			"confirmed_threshold_bps":  o.thresholds.confirmedBPS,
			"detected_required_minor":  detectedRequired.String(),
			"confirmed_required_minor": confirmedRequired.String(),
			"btc_min_confirmations":    o.confirmations.btcMin,
			"tip_height":               tipHeight,
		},
	}, nil
}

func (o *bitcoinObserver) fetchAddressStats(
	ctx context.Context,
	address string,
	network string,
) (esploraAddressStats, *apperrors.AppError) {
	endpoint := o.baseURL + "/address/" + url.PathEscape(address)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return esploraAddressStats{}, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to build bitcoin observation request",
			map[string]any{"error": err.Error(), "network": network},
		)
	}

	response, err := o.httpClient.Do(request)
	if err != nil {
		return esploraAddressStats{}, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to query bitcoin observation endpoint",
			map[string]any{"error": err.Error(), "network": network},
		)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return esploraAddressStats{}, apperrors.NewInternal(
			"chain_observation_failed",
			"bitcoin observation endpoint returned non-200 status",
			map[string]any{"status_code": response.StatusCode, "network": network},
		)
	}

	stats := esploraAddressStats{}
	if err := json.NewDecoder(response.Body).Decode(&stats); err != nil {
		return esploraAddressStats{}, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to decode bitcoin observation payload",
			map[string]any{"error": err.Error(), "network": network},
		)
	}
	return stats, nil
}

func (o *bitcoinObserver) fetchAddressUTXOs(
	ctx context.Context,
	address string,
	network string,
) ([]esploraAddressUTXO, *apperrors.AppError) {
	endpoint := o.baseURL + "/address/" + url.PathEscape(address) + "/utxo"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to build bitcoin utxo request",
			map[string]any{"error": err.Error(), "network": network},
		)
	}

	response, err := o.httpClient.Do(request)
	if err != nil {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to query bitcoin utxo endpoint",
			map[string]any{"error": err.Error(), "network": network},
		)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"bitcoin utxo endpoint returned non-200 status",
			map[string]any{"status_code": response.StatusCode, "network": network},
		)
	}

	out := []esploraAddressUTXO{}
	if err := json.NewDecoder(response.Body).Decode(&out); err != nil {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to decode bitcoin utxo payload",
			map[string]any{"error": err.Error(), "network": network},
		)
	}
	return out, nil
}

func (o *bitcoinObserver) fetchTipHeight(
	ctx context.Context,
	network string,
) (int64, *apperrors.AppError) {
	endpoint := o.baseURL + "/blocks/tip/height"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to build bitcoin tip-height request",
			map[string]any{"error": err.Error(), "network": network},
		)
	}

	response, err := o.httpClient.Do(request)
	if err != nil {
		return 0, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to query bitcoin tip-height endpoint",
			map[string]any{"error": err.Error(), "network": network},
		)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return 0, apperrors.NewInternal(
			"chain_observation_failed",
			"bitcoin tip-height endpoint returned non-200 status",
			map[string]any{"status_code": response.StatusCode, "network": network},
		)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to read bitcoin tip-height response",
			map[string]any{"error": err.Error(), "network": network},
		)
	}

	parsed, err := strconv.ParseInt(strings.TrimSpace(string(body)), 10, 64)
	if err != nil {
		return 0, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to parse bitcoin tip-height response",
			map[string]any{"error": err.Error(), "network": network},
		)
	}
	if parsed < 0 {
		return 0, apperrors.NewInternal(
			"chain_observation_failed",
			"bitcoin tip-height response is negative",
			map[string]any{"network": network},
		)
	}
	return parsed, nil
}

func (o *bitcoinObserver) confirmedAmountByMinConfirmations(
	ctx context.Context,
	address string,
	network string,
) (*big.Int, int64, *apperrors.AppError) {
	tipHeight, tipErr := o.fetchTipHeight(ctx, network)
	if tipErr != nil {
		return nil, 0, tipErr
	}

	utxos, utxoErr := o.fetchAddressUTXOs(ctx, address, network)
	if utxoErr != nil {
		return nil, tipHeight, utxoErr
	}

	sum := big.NewInt(0)
	required := int64(o.confirmations.btcMin)
	for _, utxo := range utxos {
		if !utxo.Status.Confirmed || utxo.Status.BlockHeight <= 0 {
			continue
		}
		confirmations := tipHeight - utxo.Status.BlockHeight + 1
		if confirmations < required {
			continue
		}
		sum.Add(sum, big.NewInt(utxo.Value))
	}

	return sum, tipHeight, nil
}
