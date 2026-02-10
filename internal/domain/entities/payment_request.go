package entities

import (
	"time"

	valueobjects "chaintx/internal/domain/value_objects"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type PaymentInstructions struct {
	AddressCanonical string
	AddressScheme    string
	DerivationIndex  int64
	ChainID          *int64
	TokenStandard    *string
	TokenContract    *string
	TokenDecimals    *int
}

type PaymentRequest struct {
	ID                  string
	WalletAccountID     string
	Status              valueobjects.PaymentRequestStatus
	Chain               string
	Network             string
	Asset               string
	ExpectedAmountMinor *string
	Metadata            map[string]any
	ExpiresAt           time.Time
	CreatedAt           time.Time
	Instructions        PaymentInstructions
}

type NewPaymentRequestInput struct {
	ID                  string
	WalletAccountID     string
	Chain               string
	Network             string
	Asset               string
	ExpectedAmountMinor *string
	Metadata            map[string]any
	ExpiresAt           time.Time
	CreatedAt           time.Time
	Instructions        PaymentInstructions
}

func NewPendingPaymentRequest(input NewPaymentRequestInput) (PaymentRequest, *apperrors.AppError) {
	if input.ID == "" {
		return PaymentRequest{}, apperrors.NewInternal(
			"payment_request_id_missing",
			"payment request id is required",
			nil,
		)
	}
	if input.WalletAccountID == "" {
		return PaymentRequest{}, apperrors.NewInternal(
			"wallet_account_id_missing",
			"wallet account id is required",
			nil,
		)
	}

	if !input.ExpiresAt.After(input.CreatedAt) {
		return PaymentRequest{}, apperrors.NewValidation(
			"invalid_request",
			"expires_at must be greater than created_at",
			map[string]any{"field": "expires_at"},
		)
	}

	if input.Instructions.DerivationIndex < 0 {
		return PaymentRequest{}, apperrors.NewInternal(
			"derivation_index_invalid",
			"derivation index must be non-negative",
			map[string]any{"derivation_index": input.Instructions.DerivationIndex},
		)
	}

	return PaymentRequest{
		ID:                  input.ID,
		WalletAccountID:     input.WalletAccountID,
		Status:              valueobjects.NewPendingPaymentRequestStatus(),
		Chain:               input.Chain,
		Network:             input.Network,
		Asset:               input.Asset,
		ExpectedAmountMinor: input.ExpectedAmountMinor,
		Metadata:            cloneMetadata(input.Metadata),
		ExpiresAt:           input.ExpiresAt.UTC(),
		CreatedAt:           input.CreatedAt.UTC(),
		Instructions:        input.Instructions,
	}, nil
}

func cloneMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}

	copyMap := make(map[string]any, len(metadata))
	for key, value := range metadata {
		copyMap[key] = value
	}

	return copyMap
}
