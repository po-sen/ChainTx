package paymentrequest

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"chaintx/internal/application/dto"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

var _ portsout.PaymentRequestReconciliationRepository = (*Repository)(nil)

func (r *Repository) ClaimOpenForReconciliation(
	ctx context.Context,
	now time.Time,
	limit int,
	leaseOwner string,
	leaseUntil time.Time,
) ([]dto.OpenPaymentRequestForReconciliation, *apperrors.AppError) {
	const query = `
WITH candidates AS (
  SELECT id
  FROM app.payment_requests
  WHERE status IN ('pending', 'detected')
    AND (reconcile_lease_until IS NULL OR reconcile_lease_until <= $1)
  ORDER BY created_at ASC, id ASC
  LIMIT $2
  FOR UPDATE SKIP LOCKED
)
UPDATE app.payment_requests AS pr
SET
  reconcile_lease_owner = $3,
  reconcile_lease_until = $4
FROM candidates
WHERE pr.id = candidates.id
RETURNING
  pr.id,
  pr.status,
  pr.chain,
  pr.network,
  pr.asset,
  pr.expected_amount_minor::text,
  pr.address_canonical,
  pr.expires_at,
  pr.chain_id,
  pr.token_standard,
  pr.token_contract,
  pr.token_decimals
`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		now.UTC(),
		limit,
		strings.TrimSpace(leaseOwner),
		leaseUntil.UTC(),
	)
	if err != nil {
		return nil, apperrors.NewInternal(
			"payment_request_query_failed",
			"failed to claim open payment requests for reconciliation",
			map[string]any{"error": err.Error()},
		)
	}
	defer rows.Close()

	items := make([]dto.OpenPaymentRequestForReconciliation, 0, limit)
	for rows.Next() {
		item := dto.OpenPaymentRequestForReconciliation{}

		var (
			expectedAmount sql.NullString
			chainID        sql.NullInt64
			tokenStandard  sql.NullString
			tokenContract  sql.NullString
			tokenDecimals  sql.NullInt64
		)

		if err := rows.Scan(
			&item.ID,
			&item.Status,
			&item.Chain,
			&item.Network,
			&item.Asset,
			&expectedAmount,
			&item.AddressCanonical,
			&item.ExpiresAt,
			&chainID,
			&tokenStandard,
			&tokenContract,
			&tokenDecimals,
		); err != nil {
			return nil, apperrors.NewInternal(
				"payment_request_query_failed",
				"failed to parse open payment request row",
				map[string]any{"error": err.Error()},
			)
		}

		item.Status = strings.ToLower(strings.TrimSpace(item.Status))
		item.Chain = strings.ToLower(strings.TrimSpace(item.Chain))
		item.Network = strings.ToLower(strings.TrimSpace(item.Network))
		item.Asset = strings.ToUpper(strings.TrimSpace(item.Asset))
		item.AddressCanonical = strings.TrimSpace(item.AddressCanonical)
		item.ExpiresAt = item.ExpiresAt.UTC()

		if expectedAmount.Valid {
			value := strings.TrimSpace(expectedAmount.String)
			if value != "" {
				item.ExpectedAmountMinor = &value
			}
		}
		if chainID.Valid {
			value := chainID.Int64
			item.ChainID = &value
		}
		if tokenStandard.Valid {
			value := strings.TrimSpace(tokenStandard.String)
			if value != "" {
				item.TokenStandard = &value
			}
		}
		if tokenContract.Valid {
			value := strings.TrimSpace(tokenContract.String)
			if value != "" {
				item.TokenContract = &value
			}
		}
		if tokenDecimals.Valid {
			value := int(tokenDecimals.Int64)
			item.TokenDecimals = &value
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternal(
			"payment_request_query_failed",
			"failed while iterating open payment requests",
			map[string]any{"error": err.Error()},
		)
	}

	return items, nil
}

func (r *Repository) TransitionStatusIfCurrent(
	ctx context.Context,
	id string,
	currentStatus string,
	nextStatus string,
	updatedAt time.Time,
	leaseOwner string,
	metadata dto.ReconcileTransitionMetadata,
) (bool, *apperrors.AppError) {
	const query = `
WITH updated AS (
  UPDATE app.payment_requests
  SET
    status = $3,
    metadata = COALESCE(metadata, '{}'::jsonb) || $4::jsonb,
    updated_at = $5,
    reconcile_lease_owner = NULL,
    reconcile_lease_until = NULL
  WHERE id = $1
    AND status = $2
    AND (reconcile_lease_owner IS NULL OR reconcile_lease_owner = $6)
  RETURNING
    id,
    chain,
    network,
    asset,
    expected_amount_minor::text,
    webhook_url,
    address_canonical,
    expires_at
),
event_rows AS (
  SELECT
    u.id,
    u.chain,
    u.network,
    u.asset,
    u.expected_amount_minor,
    u.webhook_url,
    u.address_canonical,
    u.expires_at,
    ('evt_' || md5(random()::text || clock_timestamp()::text || u.id)) AS event_id
  FROM updated AS u
  WHERE $7 = TRUE
    AND $3 IN ('detected', 'confirmed', 'expired')
    AND NULLIF(btrim(u.webhook_url), '') IS NOT NULL
),
inserted_events AS (
  INSERT INTO app.webhook_outbox_events (
    event_id,
    event_type,
    payment_request_id,
    destination_url,
    payload,
    delivery_status,
    attempts,
    max_attempts,
    next_attempt_at,
    created_at,
    updated_at
  )
  SELECT
    e.event_id,
    'payment_request.status_changed',
    e.id,
    e.webhook_url,
    jsonb_strip_nulls(
      jsonb_build_object(
        'event_id', e.event_id,
        'event_type', 'payment_request.status_changed',
        'occurred_at', $5,
        'data',
        jsonb_build_object(
          'payment_request',
          jsonb_strip_nulls(
            jsonb_build_object(
              'id', e.id,
              'chain', e.chain,
              'network', e.network,
              'asset', e.asset,
              'expected_amount_minor', e.expected_amount_minor,
              'address_canonical', e.address_canonical,
              'expires_at', e.expires_at,
              'previous_status', $2,
              'current_status', $3,
              'observed_amount_minor', NULLIF($4::jsonb #>> '{reconciliation,observed_amount_minor}', ''),
              'observation_source', NULLIF($4::jsonb #>> '{reconciliation,observation_source}', '')
            )
          )
        )
      )
    ),
    'pending',
    0,
    $8,
    $5,
    $5,
    $5
  FROM event_rows AS e
)
SELECT COUNT(*) FROM updated
`

	metadataPayload := map[string]any{}
	if !metadata.UpdatedAt.IsZero() || metadata.ObservedAmountMinor != "" || metadata.ObservationSource != "" || len(metadata.ObservationDetails) > 0 {
		metadataPayload["reconciliation"] = metadata
	}

	encodedMetadata := []byte("{}")
	if len(metadataPayload) > 0 {
		encoded, err := json.Marshal(metadataPayload)
		if err != nil {
			return false, apperrors.NewInternal(
				"payment_request_update_failed",
				"failed to encode reconciliation metadata",
				map[string]any{"error": err.Error(), "id": id},
			)
		}
		encodedMetadata = encoded
	}

	var updatedCount int
	err := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(id),
		strings.ToLower(strings.TrimSpace(currentStatus)),
		strings.ToLower(strings.TrimSpace(nextStatus)),
		encodedMetadata,
		updatedAt.UTC(),
		strings.TrimSpace(leaseOwner),
		r.webhookOutboxEnabled,
		r.webhookMaxAttempts,
	).Scan(&updatedCount)
	if err != nil {
		return false, apperrors.NewInternal(
			"payment_request_update_failed",
			"failed to transition payment request status",
			map[string]any{"error": err.Error(), "id": id},
		)
	}

	return updatedCount == 1, nil
}
