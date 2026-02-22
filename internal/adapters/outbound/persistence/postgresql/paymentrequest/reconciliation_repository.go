package paymentrequest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	observeWindow time.Duration,
	limit int,
	leaseOwner string,
	leaseUntil time.Time,
) ([]dto.OpenPaymentRequestForReconciliation, *apperrors.AppError) {
	const query = `
WITH candidates AS (
  SELECT id
  FROM app.payment_requests
  WHERE (
      status IN ('pending', 'detected')
      OR (
        status IN ('confirmed', 'reorged')
        AND (
          CASE
            WHEN NULLIF(btrim(metadata #>> '{reconciliation,first_confirmed_at}'), '') IS NULL THEN TRUE
            WHEN NULLIF(btrim(metadata #>> '{reconciliation,first_confirmed_at}'), '') ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T' THEN
              (NULLIF(btrim(metadata #>> '{reconciliation,first_confirmed_at}'), '')::timestamptz + ($2 * interval '1 second')) > $1
            ELSE TRUE
          END
        )
      )
    )
    AND (reconcile_lease_until IS NULL OR reconcile_lease_until <= $1)
  ORDER BY created_at ASC, id ASC
  LIMIT $3
  FOR UPDATE SKIP LOCKED
)
UPDATE app.payment_requests AS pr
SET
  reconcile_lease_owner = $4,
  reconcile_lease_until = $5
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
  pr.token_decimals,
  pr.metadata
`
	observeWindowSeconds := int64(observeWindow / time.Second)
	if observeWindowSeconds <= 0 {
		observeWindowSeconds = 1
	}

	rows, err := r.db.QueryContext(
		ctx,
		query,
		now.UTC(),
		observeWindowSeconds,
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
			metadata       []byte
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
			&metadata,
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
		if len(metadata) > 0 {
			decoded := map[string]any{}
			if err := json.Unmarshal(metadata, &decoded); err != nil {
				return nil, apperrors.NewInternal(
					"payment_request_query_failed",
					"failed to decode payment request metadata",
					map[string]any{"error": err.Error(), "id": item.ID},
				)
			}
			item.Metadata = decoded
		} else {
			item.Metadata = map[string]any{}
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

func (r *Repository) SyncObservedSettlements(
	ctx context.Context,
	requestID string,
	chain string,
	network string,
	asset string,
	observedAt time.Time,
	settlements []dto.ObservedSettlementEvidence,
) (dto.ReconcileSettlementSyncResult, *apperrors.AppError) {
	const selectExistingQuery = `
SELECT
  evidence_ref,
  amount_minor::text,
  confirmations,
  block_height,
  block_hash,
  is_canonical,
  metadata
FROM app.payment_request_settlements
WHERE payment_request_id = $1
FOR UPDATE
`
	const insertQuery = `
INSERT INTO app.payment_request_settlements (
  payment_request_id,
  evidence_ref,
  amount_minor,
  confirmations,
  block_height,
  block_hash,
  is_canonical,
  metadata,
  first_seen_at,
  last_seen_at,
  updated_at
)
VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11
)
`
	const updateQuery = `
UPDATE app.payment_request_settlements
SET
  amount_minor = $3,
  confirmations = $4,
  block_height = $5,
  block_hash = $6,
  is_canonical = $7,
  metadata = $8::jsonb,
  last_seen_at = $9,
  updated_at = $10
WHERE payment_request_id = $1
  AND evidence_ref = $2
`

	requestID = strings.TrimSpace(requestID)
	_ = chain
	_ = network
	_ = asset
	now := observedAt.UTC()

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
			"payment_request_update_failed",
			"failed to begin settlement sync transaction",
			map[string]any{"error": err.Error(), "id": requestID},
		)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	existingRows, queryErr := tx.QueryContext(ctx, selectExistingQuery, requestID)
	if queryErr != nil {
		return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
			"payment_request_query_failed",
			"failed to load existing settlement evidence",
			map[string]any{"error": queryErr.Error(), "id": requestID},
		)
	}

	existingByRef := map[string]settlementState{}
	for existingRows.Next() {
		var (
			evidenceRef    string
			amountMinor    string
			confirmations  int
			blockHeight    sql.NullInt64
			blockHash      sql.NullString
			isCanonical    bool
			metadataRaw    []byte
			metadataString string
			err            error
		)
		if err = existingRows.Scan(
			&evidenceRef,
			&amountMinor,
			&confirmations,
			&blockHeight,
			&blockHash,
			&isCanonical,
			&metadataRaw,
		); err != nil {
			existingRows.Close()
			return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
				"payment_request_query_failed",
				"failed to parse existing settlement evidence",
				map[string]any{"error": err.Error(), "id": requestID},
			)
		}

		metadataString, err = canonicalizeJSON(metadataRaw)
		if err != nil {
			existingRows.Close()
			return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
				"payment_request_query_failed",
				"failed to decode existing settlement metadata",
				map[string]any{"error": err.Error(), "id": requestID, "evidence_ref": evidenceRef},
			)
		}

		existingByRef[evidenceRef] = settlementState{
			amountMinor:   strings.TrimSpace(amountMinor),
			confirmations: confirmations,
			blockHeight:   nullableInt64Ptr(blockHeight),
			blockHash:     nullableStringPtr(blockHash),
			isCanonical:   isCanonical,
			metadataJSON:  metadataString,
		}
	}
	if rowsErr := existingRows.Err(); rowsErr != nil {
		existingRows.Close()
		return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
			"payment_request_query_failed",
			"failed while loading existing settlement evidence",
			map[string]any{"error": rowsErr.Error(), "id": requestID},
		)
	}
	existingRows.Close()

	observedRefs := make([]string, 0, len(settlements))
	observedRefSet := make(map[string]struct{}, len(settlements))
	for _, settlement := range settlements {
		evidenceRef := strings.TrimSpace(settlement.EvidenceRef)
		amountMinor := strings.TrimSpace(settlement.AmountMinor)
		if evidenceRef == "" || amountMinor == "" {
			continue
		}

		if _, seen := observedRefSet[evidenceRef]; !seen {
			observedRefSet[evidenceRef] = struct{}{}
			observedRefs = append(observedRefs, evidenceRef)
		}

		metadataRaw, metadataString, metadataErr := encodeCanonicalMetadata(settlement.Metadata)
		if metadataErr != nil {
			return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
				"payment_request_update_failed",
				"failed to encode settlement metadata",
				map[string]any{"error": metadataErr.Error(), "id": requestID, "evidence_ref": evidenceRef},
			)
		}

		confirmations := settlement.Confirmations
		if confirmations < 0 {
			confirmations = 0
		}

		nextState := settlementState{
			amountMinor:   amountMinor,
			confirmations: confirmations,
			blockHeight:   copyInt64Ptr(settlement.BlockHeight),
			blockHash:     copyTrimmedStringPtr(settlement.BlockHash),
			isCanonical:   settlement.IsCanonical,
			metadataJSON:  metadataString,
		}

		currentState, exists := existingByRef[evidenceRef]
		var blockHeightValue any
		if nextState.blockHeight != nil {
			blockHeightValue = *nextState.blockHeight
		}
		var blockHashValue any
		if nextState.blockHash != nil {
			blockHashValue = *nextState.blockHash
		}

		if !exists {
			if _, execErr := tx.ExecContext(
				ctx,
				insertQuery,
				requestID,
				evidenceRef,
				nextState.amountMinor,
				nextState.confirmations,
				blockHeightValue,
				blockHashValue,
				nextState.isCanonical,
				metadataRaw,
				now,
				now,
				now,
			); execErr != nil {
				return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
					"payment_request_update_failed",
					"failed to insert settlement evidence",
					map[string]any{"error": execErr.Error(), "id": requestID, "evidence_ref": evidenceRef},
				)
			}
			existingByRef[evidenceRef] = nextState
			continue
		}

		if currentState.equals(nextState) {
			continue
		}

		if _, execErr := tx.ExecContext(
			ctx,
			updateQuery,
			requestID,
			evidenceRef,
			nextState.amountMinor,
			nextState.confirmations,
			blockHeightValue,
			blockHashValue,
			nextState.isCanonical,
			metadataRaw,
			now,
			now,
		); execErr != nil {
			return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
				"payment_request_update_failed",
				"failed to update settlement evidence",
				map[string]any{"error": execErr.Error(), "id": requestID, "evidence_ref": evidenceRef},
			)
		}
		existingByRef[evidenceRef] = nextState
	}

	orphansUpdated := int64(0)
	if len(observedRefs) == 0 {
		result, execErr := tx.ExecContext(
			ctx,
			`UPDATE app.payment_request_settlements
			 SET is_canonical = FALSE, updated_at = $2
			 WHERE payment_request_id = $1
			   AND is_canonical = TRUE`,
			requestID,
			now,
		)
		if execErr != nil {
			return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
				"payment_request_update_failed",
				"failed to mark orphaned settlements",
				map[string]any{"error": execErr.Error(), "id": requestID},
			)
		}
		rowsAffected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
				"payment_request_update_failed",
				"failed to verify orphaned settlement updates",
				map[string]any{"error": rowsErr.Error(), "id": requestID},
			)
		}
		orphansUpdated = rowsAffected
	} else {
		args := make([]any, 0, 2+len(observedRefs))
		args = append(args, requestID, now)
		placeholders := make([]string, 0, len(observedRefs))
		for _, ref := range observedRefs {
			args = append(args, ref)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}

		query := fmt.Sprintf(`
UPDATE app.payment_request_settlements
SET is_canonical = FALSE, updated_at = $2
WHERE payment_request_id = $1
  AND is_canonical = TRUE
  AND evidence_ref NOT IN (%s)
`, strings.Join(placeholders, ", "))

		result, execErr := tx.ExecContext(ctx, query, args...)
		if execErr != nil {
			return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
				"payment_request_update_failed",
				"failed to mark missing settlements as orphaned",
				map[string]any{"error": execErr.Error(), "id": requestID},
			)
		}
		rowsAffected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
				"payment_request_update_failed",
				"failed to verify orphaned settlement updates",
				map[string]any{"error": rowsErr.Error(), "id": requestID},
			)
		}
		orphansUpdated = rowsAffected
	}

	summary := dto.ReconcileSettlementSyncResult{
		NewlyOrphanedCount: int(orphansUpdated),
	}
	countErr := tx.QueryRowContext(
		ctx,
		`
SELECT
  COUNT(*) FILTER (WHERE is_canonical = TRUE) AS canonical_count,
  COUNT(*) FILTER (WHERE is_canonical = FALSE) AS non_canonical_count
FROM app.payment_request_settlements
WHERE payment_request_id = $1
`,
		requestID,
	).Scan(&summary.CanonicalCount, &summary.NonCanonicalCount)
	if countErr != nil {
		return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
			"payment_request_query_failed",
			"failed to summarize settlement evidence",
			map[string]any{"error": countErr.Error(), "id": requestID},
		)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return dto.ReconcileSettlementSyncResult{}, apperrors.NewInternal(
			"payment_request_update_failed",
			"failed to commit settlement sync transaction",
			map[string]any{"error": commitErr.Error(), "id": requestID},
		)
	}
	committed = true

	return summary, nil
}

type settlementState struct {
	amountMinor   string
	confirmations int
	blockHeight   *int64
	blockHash     *string
	isCanonical   bool
	metadataJSON  string
}

func (s settlementState) equals(other settlementState) bool {
	if s.amountMinor != other.amountMinor ||
		s.confirmations != other.confirmations ||
		s.isCanonical != other.isCanonical ||
		s.metadataJSON != other.metadataJSON {
		return false
	}
	if !equalInt64Ptr(s.blockHeight, other.blockHeight) {
		return false
	}
	return equalStringPtr(s.blockHash, other.blockHash)
}

func equalInt64Ptr(left *int64, right *int64) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return *left == *right
}

func equalStringPtr(left *string, right *string) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return *left == *right
}

func nullableInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	typed := value.Int64
	return &typed
}

func nullableStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func copyInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func copyTrimmedStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func encodeCanonicalMetadata(metadata map[string]any) ([]byte, string, error) {
	if len(metadata) == 0 {
		return []byte("{}"), "{}", nil
	}

	encoded, err := json.Marshal(metadata)
	if err != nil {
		return nil, "", err
	}
	canonical, err := canonicalizeJSON(encoded)
	if err != nil {
		return nil, "", err
	}
	return []byte(canonical), canonical, nil
}

func canonicalizeJSON(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "{}", nil
	}

	decoded := any(nil)
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
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
    AND $2 <> $3
    AND $3 IN ('detected', 'confirmed', 'reorged', 'expired')
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
	              'observation_source', NULLIF($4::jsonb #>> '{reconciliation,observation_source}', ''),
	              'transition_reason', NULLIF($4::jsonb #>> '{reconciliation,transition_reason}', ''),
	              'finality_reached', $4::jsonb #> '{reconciliation,finality_reached}',
	              'evidence_summary', $4::jsonb #> '{reconciliation,evidence_summary}'
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
	if !metadata.UpdatedAt.IsZero() ||
		metadata.ObservedAmountMinor != "" ||
		metadata.ObservationSource != "" ||
		len(metadata.ObservationDetails) > 0 ||
		metadata.TransitionReason != "" ||
		metadata.FinalityReached != nil ||
		metadata.EvidenceSummary != nil ||
		metadata.FirstConfirmedAt != nil ||
		metadata.FinalityReachedAt != nil ||
		metadata.StabilitySignal != "" ||
		metadata.StabilityPromoteStreak > 0 ||
		metadata.StabilityDemoteStreak > 0 {
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
