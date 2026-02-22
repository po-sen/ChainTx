package paymentrequest

import (
	"context"
	"database/sql"
	"encoding/json"
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

func (r *ReadModel) ListSettlementsByPaymentRequestID(
	ctx context.Context,
	id string,
) ([]dto.PaymentRequestSettlementResource, bool, *apperrors.AppError) {
	const query = `
WITH target_request AS (
  SELECT id
  FROM app.payment_requests
  WHERE id = $1
)
SELECT
  s.evidence_ref,
  s.amount_minor::text,
  s.confirmations,
  s.block_height,
  s.block_hash,
  s.is_canonical,
  s.metadata,
  s.first_seen_at,
  s.last_seen_at,
  s.updated_at
FROM target_request tr
LEFT JOIN app.payment_request_settlements s
  ON s.payment_request_id = tr.id
ORDER BY s.first_seen_at ASC NULLS LAST, s.evidence_ref ASC NULLS LAST
`

	rows, err := r.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, false, apperrors.NewInternal(
			"payment_request_query_failed",
			"failed to query payment request settlements",
			map[string]any{"error": err.Error(), "id": id},
		)
	}
	defer rows.Close()

	settlements := make([]dto.PaymentRequestSettlementResource, 0)
	requestFound := false
	for rows.Next() {
		requestFound = true

		var (
			evidenceRef   sql.NullString
			amountMinor   sql.NullString
			confirmations sql.NullInt64
			blockHeight   sql.NullInt64
			blockHash     sql.NullString
			isCanonical   sql.NullBool
			metadataRaw   []byte
			firstSeenAt   sql.NullTime
			lastSeenAt    sql.NullTime
			updatedAt     sql.NullTime
		)

		if scanErr := rows.Scan(
			&evidenceRef,
			&amountMinor,
			&confirmations,
			&blockHeight,
			&blockHash,
			&isCanonical,
			&metadataRaw,
			&firstSeenAt,
			&lastSeenAt,
			&updatedAt,
		); scanErr != nil {
			return nil, false, apperrors.NewInternal(
				"payment_request_query_failed",
				"failed to parse payment request settlement row",
				map[string]any{"error": scanErr.Error(), "id": id},
			)
		}

		if !evidenceRef.Valid {
			continue
		}

		metadata := map[string]any{}
		if len(metadataRaw) > 0 {
			if decodeErr := json.Unmarshal(metadataRaw, &metadata); decodeErr != nil {
				return nil, false, apperrors.NewInternal(
					"payment_request_query_failed",
					"failed to decode payment request settlement metadata",
					map[string]any{"error": decodeErr.Error(), "id": id, "evidence_ref": evidenceRef.String},
				)
			}
		}

		item := dto.PaymentRequestSettlementResource{
			EvidenceRef:   strings.TrimSpace(evidenceRef.String),
			AmountMinor:   strings.TrimSpace(amountMinor.String),
			Confirmations: int(confirmations.Int64),
			IsCanonical:   isCanonical.Bool,
			Metadata:      metadata,
		}
		if blockHeight.Valid {
			value := blockHeight.Int64
			item.BlockHeight = &value
		}
		if blockHash.Valid {
			value := strings.TrimSpace(blockHash.String)
			item.BlockHash = &value
		}
		if firstSeenAt.Valid {
			item.FirstSeenAt = firstSeenAt.Time.UTC()
		}
		if lastSeenAt.Valid {
			item.LastSeenAt = lastSeenAt.Time.UTC()
		}
		if updatedAt.Valid {
			item.UpdatedAt = updatedAt.Time.UTC()
		}

		settlements = append(settlements, item)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, false, apperrors.NewInternal(
			"payment_request_query_failed",
			"failed while iterating payment request settlements",
			map[string]any{"error": rowsErr.Error(), "id": id},
		)
	}

	return settlements, requestFound, nil
}
