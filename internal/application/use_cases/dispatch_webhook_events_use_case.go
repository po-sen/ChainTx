package use_cases

import (
	"context"
	"fmt"
	"hash/fnv"
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
	if command.RetryJitterBPS < 0 || command.RetryJitterBPS > 10000 {
		return dto.DispatchWebhookEventsOutput{}, apperrors.NewValidation(
			"dispatch_webhook_retry_jitter_bps_invalid",
			"dispatch webhook retry jitter bps must be between 0 and 10000",
			map[string]any{"retry_jitter_bps": command.RetryJitterBPS},
		)
	}
	if command.RetryBudget < 0 {
		return dto.DispatchWebhookEventsOutput{}, apperrors.NewValidation(
			"dispatch_webhook_retry_budget_invalid",
			"dispatch webhook retry budget must be a non-negative integer",
			map[string]any{"retry_budget": command.RetryBudget},
		)
	}
	heartbeatInterval, heartbeatIntervalErr := webhookLeaseHeartbeatInterval(command.LeaseDuration)
	if heartbeatIntervalErr != nil {
		return dto.DispatchWebhookEventsOutput{}, heartbeatIntervalErr
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
		deliveryAttempt := row.Attempts + 1
		if deliveryAttempt <= 0 {
			deliveryAttempt = 1
		}

		heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
		heartbeatErrCh := make(chan *apperrors.AppError, 1)
		heartbeatDoneCh := make(chan struct{})
		go func(eventID string, id int64) {
			defer close(heartbeatDoneCh)
			u.runLeaseHeartbeat(
				heartbeatCtx,
				eventID,
				id,
				workerID,
				command.LeaseDuration,
				heartbeatInterval,
				heartbeatErrCh,
			)
		}(row.EventID, row.ID)

		sendOutput, sendErr := u.gateway.SendWebhookEvent(ctx, dto.SendWebhookEventInput{
			EventID:         row.EventID,
			EventType:       row.EventType,
			DeliveryAttempt: deliveryAttempt,
			DestinationURL:  destinationURL,
			Payload:         row.Payload,
		})
		stopHeartbeat()
		<-heartbeatDoneCh
		heartbeatErr := drainWebhookHeartbeatError(heartbeatErrCh)
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
			if heartbeatErr != nil {
				return output, heartbeatErr
			}
			continue
		}

		output.Errors++
		nextAttempts := row.Attempts + 1
		effectiveMaxAttempts := webhookEffectiveMaxAttempts(row.MaxAttempts, command.RetryBudget)
		errorMessage := webhookDispatchErrorMessage(sendErr, sendOutput.StatusCode)
		if nextAttempts >= effectiveMaxAttempts {
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
			if heartbeatErr != nil {
				return output, heartbeatErr
			}
			continue
		}

		backoff := webhookRetryBackoffWithJitter(
			nextAttempts,
			command.InitialBackoff,
			command.MaxBackoff,
			command.RetryJitterBPS,
			row.EventID,
			row.ID,
		)
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
		if heartbeatErr != nil {
			return output, heartbeatErr
		}
	}

	output.LatencyMS = time.Since(startedAt).Milliseconds()
	return output, nil
}

func (u *dispatchWebhookEventsUseCase) runLeaseHeartbeat(
	ctx context.Context,
	eventID string,
	id int64,
	workerID string,
	leaseDuration time.Duration,
	interval time.Duration,
	errorCh chan<- *apperrors.AppError,
) {
	renewAt := time.Now().UTC()
	if appErr := u.renewWebhookLease(ctx, eventID, id, workerID, leaseDuration, renewAt); appErr != nil {
		reportWebhookHeartbeatError(errorCh, appErr)
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case tickAt := <-ticker.C:
			if appErr := u.renewWebhookLease(
				ctx,
				eventID,
				id,
				workerID,
				leaseDuration,
				tickAt.UTC(),
			); appErr != nil {
				reportWebhookHeartbeatError(errorCh, appErr)
				return
			}
		}
	}
}

func (u *dispatchWebhookEventsUseCase) renewWebhookLease(
	ctx context.Context,
	eventID string,
	id int64,
	workerID string,
	leaseDuration time.Duration,
	updatedAt time.Time,
) *apperrors.AppError {
	renewed, renewErr := u.repository.RenewLease(
		ctx,
		id,
		workerID,
		updatedAt.Add(leaseDuration),
		updatedAt,
	)
	if renewErr != nil {
		return apperrors.NewInternal(
			"dispatch_webhook_lease_renew_failed",
			"failed to renew webhook outbox lease",
			map[string]any{
				"event_id":  eventID,
				"row_id":    id,
				"worker_id": workerID,
				"error":     renewErr.Message,
			},
		)
	}
	if !renewed {
		return apperrors.NewInternal(
			"dispatch_webhook_lease_lost",
			"webhook outbox lease ownership was lost during dispatch",
			map[string]any{
				"event_id":  eventID,
				"row_id":    id,
				"worker_id": workerID,
			},
		)
	}
	return nil
}

func webhookLeaseHeartbeatInterval(leaseDuration time.Duration) (time.Duration, *apperrors.AppError) {
	interval := leaseDuration / 3
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	if interval >= leaseDuration {
		interval = leaseDuration / 2
	}
	if interval <= 0 || interval >= leaseDuration {
		return 0, apperrors.NewValidation(
			"dispatch_webhook_lease_heartbeat_interval_invalid",
			"dispatch webhook lease duration is too small for heartbeat interval",
			map[string]any{"lease_duration": leaseDuration.String()},
		)
	}
	return interval, nil
}

func reportWebhookHeartbeatError(errorCh chan<- *apperrors.AppError, appErr *apperrors.AppError) {
	if appErr == nil {
		return
	}
	select {
	case errorCh <- appErr:
	default:
	}
}

func drainWebhookHeartbeatError(errorCh <-chan *apperrors.AppError) *apperrors.AppError {
	select {
	case appErr := <-errorCh:
		return appErr
	default:
		return nil
	}
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

func webhookEffectiveMaxAttempts(rowMaxAttempts int, retryBudget int) int {
	effective := rowMaxAttempts
	if retryBudget <= 0 {
		return effective
	}
	budgetMaxAttempts := retryBudget + 1
	if budgetMaxAttempts < effective {
		effective = budgetMaxAttempts
	}
	return effective
}

func webhookRetryBackoffWithJitter(
	attempts int,
	initial time.Duration,
	max time.Duration,
	jitterBPS int,
	eventID string,
	rowID int64,
) time.Duration {
	base := webhookRetryBackoff(attempts, initial, max)
	if jitterBPS <= 0 || base <= 0 {
		return base
	}

	offsetBPS := webhookDeterministicJitterOffsetBPS(eventID, rowID, attempts, jitterBPS)
	factorBPS := 10000 + offsetBPS
	if factorBPS < 1 {
		factorBPS = 1
	}
	jitteredNanos := int64(base) * int64(factorBPS) / 10000
	if jitteredNanos <= 0 {
		jitteredNanos = 1
	}
	jittered := time.Duration(jitteredNanos)
	if jittered > max {
		return max
	}
	return jittered
}

func webhookDeterministicJitterOffsetBPS(
	eventID string,
	rowID int64,
	attempt int,
	jitterBPS int,
) int {
	if jitterBPS <= 0 {
		return 0
	}
	span := (jitterBPS * 2) + 1
	if span <= 1 {
		return 0
	}

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(strings.TrimSpace(eventID)))
	_, _ = hasher.Write([]byte("|"))
	_, _ = hasher.Write([]byte(fmt.Sprintf("%d", rowID)))
	_, _ = hasher.Write([]byte("|"))
	_, _ = hasher.Write([]byte(fmt.Sprintf("%d", attempt)))
	value := int(hasher.Sum32() % uint32(span))
	return value - jitterBPS
}
