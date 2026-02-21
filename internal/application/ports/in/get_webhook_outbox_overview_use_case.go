package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type GetWebhookOutboxOverviewUseCase interface {
	Execute(ctx context.Context, query dto.GetWebhookOutboxOverviewQuery) (dto.WebhookOutboxOverview, *apperrors.AppError)
}
