package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type DispatchWebhookEventsUseCase interface {
	Execute(
		ctx context.Context,
		command dto.DispatchWebhookEventsCommand,
	) (dto.DispatchWebhookEventsOutput, *apperrors.AppError)
}
