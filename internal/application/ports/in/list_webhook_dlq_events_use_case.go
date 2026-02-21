package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type ListWebhookDLQEventsUseCase interface {
	Execute(ctx context.Context, query dto.ListWebhookDLQEventsQuery) (dto.ListWebhookDLQEventsOutput, *apperrors.AppError)
}
