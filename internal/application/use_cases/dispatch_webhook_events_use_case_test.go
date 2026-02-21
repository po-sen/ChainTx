//go:build !integration

package use_cases

import (
	"context"
	"sync"
	"testing"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

func TestDispatchWebhookEventsUseCaseValidatesInput(t *testing.T) {
	useCase := NewDispatchWebhookEventsUseCase(
		&fakeWebhookOutboxRepository{},
		&fakeWebhookEventGateway{},
	)

	_, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		BatchSize: 0,
	})
	if appErr == nil || appErr.Code != "dispatch_webhook_batch_size_invalid" {
		t.Fatalf("expected dispatch_webhook_batch_size_invalid, got %+v", appErr)
	}
}

func TestDispatchWebhookEventsUseCaseRejectsInvalidRetryJitterBPS(t *testing.T) {
	useCase := NewDispatchWebhookEventsUseCase(
		&fakeWebhookOutboxRepository{},
		&fakeWebhookEventGateway{},
	)

	_, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		Now:            time.Now().UTC(),
		BatchSize:      10,
		WorkerID:       "webhook-worker-a",
		LeaseDuration:  30 * time.Second,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     60 * time.Second,
		RetryJitterBPS: 10001,
	})
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
	if appErr.Code != "dispatch_webhook_retry_jitter_bps_invalid" {
		t.Fatalf("expected dispatch_webhook_retry_jitter_bps_invalid, got %s", appErr.Code)
	}
}

func TestDispatchWebhookEventsUseCaseRejectsNegativeRetryBudget(t *testing.T) {
	useCase := NewDispatchWebhookEventsUseCase(
		&fakeWebhookOutboxRepository{},
		&fakeWebhookEventGateway{},
	)

	_, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		Now:            time.Now().UTC(),
		BatchSize:      10,
		WorkerID:       "webhook-worker-a",
		LeaseDuration:  30 * time.Second,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     60 * time.Second,
		RetryBudget:    -1,
	})
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
	if appErr.Code != "dispatch_webhook_retry_budget_invalid" {
		t.Fatalf("expected dispatch_webhook_retry_budget_invalid, got %s", appErr.Code)
	}
}

func TestDispatchWebhookEventsUseCaseMarksDeliveredOnSuccess(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	repo := &fakeWebhookOutboxRepository{
		claimed: []dto.PendingWebhookOutboxEvent{
			{
				ID:             1,
				EventID:        "evt_1",
				EventType:      "payment_request.status_changed",
				DestinationURL: "https://hooks.example.com/evt_1",
				Payload:        []byte(`{"event_id":"evt_1"}`),
				Attempts:       0,
				MaxAttempts:    3,
			},
		},
	}
	gateway := &fakeWebhookEventGateway{
		results: map[string]dto.SendWebhookEventOutput{
			"evt_1": {StatusCode: 204},
		},
	}
	useCase := NewDispatchWebhookEventsUseCase(repo, gateway)

	output, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		Now:            now,
		BatchSize:      10,
		WorkerID:       "webhook-worker-a",
		LeaseDuration:  30 * time.Second,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     60 * time.Second,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if output.Claimed != 1 || output.Sent != 1 {
		t.Fatalf("expected claimed=1 sent=1, got %+v", output)
	}
	if output.HTTP2xxCount != 1 || output.HTTP4xxCount != 0 || output.HTTP5xxCount != 0 || output.NetworkErrorCount != 0 {
		t.Fatalf("expected 2xx=1 4xx=0 5xx=0 network=0, got %+v", output)
	}
	if len(repo.delivered) != 1 || repo.delivered[0].id != 1 {
		t.Fatalf("expected delivered id=1, got %+v", repo.delivered)
	}
}

func TestDispatchWebhookEventsUseCasePassesDeliveryAttempt(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	repo := &fakeWebhookOutboxRepository{
		claimed: []dto.PendingWebhookOutboxEvent{
			{
				ID:             10,
				EventID:        "evt_10",
				EventType:      "payment_request.status_changed",
				DestinationURL: "https://hooks.example.com/evt_10",
				Payload:        []byte(`{"event_id":"evt_10"}`),
				Attempts:       2,
				MaxAttempts:    5,
			},
		},
	}
	gateway := &fakeWebhookEventGateway{
		results: map[string]dto.SendWebhookEventOutput{
			"evt_10": {StatusCode: 204},
		},
	}
	useCase := NewDispatchWebhookEventsUseCase(repo, gateway)

	_, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		Now:            now,
		BatchSize:      10,
		WorkerID:       "webhook-worker-a",
		LeaseDuration:  30 * time.Second,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     60 * time.Second,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	inputs := gateway.sentInputs()
	if len(inputs) != 1 {
		t.Fatalf("expected one webhook send input, got %d", len(inputs))
	}
	if inputs[0].DeliveryAttempt != 3 {
		t.Fatalf("expected delivery attempt 3, got %d", inputs[0].DeliveryAttempt)
	}
}

func TestDispatchWebhookEventsUseCaseRetriesOnFailure(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	repo := &fakeWebhookOutboxRepository{
		claimed: []dto.PendingWebhookOutboxEvent{
			{
				ID:             2,
				EventID:        "evt_2",
				EventType:      "payment_request.status_changed",
				DestinationURL: "https://hooks.example.com/evt_2",
				Payload:        []byte(`{"event_id":"evt_2"}`),
				Attempts:       0,
				MaxAttempts:    3,
			},
		},
	}
	gateway := &fakeWebhookEventGateway{
		errors: map[string]*apperrors.AppError{
			"evt_2": apperrors.NewInternal("webhook_http_failed", "endpoint timeout", nil),
		},
	}
	useCase := NewDispatchWebhookEventsUseCase(repo, gateway)

	output, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		Now:            now,
		BatchSize:      10,
		WorkerID:       "webhook-worker-a",
		LeaseDuration:  30 * time.Second,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     60 * time.Second,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if output.Retried != 1 || output.Failed != 0 {
		t.Fatalf("expected retried=1 failed=0, got %+v", output)
	}
	if output.NetworkErrorCount != 1 || output.HTTP2xxCount != 0 || output.HTTP4xxCount != 0 || output.HTTP5xxCount != 0 {
		t.Fatalf("expected network=1 only, got %+v", output)
	}
	if len(repo.retried) != 1 {
		t.Fatalf("expected one retry update, got %+v", repo.retried)
	}
	if repo.retried[0].attempts != 1 {
		t.Fatalf("expected attempts=1, got %+v", repo.retried[0])
	}
}

func TestDispatchWebhookEventsUseCaseFailsWhenRetryBudgetReached(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	repo := &fakeWebhookOutboxRepository{
		claimed: []dto.PendingWebhookOutboxEvent{
			{
				ID:             20,
				EventID:        "evt_20",
				EventType:      "payment_request.status_changed",
				DestinationURL: "https://hooks.example.com/evt_20",
				Payload:        []byte(`{"event_id":"evt_20"}`),
				Attempts:       2,
				MaxAttempts:    8,
			},
		},
	}
	gateway := &fakeWebhookEventGateway{
		results: map[string]dto.SendWebhookEventOutput{
			"evt_20": {StatusCode: 500},
		},
	}
	useCase := NewDispatchWebhookEventsUseCase(repo, gateway)

	output, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		Now:            now,
		BatchSize:      10,
		WorkerID:       "webhook-worker-a",
		LeaseDuration:  30 * time.Second,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     60 * time.Second,
		RetryBudget:    2,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if output.Failed != 1 || output.Retried != 0 {
		t.Fatalf("expected failed=1 retried=0, got %+v", output)
	}
	if len(repo.failed) != 1 || repo.failed[0].attempts != 3 {
		t.Fatalf("expected failed attempts=3, got %+v", repo.failed)
	}
}

func TestDispatchWebhookEventsUseCaseMarksFailedAtMaxAttempts(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	repo := &fakeWebhookOutboxRepository{
		claimed: []dto.PendingWebhookOutboxEvent{
			{
				ID:             3,
				EventID:        "evt_3",
				EventType:      "payment_request.status_changed",
				DestinationURL: "https://hooks.example.com/evt_3",
				Payload:        []byte(`{"event_id":"evt_3"}`),
				Attempts:       2,
				MaxAttempts:    3,
			},
		},
	}
	gateway := &fakeWebhookEventGateway{
		results: map[string]dto.SendWebhookEventOutput{
			"evt_3": {StatusCode: 500},
		},
	}
	useCase := NewDispatchWebhookEventsUseCase(repo, gateway)

	output, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		Now:            now,
		BatchSize:      10,
		WorkerID:       "webhook-worker-a",
		LeaseDuration:  30 * time.Second,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     60 * time.Second,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if output.Failed != 1 || output.Retried != 0 {
		t.Fatalf("expected failed=1 retried=0, got %+v", output)
	}
	if output.HTTP5xxCount != 1 || output.HTTP2xxCount != 0 || output.HTTP4xxCount != 0 || output.NetworkErrorCount != 0 {
		t.Fatalf("expected 5xx=1 only, got %+v", output)
	}
	if len(repo.failed) != 1 || repo.failed[0].attempts != 3 {
		t.Fatalf("expected terminal failure attempts=3, got %+v", repo.failed)
	}
}

func TestDispatchWebhookEventsUseCaseCounts4xxBucket(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	repo := &fakeWebhookOutboxRepository{
		claimed: []dto.PendingWebhookOutboxEvent{
			{
				ID:             31,
				EventID:        "evt_31",
				EventType:      "payment_request.status_changed",
				DestinationURL: "https://hooks.example.com/evt_31",
				Payload:        []byte(`{"event_id":"evt_31"}`),
				Attempts:       0,
				MaxAttempts:    3,
			},
		},
	}
	gateway := &fakeWebhookEventGateway{
		results: map[string]dto.SendWebhookEventOutput{
			"evt_31": {StatusCode: 429},
		},
	}
	useCase := NewDispatchWebhookEventsUseCase(repo, gateway)

	output, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		Now:            now,
		BatchSize:      10,
		WorkerID:       "webhook-worker-a",
		LeaseDuration:  30 * time.Second,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     60 * time.Second,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if output.HTTP4xxCount != 1 || output.HTTP2xxCount != 0 || output.HTTP5xxCount != 0 || output.NetworkErrorCount != 0 {
		t.Fatalf("expected 4xx=1 only, got %+v", output)
	}
	if output.Retried != 1 {
		t.Fatalf("expected retried=1, got %+v", output)
	}
}

func TestDispatchWebhookEventsUseCaseRenewsLeaseDuringSlowSend(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	repo := &fakeWebhookOutboxRepository{
		claimed: []dto.PendingWebhookOutboxEvent{
			{
				ID:             4,
				EventID:        "evt_4",
				EventType:      "payment_request.status_changed",
				DestinationURL: "https://hooks.example.com/evt_4",
				Payload:        []byte(`{"event_id":"evt_4"}`),
				Attempts:       0,
				MaxAttempts:    3,
			},
		},
	}
	gateway := &fakeWebhookEventGateway{
		results: map[string]dto.SendWebhookEventOutput{
			"evt_4": {StatusCode: 204},
		},
		sendDelay: 220 * time.Millisecond,
	}
	useCase := NewDispatchWebhookEventsUseCase(repo, gateway)

	_, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		Now:            now,
		BatchSize:      10,
		WorkerID:       "webhook-worker-a",
		LeaseDuration:  120 * time.Millisecond,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     60 * time.Second,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if repo.renewCount() < 2 {
		t.Fatalf("expected at least 2 lease renewals, got %d", repo.renewCount())
	}
}

func TestDispatchWebhookEventsUseCaseReturnsRenewLeaseError(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	repo := &fakeWebhookOutboxRepository{
		claimed: []dto.PendingWebhookOutboxEvent{
			{
				ID:             5,
				EventID:        "evt_5",
				EventType:      "payment_request.status_changed",
				DestinationURL: "https://hooks.example.com/evt_5",
				Payload:        []byte(`{"event_id":"evt_5"}`),
				Attempts:       0,
				MaxAttempts:    3,
			},
		},
		renewErr: apperrors.NewInternal("webhook_outbox_update_failed", "db write error", nil),
	}
	gateway := &fakeWebhookEventGateway{
		results: map[string]dto.SendWebhookEventOutput{
			"evt_5": {StatusCode: 204},
		},
	}
	useCase := NewDispatchWebhookEventsUseCase(repo, gateway)

	_, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		Now:            now,
		BatchSize:      10,
		WorkerID:       "webhook-worker-a",
		LeaseDuration:  30 * time.Second,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     60 * time.Second,
	})
	if appErr == nil {
		t.Fatalf("expected renewal error")
	}
	if appErr.Code != "dispatch_webhook_lease_renew_failed" {
		t.Fatalf("expected dispatch_webhook_lease_renew_failed, got %s", appErr.Code)
	}
}

func TestDispatchWebhookEventsUseCaseReturnsLeaseLostWhenRenewReturnsFalse(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	repo := &fakeWebhookOutboxRepository{
		claimed: []dto.PendingWebhookOutboxEvent{
			{
				ID:             6,
				EventID:        "evt_6",
				EventType:      "payment_request.status_changed",
				DestinationURL: "https://hooks.example.com/evt_6",
				Payload:        []byte(`{"event_id":"evt_6"}`),
				Attempts:       0,
				MaxAttempts:    3,
			},
		},
		renewByIDPass: map[int64]bool{
			6: false,
		},
	}
	gateway := &fakeWebhookEventGateway{
		results: map[string]dto.SendWebhookEventOutput{
			"evt_6": {StatusCode: 204},
		},
	}
	useCase := NewDispatchWebhookEventsUseCase(repo, gateway)

	_, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		Now:            now,
		BatchSize:      10,
		WorkerID:       "webhook-worker-a",
		LeaseDuration:  30 * time.Second,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     60 * time.Second,
	})
	if appErr == nil {
		t.Fatalf("expected lease lost error")
	}
	if appErr.Code != "dispatch_webhook_lease_lost" {
		t.Fatalf("expected dispatch_webhook_lease_lost, got %s", appErr.Code)
	}
}

func TestDispatchWebhookEventsUseCaseRejectsLeaseTooSmallForHeartbeat(t *testing.T) {
	useCase := NewDispatchWebhookEventsUseCase(
		&fakeWebhookOutboxRepository{},
		&fakeWebhookEventGateway{},
	)

	_, appErr := useCase.Execute(context.Background(), dto.DispatchWebhookEventsCommand{
		Now:            time.Now().UTC(),
		BatchSize:      10,
		WorkerID:       "webhook-worker-a",
		LeaseDuration:  time.Nanosecond,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     60 * time.Second,
	})
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
	if appErr.Code != "dispatch_webhook_lease_heartbeat_interval_invalid" {
		t.Fatalf("expected dispatch_webhook_lease_heartbeat_interval_invalid, got %s", appErr.Code)
	}
}

func TestWebhookRetryBackoffWithJitterDisabledMatchesBase(t *testing.T) {
	base := webhookRetryBackoff(3, 5*time.Second, 60*time.Second)
	jittered := webhookRetryBackoffWithJitter(
		3,
		5*time.Second,
		60*time.Second,
		0,
		"evt_1",
		1,
	)
	if jittered != base {
		t.Fatalf("expected jitter disabled backoff %s, got %s", base, jittered)
	}
}

func TestWebhookRetryBackoffWithJitterWithinBounds(t *testing.T) {
	base := webhookRetryBackoff(3, 5*time.Second, 60*time.Second)
	jittered := webhookRetryBackoffWithJitter(
		3,
		5*time.Second,
		60*time.Second,
		2000,
		"evt_jitter",
		99,
	)
	min := base * 80 / 100
	max := base * 120 / 100
	if jittered < min || jittered > max {
		t.Fatalf("expected jittered backoff in [%s,%s], got %s", min, max, jittered)
	}
}

type fakeWebhookOutboxRepository struct {
	mu sync.Mutex

	claimed []dto.PendingWebhookOutboxEvent

	delivered []fakeWebhookDelivered
	retried   []fakeWebhookRetried
	failed    []fakeWebhookFailed
	renewed   []fakeWebhookRenewal

	renewErr      *apperrors.AppError
	renewByIDPass map[int64]bool
}

type fakeWebhookDelivered struct {
	id int64
}

type fakeWebhookRetried struct {
	id       int64
	attempts int
}

type fakeWebhookFailed struct {
	id       int64
	attempts int
}

type fakeWebhookRenewal struct {
	id int64
}

func (f *fakeWebhookOutboxRepository) ClaimPendingForDispatch(
	_ context.Context,
	_ time.Time,
	_ int,
	_ string,
	_ time.Time,
) ([]dto.PendingWebhookOutboxEvent, *apperrors.AppError) {
	return f.claimed, nil
}

func (f *fakeWebhookOutboxRepository) MarkDelivered(
	_ context.Context,
	id int64,
	_ string,
	_ time.Time,
) (bool, *apperrors.AppError) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.delivered = append(f.delivered, fakeWebhookDelivered{id: id})
	return true, nil
}

func (f *fakeWebhookOutboxRepository) MarkRetry(
	_ context.Context,
	id int64,
	_ string,
	attempts int,
	_ time.Time,
	_ string,
	_ time.Time,
) (bool, *apperrors.AppError) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.retried = append(f.retried, fakeWebhookRetried{id: id, attempts: attempts})
	return true, nil
}

func (f *fakeWebhookOutboxRepository) MarkFailed(
	_ context.Context,
	id int64,
	_ string,
	attempts int,
	_ string,
	_ time.Time,
) (bool, *apperrors.AppError) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failed = append(f.failed, fakeWebhookFailed{id: id, attempts: attempts})
	return true, nil
}

func (f *fakeWebhookOutboxRepository) RenewLease(
	_ context.Context,
	id int64,
	_ string,
	_ time.Time,
	_ time.Time,
) (bool, *apperrors.AppError) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.renewed = append(f.renewed, fakeWebhookRenewal{id: id})
	if f.renewErr != nil {
		return false, f.renewErr
	}
	if f.renewByIDPass != nil {
		updated, exists := f.renewByIDPass[id]
		if exists {
			return updated, nil
		}
	}
	return true, nil
}

func (f *fakeWebhookOutboxRepository) RequeueFailedByEventID(
	_ context.Context,
	_ string,
	_ time.Time,
) (dto.WebhookOutboxMutationResult, *apperrors.AppError) {
	return dto.WebhookOutboxMutationResult{}, nil
}

func (f *fakeWebhookOutboxRepository) CancelByEventID(
	_ context.Context,
	_ string,
	_ string,
	_ time.Time,
) (dto.WebhookOutboxMutationResult, *apperrors.AppError) {
	return dto.WebhookOutboxMutationResult{}, nil
}

func (f *fakeWebhookOutboxRepository) renewCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.renewed)
}

type fakeWebhookEventGateway struct {
	mu        sync.Mutex
	results   map[string]dto.SendWebhookEventOutput
	errors    map[string]*apperrors.AppError
	sendDelay time.Duration
	inputs    []dto.SendWebhookEventInput
}

func (f *fakeWebhookEventGateway) SendWebhookEvent(
	_ context.Context,
	input dto.SendWebhookEventInput,
) (dto.SendWebhookEventOutput, *apperrors.AppError) {
	f.mu.Lock()
	f.inputs = append(f.inputs, input)
	f.mu.Unlock()

	if f.sendDelay > 0 {
		time.Sleep(f.sendDelay)
	}
	if f.errors != nil {
		if appErr, exists := f.errors[input.EventID]; exists {
			return dto.SendWebhookEventOutput{}, appErr
		}
	}
	if f.results != nil {
		if out, exists := f.results[input.EventID]; exists {
			return out, nil
		}
	}
	return dto.SendWebhookEventOutput{StatusCode: 200}, nil
}

func (f *fakeWebhookEventGateway) sentInputs() []dto.SendWebhookEventInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.SendWebhookEventInput, len(f.inputs))
	copy(out, f.inputs)
	return out
}
