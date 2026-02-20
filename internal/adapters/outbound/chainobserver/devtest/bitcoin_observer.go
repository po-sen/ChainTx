package devtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type bitcoinObserver struct {
	baseURL     string
	httpClient  *http.Client
	httpTimeout time.Duration
	thresholds  thresholdPolicy
}

type esploraAddressStats struct {
	ChainStats struct {
		FundedTXOSum int64 `json:"funded_txo_sum"`
	} `json:"chain_stats"`
	MempoolStats struct {
		FundedTXOSum int64 `json:"funded_txo_sum"`
	} `json:"mempool_stats"`
}

func newBitcoinObserver(
	baseURL string,
	httpClient *http.Client,
	httpTimeout time.Duration,
	thresholds thresholdPolicy,
) *bitcoinObserver {
	return &bitcoinObserver{
		baseURL:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient:  httpClient,
		httpTimeout: httpTimeout,
		thresholds:  thresholds,
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
	endpoint := o.baseURL + "/address/" + url.PathEscape(address)

	requestCtx, cancel := context.WithTimeout(ctx, o.httpTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return dto.ObservePaymentRequestOutput{}, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to build bitcoin observation request",
			map[string]any{"error": err.Error(), "network": network},
		)
	}

	response, err := o.httpClient.Do(request)
	if err != nil {
		return dto.ObservePaymentRequestOutput{}, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to query bitcoin observation endpoint",
			map[string]any{"error": err.Error(), "network": network},
		)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return dto.ObservePaymentRequestOutput{}, apperrors.NewInternal(
			"chain_observation_failed",
			"bitcoin observation endpoint returned non-200 status",
			map[string]any{"status_code": response.StatusCode, "network": network},
		)
	}

	stats := esploraAddressStats{}
	if err := json.NewDecoder(response.Body).Decode(&stats); err != nil {
		return dto.ObservePaymentRequestOutput{}, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to decode bitcoin observation payload",
			map[string]any{"error": err.Error(), "network": network},
		)
	}

	confirmedAmount := big.NewInt(stats.ChainStats.FundedTXOSum)
	detectedAmount := big.NewInt(stats.ChainStats.FundedTXOSum + stats.MempoolStats.FundedTXOSum)

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
		},
	}, nil
}
