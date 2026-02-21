package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type CancelWebhookOutboxEventUseCase interface {
	Execute(ctx context.Context, command dto.CancelWebhookOutboxEventCommand) (dto.CancelWebhookOutboxEventOutput, *apperrors.AppError)
}
