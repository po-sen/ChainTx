package dto

import (
	"context"
	"time"

	apperrors "chaintx/internal/shared_kernel/errors"
)

type IdempotencyScope struct {
	PrincipalID string
	HTTPMethod  string
	HTTPPath    string
}

type CreatePaymentRequestCommand struct {
	IdempotencyScope    IdempotencyScope
	IdempotencyKey      string
	Chain               string
	Network             string
	Asset               string
	WebhookURL          string
	ExpectedAmountMinor *string
	ExpiresInSeconds    *int64
	Metadata            map[string]any
}

type CreatePaymentRequestOutput struct {
	Resource PaymentRequestResource
	Replayed bool
}

type CreatePaymentRequestPersistenceCommand struct {
	ResourceID           string
	IdempotencyScope     IdempotencyScope
	IdempotencyKey       string
	RequestHash          string
	HashAlgorithm        string
	Status               string
	Chain                string
	Network              string
	Asset                string
	WebhookURL           string
	ExpectedAmountMinor  *string
	Metadata             map[string]any
	ExpiresAt            time.Time
	IdempotencyExpiresAt time.Time
	CreatedAt            time.Time
	AssetCatalogSnapshot AssetCatalogEntry
	AllocationMode       string
}

type ResolvePaymentAddressInput struct {
	Chain                  string
	Network                string
	AddressScheme          string
	KeysetID               string
	DerivationPathTemplate string
	DerivationIndex        int64
	ChainID                *int64
}

type ResolvePaymentAddressOutput struct {
	AddressCanonical string
	Address          string
}

type ResolvePaymentAddressFunc func(ctx context.Context, input ResolvePaymentAddressInput) (ResolvePaymentAddressOutput, *apperrors.AppError)

type CreatePaymentRequestPersistenceResult struct {
	Resource PaymentRequestResource
	Replayed bool
}

type GetPaymentRequestQuery struct {
	ID string
}

type GetPaymentRequestSettlementsQuery struct {
	ID string
}

type PaymentRequestResource struct {
	ID                  string              `json:"id"`
	Status              string              `json:"status"`
	Chain               string              `json:"chain"`
	Network             string              `json:"network"`
	Asset               string              `json:"asset"`
	ExpectedAmountMinor *string             `json:"expected_amount_minor,omitempty"`
	ExpiresAt           time.Time           `json:"expires_at"`
	CreatedAt           time.Time           `json:"created_at"`
	PaymentInstructions PaymentInstructions `json:"payment_instructions"`
}

type PaymentInstructions struct {
	Address         string  `json:"address"`
	AddressScheme   string  `json:"address_scheme"`
	DerivationIndex int64   `json:"derivation_index"`
	ChainID         *int64  `json:"chain_id,omitempty"`
	TokenStandard   *string `json:"token_standard,omitempty"`
	TokenContract   *string `json:"token_contract,omitempty"`
	TokenDecimals   *int    `json:"token_decimals,omitempty"`
}

type PaymentRequestSettlementResource struct {
	EvidenceRef   string         `json:"evidence_ref"`
	AmountMinor   string         `json:"amount_minor"`
	Confirmations int            `json:"confirmations"`
	BlockHeight   *int64         `json:"block_height,omitempty"`
	BlockHash     *string        `json:"block_hash,omitempty"`
	IsCanonical   bool           `json:"is_canonical"`
	Metadata      map[string]any `json:"metadata"`
	FirstSeenAt   time.Time      `json:"first_seen_at"`
	LastSeenAt    time.Time      `json:"last_seen_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type PaymentRequestSettlementsResource struct {
	PaymentRequestID string                             `json:"payment_request_id"`
	Settlements      []PaymentRequestSettlementResource `json:"settlements"`
}
