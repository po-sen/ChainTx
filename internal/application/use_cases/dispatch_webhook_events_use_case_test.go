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
	if len(repo.delivered) != 1 || repo.delivered[0].id != 1 {
		t.Fatalf("expected delivered id=1, got %+v", repo.delivered)
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
	if len(repo.retried) != 1 {
		t.Fatalf("expected one retry update, got %+v", repo.retried)
	}
	if repo.retried[0].attempts != 1 {
		t.Fatalf("expected attempts=1, got %+v", repo.retried[0])
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
	if len(repo.failed) != 1 || repo.failed[0].attempts != 3 {
		t.Fatalf("expected terminal failure attempts=3, got %+v", repo.failed)
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

func (f *fakeWebhookOutboxRepository) renewCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.renewed)
}

type fakeWebhookEventGateway struct {
	results   map[string]dto.SendWebhookEventOutput
	errors    map[string]*apperrors.AppError
	sendDelay time.Duration
}

func (f *fakeWebhookEventGateway) SendWebhookEvent(
	_ context.Context,
	input dto.SendWebhookEventInput,
) (dto.SendWebhookEventOutput, *apperrors.AppError) {
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
