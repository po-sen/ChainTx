package use_cases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type dispatchWebhookEventsUseCase struct {
	repository portsout.WebhookOutboxRepository
	gateway    portsout.WebhookEventGateway
}

func NewDispatchWebhookEventsUseCase(
	repository portsout.WebhookOutboxRepository,
	gateway portsout.WebhookEventGateway,
) portsin.DispatchWebhookEventsUseCase {
	return &dispatchWebhookEventsUseCase{
		repository: repository,
		gateway:    gateway,
	}
}

func (u *dispatchWebhookEventsUseCase) Execute(
	ctx context.Context,
	command dto.DispatchWebhookEventsCommand,
) (dto.DispatchWebhookEventsOutput, *apperrors.AppError) {
	if u.repository == nil {
		return dto.DispatchWebhookEventsOutput{}, apperrors.NewInternal(
			"webhook_outbox_repository_missing",
			"webhook outbox repository is required",
			nil,
		)
	}
	if u.gateway == nil {
		return dto.DispatchWebhookEventsOutput{}, apperrors.NewInternal(
			"webhook_event_gateway_missing",
			"webhook event gateway is required",
			nil,
		)
	}
	if command.BatchSize <= 0 {
		return dto.DispatchWebhookEventsOutput{}, apperrors.NewValidation(
			"dispatch_webhook_batch_size_invalid",
			"dispatch webhook batch size must be greater than zero",
			map[string]any{"batch_size": command.BatchSize},
		)
	}
	workerID := strings.TrimSpace(command.WorkerID)
	if workerID == "" {
		return dto.DispatchWebhookEventsOutput{}, apperrors.NewValidation(
			"dispatch_webhook_worker_id_invalid",
			"dispatch webhook worker id is required",
			nil,
		)
	}
	if command.LeaseDuration <= 0 {
		return dto.DispatchWebhookEventsOutput{}, apperrors.NewValidation(
			"dispatch_webhook_lease_duration_invalid",
			"dispatch webhook lease duration must be greater than zero",
			map[string]any{"lease_duration": command.LeaseDuration.String()},
		)
	}
	if command.InitialBackoff <= 0 {
		return dto.DispatchWebhookEventsOutput{}, apperrors.NewValidation(
			"dispatch_webhook_initial_backoff_invalid",
			"dispatch webhook initial backoff must be greater than zero",
			map[string]any{"initial_backoff": command.InitialBackoff.String()},
		)
	}
	if command.MaxBackoff < command.InitialBackoff {
		return dto.DispatchWebhookEventsOutput{}, apperrors.NewValidation(
			"dispatch_webhook_max_backoff_invalid",
			"dispatch webhook max backoff must be greater than or equal to initial backoff",
			map[string]any{
				"initial_backoff": command.InitialBackoff.String(),
				"max_backoff":     command.MaxBackoff.String(),
			},
		)
	}

	startedAt := time.Now().UTC()
	now := command.Now.UTC()
	if command.Now.IsZero() {
		now = startedAt
	}
	leaseUntil := now.Add(command.LeaseDuration)

	rows, appErr := u.repository.ClaimPendingForDispatch(
		ctx,
		now,
		command.BatchSize,
		workerID,
		leaseUntil,
	)
	if appErr != nil {
		return dto.DispatchWebhookEventsOutput{}, appErr
	}

	output := dto.DispatchWebhookEventsOutput{
		Claimed: len(rows),
	}
	for _, row := range rows {
		destinationURL := strings.TrimSpace(row.DestinationURL)
		if destinationURL == "" {
			output.Errors++
			output.Skipped++
			continue
		}

		sendOutput, sendErr := u.gateway.SendWebhookEvent(ctx, dto.SendWebhookEventInput{
			EventID:        row.EventID,
			EventType:      row.EventType,
			DestinationURL: destinationURL,
			Payload:        row.Payload,
		})
		if sendErr == nil && sendOutput.StatusCode >= 200 && sendOutput.StatusCode <= 299 {
			updated, deliveredErr := u.repository.MarkDelivered(ctx, row.ID, workerID, now)
			if deliveredErr != nil {
				return output, deliveredErr
			}
			if updated {
				output.Sent++
			} else {
				output.Skipped++
			}
			continue
		}

		output.Errors++
		nextAttempts := row.Attempts + 1
		errorMessage := webhookDispatchErrorMessage(sendErr, sendOutput.StatusCode)
		if nextAttempts >= row.MaxAttempts {
			updated, markErr := u.repository.MarkFailed(
				ctx,
				row.ID,
				workerID,
				nextAttempts,
				errorMessage,
				now,
			)
			if markErr != nil {
				return output, markErr
			}
			if updated {
				output.Failed++
			} else {
				output.Skipped++
			}
			continue
		}

		backoff := webhookRetryBackoff(nextAttempts, command.InitialBackoff, command.MaxBackoff)
		nextAttemptAt := now.Add(backoff)
		updated, markErr := u.repository.MarkRetry(
			ctx,
			row.ID,
			workerID,
			nextAttempts,
			nextAttemptAt,
			errorMessage,
			now,
		)
		if markErr != nil {
			return output, markErr
		}
		if updated {
			output.Retried++
		} else {
			output.Skipped++
		}
	}

	output.LatencyMS = time.Since(startedAt).Milliseconds()
	return output, nil
}

func webhookDispatchErrorMessage(appErr *apperrors.AppError, statusCode int) string {
	if appErr != nil {
		message := strings.TrimSpace(appErr.Message)
		if message == "" {
			message = strings.TrimSpace(appErr.Code)
		}
		if message == "" {
			message = "webhook dispatch failed"
		}
		return message
	}

	if statusCode <= 0 {
		return "webhook dispatch failed"
	}
	return fmt.Sprintf("webhook endpoint returned status %d", statusCode)
}

func webhookRetryBackoff(attempts int, initial time.Duration, max time.Duration) time.Duration {
	if attempts <= 1 {
		return initial
	}

	backoff := initial
	for i := 1; i < attempts; i++ {
		if backoff >= max {
			return max
		}
		backoff *= 2
		if backoff > max {
			return max
		}
	}

	return backoff
}
