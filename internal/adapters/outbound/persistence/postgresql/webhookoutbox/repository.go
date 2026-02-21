package webhookoutbox

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"chaintx/internal/application/dto"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type Repository struct {
	db *sql.DB
}

var _ portsout.WebhookOutboxRepository = (*Repository)(nil)
var _ portsout.WebhookOutboxReadModel = (*Repository)(nil)

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ClaimPendingForDispatch(
	ctx context.Context,
	now time.Time,
	limit int,
	leaseOwner string,
	leaseUntil time.Time,
) ([]dto.PendingWebhookOutboxEvent, *apperrors.AppError) {
	const query = `
WITH candidates AS (
  SELECT id
  FROM app.webhook_outbox_events
  WHERE delivery_status = 'pending'
    AND destination_url IS NOT NULL
    AND btrim(destination_url) <> ''
    AND next_attempt_at <= $1
    AND (lease_until IS NULL OR lease_until <= $1)
  ORDER BY created_at ASC, id ASC
  LIMIT $2
  FOR UPDATE SKIP LOCKED
)
UPDATE app.webhook_outbox_events AS e
SET
  lease_owner = $3,
  lease_until = $4,
  updated_at = $1
FROM candidates
WHERE e.id = candidates.id
RETURNING
  e.id,
  e.event_id,
  e.event_type,
  e.destination_url,
  e.payload,
  e.attempts,
  e.max_attempts
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
			"webhook_outbox_query_failed",
			"failed to claim webhook outbox events",
			map[string]any{"error": err.Error()},
		)
	}
	defer rows.Close()

	items := make([]dto.PendingWebhookOutboxEvent, 0, limit)
	for rows.Next() {
		item := dto.PendingWebhookOutboxEvent{}
		if err := rows.Scan(
			&item.ID,
			&item.EventID,
			&item.EventType,
			&item.DestinationURL,
			&item.Payload,
			&item.Attempts,
			&item.MaxAttempts,
		); err != nil {
			return nil, apperrors.NewInternal(
				"webhook_outbox_query_failed",
				"failed to parse claimed webhook outbox event",
				map[string]any{"error": err.Error()},
			)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternal(
			"webhook_outbox_query_failed",
			"failed while iterating claimed webhook outbox events",
			map[string]any{"error": err.Error()},
		)
	}

	return items, nil
}

func (r *Repository) MarkDelivered(
	ctx context.Context,
	id int64,
	leaseOwner string,
	deliveredAt time.Time,
) (bool, *apperrors.AppError) {
	const query = `
UPDATE app.webhook_outbox_events
SET
  delivery_status = 'delivered',
  delivered_at = $3,
  last_error = NULL,
  lease_owner = NULL,
  lease_until = NULL,
  updated_at = $3
WHERE id = $1
  AND delivery_status = 'pending'
  AND (lease_owner IS NULL OR lease_owner = $2)
`
	return execRowsAffected(ctx, r.db, query, id, strings.TrimSpace(leaseOwner), deliveredAt.UTC())
}

func (r *Repository) MarkRetry(
	ctx context.Context,
	id int64,
	leaseOwner string,
	attempts int,
	nextAttemptAt time.Time,
	lastError string,
	updatedAt time.Time,
) (bool, *apperrors.AppError) {
	const query = `
UPDATE app.webhook_outbox_events
SET
  attempts = $3,
  next_attempt_at = $4,
  last_error = $5,
  lease_owner = NULL,
  lease_until = NULL,
  updated_at = $6
WHERE id = $1
  AND delivery_status = 'pending'
  AND (lease_owner IS NULL OR lease_owner = $2)
`
	return execRowsAffected(
		ctx,
		r.db,
		query,
		id,
		strings.TrimSpace(leaseOwner),
		attempts,
		nextAttemptAt.UTC(),
		strings.TrimSpace(lastError),
		updatedAt.UTC(),
	)
}

func (r *Repository) MarkFailed(
	ctx context.Context,
	id int64,
	leaseOwner string,
	attempts int,
	lastError string,
	updatedAt time.Time,
) (bool, *apperrors.AppError) {
	const query = `
UPDATE app.webhook_outbox_events
SET
  delivery_status = 'failed',
  attempts = $3,
  last_error = $4,
  lease_owner = NULL,
  lease_until = NULL,
  updated_at = $5
WHERE id = $1
  AND delivery_status = 'pending'
  AND (lease_owner IS NULL OR lease_owner = $2)
`
	return execRowsAffected(
		ctx,
		r.db,
		query,
		id,
		strings.TrimSpace(leaseOwner),
		attempts,
		strings.TrimSpace(lastError),
		updatedAt.UTC(),
	)
}

func (r *Repository) RenewLease(
	ctx context.Context,
	id int64,
	leaseOwner string,
	leaseUntil time.Time,
	updatedAt time.Time,
) (bool, *apperrors.AppError) {
	const query = `
UPDATE app.webhook_outbox_events
SET
  lease_until = $3,
  updated_at = $4
WHERE id = $1
  AND delivery_status = 'pending'
  AND lease_owner = $2
`
	return execRowsAffected(
		ctx,
		r.db,
		query,
		id,
		strings.TrimSpace(leaseOwner),
		leaseUntil.UTC(),
		updatedAt.UTC(),
	)
}

func (r *Repository) RequeueFailedByEventID(
	ctx context.Context,
	eventID string,
	updatedAt time.Time,
) (dto.WebhookOutboxMutationResult, *apperrors.AppError) {
	const query = `
WITH selected AS (
  SELECT id, delivery_status
  FROM app.webhook_outbox_events
  WHERE event_id = $1
  FOR UPDATE
),
updated AS (
  UPDATE app.webhook_outbox_events AS e
  SET
    delivery_status = 'pending',
    attempts = 0,
    next_attempt_at = $2,
    last_error = NULL,
    delivered_at = NULL,
    lease_owner = NULL,
    lease_until = NULL,
    updated_at = $2
  FROM selected AS s
  WHERE e.id = s.id
    AND s.delivery_status = 'failed'
  RETURNING e.id
)
SELECT
  EXISTS(SELECT 1 FROM selected) AS found,
  COALESCE((SELECT delivery_status FROM selected LIMIT 1), '') AS current_status,
  EXISTS(SELECT 1 FROM updated) AS updated
`
	return runWebhookOutboxMutationWithStatus(ctx, r.db, query, strings.TrimSpace(eventID), updatedAt.UTC())
}

func (r *Repository) CancelByEventID(
	ctx context.Context,
	eventID string,
	lastError string,
	updatedAt time.Time,
) (dto.WebhookOutboxMutationResult, *apperrors.AppError) {
	const query = `
WITH selected AS (
  SELECT id, delivery_status
  FROM app.webhook_outbox_events
  WHERE event_id = $1
  FOR UPDATE
),
updated AS (
  UPDATE app.webhook_outbox_events AS e
  SET
    delivery_status = 'failed',
    last_error = $2,
    lease_owner = NULL,
    lease_until = NULL,
    updated_at = $3
  FROM selected AS s
  WHERE e.id = s.id
    AND s.delivery_status IN ('pending', 'failed')
  RETURNING e.id
)
SELECT
  EXISTS(SELECT 1 FROM selected) AS found,
  COALESCE((SELECT delivery_status FROM selected LIMIT 1), '') AS current_status,
  EXISTS(SELECT 1 FROM updated) AS updated
`
	return runWebhookOutboxMutationWithStatus(
		ctx,
		r.db,
		query,
		strings.TrimSpace(eventID),
		strings.TrimSpace(lastError),
		updatedAt.UTC(),
	)
}

func (r *Repository) GetOverview(
	ctx context.Context,
	now time.Time,
) (dto.WebhookOutboxOverview, *apperrors.AppError) {
	const query = `
SELECT
  COALESCE(SUM(CASE WHEN delivery_status = 'pending' THEN 1 ELSE 0 END), 0) AS pending_count,
  COALESCE(
    SUM(
      CASE
        WHEN delivery_status = 'pending'
         AND next_attempt_at <= $1
         AND (lease_until IS NULL OR lease_until <= $1)
        THEN 1
        ELSE 0
      END
    ),
    0
  ) AS pending_ready_count,
  COALESCE(SUM(CASE WHEN delivery_status = 'pending' AND attempts > 0 THEN 1 ELSE 0 END), 0) AS retrying_count,
  COALESCE(SUM(CASE WHEN delivery_status = 'failed' THEN 1 ELSE 0 END), 0) AS failed_count,
  COALESCE(SUM(CASE WHEN delivery_status = 'delivered' THEN 1 ELSE 0 END), 0) AS delivered_count,
  MIN(CASE WHEN delivery_status = 'pending' THEN created_at END) AS oldest_pending_created_at
FROM app.webhook_outbox_events
`

	output := dto.WebhookOutboxOverview{}
	var oldestPending sql.NullTime
	err := r.db.QueryRowContext(ctx, query, now.UTC()).Scan(
		&output.PendingCount,
		&output.PendingReadyCount,
		&output.RetryingCount,
		&output.FailedCount,
		&output.DeliveredCount,
		&oldestPending,
	)
	if err != nil {
		return dto.WebhookOutboxOverview{}, apperrors.NewInternal(
			"webhook_outbox_query_failed",
			"failed to query webhook outbox overview",
			map[string]any{"error": err.Error()},
		)
	}

	if oldestPending.Valid {
		oldest := oldestPending.Time.UTC()
		output.OldestPendingCreatedAt = &oldest
		ageSeconds := now.UTC().Sub(oldest).Seconds()
		if ageSeconds < 0 {
			ageSeconds = 0
		}
		age := int64(ageSeconds)
		output.OldestPendingAgeSec = &age
	}

	return output, nil
}

func (r *Repository) ListDLQ(
	ctx context.Context,
	limit int,
) ([]dto.WebhookDLQEvent, *apperrors.AppError) {
	const query = `
SELECT
  event_id,
  event_type,
  payment_request_id,
  destination_url,
  attempts,
  max_attempts,
  last_error,
  created_at,
  updated_at,
  delivered_at
FROM app.webhook_outbox_events
WHERE delivery_status = 'failed'
ORDER BY updated_at DESC, id DESC
LIMIT $1
`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, apperrors.NewInternal(
			"webhook_outbox_query_failed",
			"failed to list webhook dlq events",
			map[string]any{"error": err.Error()},
		)
	}
	defer rows.Close()

	output := make([]dto.WebhookDLQEvent, 0, limit)
	for rows.Next() {
		item := dto.WebhookDLQEvent{}
		var (
			lastError   sql.NullString
			deliveredAt sql.NullTime
		)
		if err := rows.Scan(
			&item.EventID,
			&item.EventType,
			&item.PaymentRequestID,
			&item.DestinationURL,
			&item.Attempts,
			&item.MaxAttempts,
			&lastError,
			&item.CreatedAt,
			&item.UpdatedAt,
			&deliveredAt,
		); err != nil {
			return nil, apperrors.NewInternal(
				"webhook_outbox_query_failed",
				"failed to parse webhook dlq event row",
				map[string]any{"error": err.Error()},
			)
		}
		if lastError.Valid {
			value := lastError.String
			item.LastError = &value
		}
		if deliveredAt.Valid {
			value := deliveredAt.Time.UTC()
			item.DeliveredAt = &value
		}
		output = append(output, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternal(
			"webhook_outbox_query_failed",
			"failed while iterating webhook dlq rows",
			map[string]any{"error": err.Error()},
		)
	}

	return output, nil
}

func execRowsAffected(
	ctx context.Context,
	db *sql.DB,
	query string,
	args ...any,
) (bool, *apperrors.AppError) {
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, apperrors.NewInternal(
			"webhook_outbox_update_failed",
			"failed to update webhook outbox event",
			map[string]any{"error": err.Error()},
		)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, apperrors.NewInternal(
			"webhook_outbox_update_failed",
			"failed to verify webhook outbox event update",
			map[string]any{"error": err.Error()},
		)
	}
	return rowsAffected == 1, nil
}

func runWebhookOutboxMutationWithStatus(
	ctx context.Context,
	db *sql.DB,
	query string,
	args ...any,
) (dto.WebhookOutboxMutationResult, *apperrors.AppError) {
	result := dto.WebhookOutboxMutationResult{}
	if err := db.QueryRowContext(ctx, query, args...).Scan(
		&result.Found,
		&result.CurrentStatus,
		&result.Updated,
	); err != nil {
		return dto.WebhookOutboxMutationResult{}, apperrors.NewInternal(
			"webhook_outbox_update_failed",
			"failed to update webhook outbox event",
			map[string]any{"error": err.Error()},
		)
	}
	result.CurrentStatus = strings.ToLower(strings.TrimSpace(result.CurrentStatus))
	return result, nil
}
