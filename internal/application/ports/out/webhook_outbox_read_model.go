package out

import (
	"context"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type WebhookOutboxReadModel interface {
	GetOverview(ctx context.Context, now time.Time) (dto.WebhookOutboxOverview, *apperrors.AppError)
	ListDLQ(ctx context.Context, limit int) ([]dto.WebhookDLQEvent, *apperrors.AppError)
}
