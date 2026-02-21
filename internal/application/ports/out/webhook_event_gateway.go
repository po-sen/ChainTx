package out

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type WebhookEventGateway interface {
	SendWebhookEvent(
		ctx context.Context,
		input dto.SendWebhookEventInput,
	) (dto.SendWebhookEventOutput, *apperrors.AppError)
}
