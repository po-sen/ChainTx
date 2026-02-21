package out

import (
	"context"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type WebhookOutboxRepository interface {
	ClaimPendingForDispatch(
		ctx context.Context,
		now time.Time,
		limit int,
		leaseOwner string,
		leaseUntil time.Time,
	) ([]dto.PendingWebhookOutboxEvent, *apperrors.AppError)
	MarkDelivered(
		ctx context.Context,
		id int64,
		leaseOwner string,
		deliveredAt time.Time,
	) (bool, *apperrors.AppError)
	MarkRetry(
		ctx context.Context,
		id int64,
		leaseOwner string,
		attempts int,
		nextAttemptAt time.Time,
		lastError string,
		updatedAt time.Time,
	) (bool, *apperrors.AppError)
	MarkFailed(
		ctx context.Context,
		id int64,
		leaseOwner string,
		attempts int,
		lastError string,
		updatedAt time.Time,
	) (bool, *apperrors.AppError)
}
