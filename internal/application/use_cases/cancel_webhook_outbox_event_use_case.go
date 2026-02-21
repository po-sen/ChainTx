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

type cancelWebhookOutboxEventUseCase struct {
	repository portsout.WebhookOutboxRepository
}

func NewCancelWebhookOutboxEventUseCase(repository portsout.WebhookOutboxRepository) portsin.CancelWebhookOutboxEventUseCase {
	return &cancelWebhookOutboxEventUseCase{repository: repository}
}

func (u *cancelWebhookOutboxEventUseCase) Execute(
	ctx context.Context,
	command dto.CancelWebhookOutboxEventCommand,
) (dto.CancelWebhookOutboxEventOutput, *apperrors.AppError) {
	if u.repository == nil {
		return dto.CancelWebhookOutboxEventOutput{}, apperrors.NewInternal(
			"webhook_outbox_repository_missing",
			"webhook outbox repository is required",
			nil,
		)
	}

	eventID := strings.TrimSpace(command.EventID)
	if eventID == "" {
		return dto.CancelWebhookOutboxEventOutput{}, apperrors.NewValidation(
			"invalid_request",
			"event_id is required",
			map[string]any{"field": "event_id"},
		)
	}

	now := command.Now.UTC()
	if command.Now.IsZero() {
		now = time.Now().UTC()
	}

	lastError := normalizeWebhookManualCancelReason(command.Reason)
	result, appErr := u.repository.CancelByEventID(ctx, eventID, lastError, now)
	if appErr != nil {
		return dto.CancelWebhookOutboxEventOutput{}, appErr
	}
	if !result.Found {
		return dto.CancelWebhookOutboxEventOutput{}, apperrors.NewNotFound(
			"webhook_outbox_event_not_found",
			"webhook outbox event was not found",
			map[string]any{"event_id": eventID},
		)
	}
	if !result.Updated {
		return dto.CancelWebhookOutboxEventOutput{}, apperrors.NewConflict(
			"webhook_outbox_event_not_cancellable",
			"webhook outbox event is not cancellable",
			map[string]any{
				"event_id":        eventID,
				"delivery_status": result.CurrentStatus,
			},
		)
	}

	return dto.CancelWebhookOutboxEventOutput{
		EventID:        eventID,
		DeliveryStatus: "failed",
		LastError:      lastError,
		UpdatedAt:      now,
	}, nil
}

func normalizeWebhookManualCancelReason(reason string) string {
	normalized := strings.TrimSpace(reason)
	if normalized == "" {
		return "manual_cancelled"
	}
	return "manual_cancelled: " + normalized
}
