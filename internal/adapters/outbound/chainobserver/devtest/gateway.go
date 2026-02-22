package devtest

import (
	"context"
	"math/big"
	"net/http"
	"strings"
	"time"

	"chaintx/internal/application/dto"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

const (
	defaultHTTPTimeout  = 5 * time.Second
	defaultDetectedBPS  = 10000
	defaultConfirmedBPS = 10000
	defaultBTCMinConf   = 1
	defaultEVMMinConf   = 1
)

type Config struct {
	BTCExploraBaseURL  string
	EVMRPCURLs         map[string]string
	DetectedBPS        int
	ConfirmedBPS       int
	BTCMinConf         int
	BTCFinalityMinConf int
	EVMMinConf         int
	EVMFinalityMinConf int
	HTTPTimeout        time.Duration
}

type paymentObserver interface {
	Observe(
		ctx context.Context,
		input dto.ObservePaymentRequestInput,
		expected *big.Int,
	) (dto.ObservePaymentRequestOutput, *apperrors.AppError)
}

type Gateway struct {
	observers map[string]paymentObserver
}

var _ portsout.PaymentChainObserverGateway = (*Gateway)(nil)

func NewGateway(cfg Config) *Gateway {
	httpTimeout := cfg.HTTPTimeout
	if httpTimeout <= 0 {
		httpTimeout = defaultHTTPTimeout
	}
	thresholds := newThresholdPolicy(
		cfg.DetectedBPS,
		cfg.ConfirmedBPS,
		defaultDetectedBPS,
		defaultConfirmedBPS,
	)
	confirmations := newConfirmationPolicy(
		cfg.BTCMinConf,
		cfg.BTCFinalityMinConf,
		cfg.EVMMinConf,
		cfg.EVMFinalityMinConf,
		defaultBTCMinConf,
		defaultEVMMinConf,
	)

	httpClient := &http.Client{}
	rpcClient := newJSONRPCClient(httpClient, httpTimeout)

	return &Gateway{
		observers: map[string]paymentObserver{
			"bitcoin":  newBitcoinObserver(cfg.BTCExploraBaseURL, httpClient, httpTimeout, thresholds, confirmations),
			"ethereum": newEVMObserver(cfg.EVMRPCURLs, rpcClient, thresholds, confirmations),
		},
	}
}

func (g *Gateway) ObservePaymentRequest(
	ctx context.Context,
	input dto.ObservePaymentRequestInput,
) (dto.ObservePaymentRequestOutput, *apperrors.AppError) {
	expected, appErr := parseExpectedThreshold(input.ExpectedAmountMinor)
	if appErr != nil {
		return dto.ObservePaymentRequestOutput{}, appErr
	}

	chain := strings.ToLower(strings.TrimSpace(input.Chain))
	observer, exists := g.observers[chain]
	if !exists || observer == nil {
		return dto.ObservePaymentRequestOutput{Supported: false}, nil
	}

	return observer.Observe(ctx, input, expected)
}
