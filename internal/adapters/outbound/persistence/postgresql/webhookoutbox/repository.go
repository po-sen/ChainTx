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
