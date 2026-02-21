package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type RequeueWebhookDLQEventUseCase interface {
	Execute(ctx context.Context, command dto.RequeueWebhookDLQEventCommand) (dto.RequeueWebhookDLQEventOutput, *apperrors.AppError)
}
