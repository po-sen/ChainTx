package paymentrequest

import (
	"context"
	"database/sql"
	stderrors "errors"
	"strings"

	"chaintx/internal/application/dto"
	portsout "chaintx/internal/application/ports/out"
	valueobjects "chaintx/internal/domain/value_objects"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type ReadModel struct {
	db *sql.DB
}

var _ portsout.PaymentRequestReadModel = (*ReadModel)(nil)

func NewReadModel(db *sql.DB) *ReadModel {
	return &ReadModel{db: db}
}

func (r *ReadModel) GetByID(ctx context.Context, id string) (dto.PaymentRequestResource, bool, *apperrors.AppError) {
	const query = `
SELECT
  id,
  status,
  chain,
  network,
  asset,
  expected_amount_minor::text,
  address_canonical,
  address_scheme,
  derivation_index,
  chain_id,
  token_standard,
  token_contract,
  token_decimals,
  expires_at,
  created_at
FROM app.payment_requests
WHERE id = $1
`

	var (
		resource         dto.PaymentRequestResource
		expectedAmount   sql.NullString
		addressCanonical string
		chainID          sql.NullInt64
		tokenStandard    sql.NullString
		tokenContract    sql.NullString
		tokenDecimals    sql.NullInt64
	)

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&resource.ID,
		&resource.Status,
		&resource.Chain,
		&resource.Network,
		&resource.Asset,
		&expectedAmount,
		&addressCanonical,
		&resource.PaymentInstructions.AddressScheme,
		&resource.PaymentInstructions.DerivationIndex,
		&chainID,
		&tokenStandard,
		&tokenContract,
		&tokenDecimals,
		&resource.ExpiresAt,
		&resource.CreatedAt,
	)
	if stderrors.Is(err, sql.ErrNoRows) {
		return dto.PaymentRequestResource{}, false, nil
	}
	if err != nil {
		return dto.PaymentRequestResource{}, false, apperrors.NewInternal(
			"payment_request_query_failed",
			"failed to query payment request",
			map[string]any{"error": err.Error(), "id": id},
		)
	}

	resource.Chain = strings.ToLower(resource.Chain)
	resource.Network = strings.ToLower(resource.Network)
	resource.Asset = strings.ToUpper(resource.Asset)

	if expectedAmount.Valid {
		value := expectedAmount.String
		resource.ExpectedAmountMinor = &value
	}
	if chainID.Valid {
		value := chainID.Int64
		resource.PaymentInstructions.ChainID = &value
	}
	if tokenStandard.Valid {
		value := tokenStandard.String
		resource.PaymentInstructions.TokenStandard = &value
	}
	if tokenContract.Valid {
		normalized, appErr := valueobjects.NormalizeTokenContract(tokenContract.String)
		if appErr != nil {
			return dto.PaymentRequestResource{}, false, apperrors.NewInternal(
				"payment_request_token_contract_invalid",
				"stored token contract is invalid",
				map[string]any{"id": id},
			)
		}
		resource.PaymentInstructions.TokenContract = &normalized
	}
	if tokenDecimals.Valid {
		value := int(tokenDecimals.Int64)
		resource.PaymentInstructions.TokenDecimals = &value
	}

	addressResponse, appErr := valueobjects.FormatAddressForResponse(resource.Chain, addressCanonical)
	if appErr != nil {
		return dto.PaymentRequestResource{}, false, appErr
	}
	resource.PaymentInstructions.Address = addressResponse

	return resource, true, nil
}
