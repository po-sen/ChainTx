package use_cases

import (
	"context"
	"strings"
	"time"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type requeueWebhookDLQEventUseCase struct {
	repository portsout.WebhookOutboxRepository
}

func NewRequeueWebhookDLQEventUseCase(repository portsout.WebhookOutboxRepository) portsin.RequeueWebhookDLQEventUseCase {
	return &requeueWebhookDLQEventUseCase{repository: repository}
}

func (u *requeueWebhookDLQEventUseCase) Execute(
	ctx context.Context,
	command dto.RequeueWebhookDLQEventCommand,
) (dto.RequeueWebhookDLQEventOutput, *apperrors.AppError) {
	if u.repository == nil {
		return dto.RequeueWebhookDLQEventOutput{}, apperrors.NewInternal(
			"webhook_outbox_repository_missing",
			"webhook outbox repository is required",
			nil,
		)
	}

	eventID := strings.TrimSpace(command.EventID)
	if eventID == "" {
		return dto.RequeueWebhookDLQEventOutput{}, apperrors.NewValidation(
			"invalid_request",
			"event_id is required",
			map[string]any{"field": "event_id"},
		)
	}
	operatorID := strings.TrimSpace(command.OperatorID)
	if operatorID == "" {
		return dto.RequeueWebhookDLQEventOutput{}, apperrors.NewValidation(
			"invalid_request",
			"x_principal_id is required",
			map[string]any{"field": "x_principal_id"},
		)
	}

	now := command.Now.UTC()
	if command.Now.IsZero() {
		now = time.Now().UTC()
	}

	result, appErr := u.repository.RequeueFailedByEventID(ctx, eventID, operatorID, now)
	if appErr != nil {
		return dto.RequeueWebhookDLQEventOutput{}, appErr
	}
	if !result.Found {
		return dto.RequeueWebhookDLQEventOutput{}, apperrors.NewNotFound(
			"webhook_outbox_event_not_found",
			"webhook outbox event was not found",
			map[string]any{"event_id": eventID},
		)
	}
	if !result.Updated {
		return dto.RequeueWebhookDLQEventOutput{}, apperrors.NewConflict(
			"webhook_outbox_event_not_requeueable",
			"webhook outbox event is not requeueable",
			map[string]any{
				"event_id":        eventID,
				"delivery_status": result.CurrentStatus,
			},
		)
	}

	return dto.RequeueWebhookDLQEventOutput{
		EventID:        eventID,
		DeliveryStatus: "pending",
		UpdatedAt:      now,
	}, nil
}
