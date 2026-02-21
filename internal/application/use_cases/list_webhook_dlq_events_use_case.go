package use_cases

import (
	"context"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

const (
	defaultWebhookDLQLimit = 50
	maxWebhookDLQLimit     = 200
)

type listWebhookDLQEventsUseCase struct {
	readModel portsout.WebhookOutboxReadModel
}

func NewListWebhookDLQEventsUseCase(readModel portsout.WebhookOutboxReadModel) portsin.ListWebhookDLQEventsUseCase {
	return &listWebhookDLQEventsUseCase{readModel: readModel}
}

func (u *listWebhookDLQEventsUseCase) Execute(
	ctx context.Context,
	query dto.ListWebhookDLQEventsQuery,
) (dto.ListWebhookDLQEventsOutput, *apperrors.AppError) {
	if u.readModel == nil {
		return dto.ListWebhookDLQEventsOutput{}, apperrors.NewInternal(
			"webhook_outbox_read_model_missing",
			"webhook outbox read model is required",
			nil,
		)
	}

	limit := query.Limit
	if limit == 0 {
		limit = defaultWebhookDLQLimit
	}
	if limit < 1 || limit > maxWebhookDLQLimit {
		return dto.ListWebhookDLQEventsOutput{}, apperrors.NewValidation(
			"invalid_request",
			"limit must be between 1 and 200",
			map[string]any{"field": "limit"},
		)
	}

	events, appErr := u.readModel.ListDLQ(ctx, limit)
	if appErr != nil {
		return dto.ListWebhookDLQEventsOutput{}, appErr
	}

	return dto.ListWebhookDLQEventsOutput{Events: events}, nil
}
