//go:build !integration

package use_cases

import (
	"context"
	"testing"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

func TestGetWebhookOutboxOverviewUseCaseReturnsOverview(t *testing.T) {
	now := time.Date(2026, 2, 21, 13, 0, 0, 0, time.UTC)
	readModel := &fakeWebhookOutboxReadModel{
		overview: dto.WebhookOutboxOverview{PendingCount: 2},
	}
	useCase := NewGetWebhookOutboxOverviewUseCase(readModel)

	output, appErr := useCase.Execute(context.Background(), dto.GetWebhookOutboxOverviewQuery{Now: now})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if output.PendingCount != 2 {
		t.Fatalf("expected pending_count=2, got %+v", output)
	}
	if readModel.lastOverviewNow.IsZero() || !readModel.lastOverviewNow.Equal(now) {
		t.Fatalf("expected overview query now=%s, got %s", now, readModel.lastOverviewNow)
	}
}

func TestListWebhookDLQEventsUseCaseDefaultLimit(t *testing.T) {
	readModel := &fakeWebhookOutboxReadModel{}
	useCase := NewListWebhookDLQEventsUseCase(readModel)

	_, appErr := useCase.Execute(context.Background(), dto.ListWebhookDLQEventsQuery{})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if readModel.lastDLQLimit != defaultWebhookDLQLimit {
		t.Fatalf("expected default limit %d, got %d", defaultWebhookDLQLimit, readModel.lastDLQLimit)
	}
}

func TestListWebhookDLQEventsUseCaseRejectsInvalidLimit(t *testing.T) {
	useCase := NewListWebhookDLQEventsUseCase(&fakeWebhookOutboxReadModel{})

	_, appErr := useCase.Execute(context.Background(), dto.ListWebhookDLQEventsQuery{Limit: 201})
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
	if appErr.Code != "invalid_request" {
		t.Fatalf("expected invalid_request, got %s", appErr.Code)
	}
}

func TestRequeueWebhookDLQEventUseCaseReturnsConflictWhenStatusNotFailed(t *testing.T) {
	now := time.Date(2026, 2, 21, 13, 5, 0, 0, time.UTC)
	repo := &fakeWebhookOutboxOpsRepository{
		requeueResult: dto.WebhookOutboxMutationResult{Found: true, Updated: false, CurrentStatus: "delivered"},
	}
	useCase := NewRequeueWebhookDLQEventUseCase(repo)

	_, appErr := useCase.Execute(context.Background(), dto.RequeueWebhookDLQEventCommand{EventID: "evt_x", Now: now})
	if appErr == nil {
		t.Fatalf("expected conflict error")
	}
	if appErr.Type != apperrors.TypeConflict {
		t.Fatalf("expected conflict type, got %+v", appErr)
	}
}

func TestRequeueWebhookDLQEventUseCaseSuccess(t *testing.T) {
	now := time.Date(2026, 2, 21, 13, 10, 0, 0, time.UTC)
	repo := &fakeWebhookOutboxOpsRepository{
		requeueResult: dto.WebhookOutboxMutationResult{Found: true, Updated: true, CurrentStatus: "failed"},
	}
	useCase := NewRequeueWebhookDLQEventUseCase(repo)

	output, appErr := useCase.Execute(context.Background(), dto.RequeueWebhookDLQEventCommand{EventID: "evt_x", Now: now})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if output.DeliveryStatus != "pending" {
		t.Fatalf("expected pending status, got %+v", output)
	}
}

func TestCancelWebhookOutboxEventUseCaseUsesDefaultReason(t *testing.T) {
	now := time.Date(2026, 2, 21, 13, 15, 0, 0, time.UTC)
	repo := &fakeWebhookOutboxOpsRepository{
		cancelResult: dto.WebhookOutboxMutationResult{Found: true, Updated: true, CurrentStatus: "pending"},
	}
	useCase := NewCancelWebhookOutboxEventUseCase(repo)

	output, appErr := useCase.Execute(context.Background(), dto.CancelWebhookOutboxEventCommand{
		EventID: "evt_x",
		Now:     now,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if output.LastError != "manual_cancelled" {
		t.Fatalf("expected default reason manual_cancelled, got %s", output.LastError)
	}
	if repo.lastCancelError != "manual_cancelled" {
		t.Fatalf("expected persisted cancel reason manual_cancelled, got %s", repo.lastCancelError)
	}
}

func TestCancelWebhookOutboxEventUseCaseReturnsNotFound(t *testing.T) {
	repo := &fakeWebhookOutboxOpsRepository{
		cancelResult: dto.WebhookOutboxMutationResult{Found: false, Updated: false},
	}
	useCase := NewCancelWebhookOutboxEventUseCase(repo)

	_, appErr := useCase.Execute(context.Background(), dto.CancelWebhookOutboxEventCommand{EventID: "evt_missing"})
	if appErr == nil {
		t.Fatalf("expected not found error")
	}
	if appErr.Type != apperrors.TypeNotFound {
		t.Fatalf("expected not_found type, got %+v", appErr)
	}
}

type fakeWebhookOutboxReadModel struct {
	overview        dto.WebhookOutboxOverview
	overviewErr     *apperrors.AppError
	dlqEvents       []dto.WebhookDLQEvent
	dlqErr          *apperrors.AppError
	lastOverviewNow time.Time
	lastDLQLimit    int
}

func (f *fakeWebhookOutboxReadModel) GetOverview(_ context.Context, now time.Time) (dto.WebhookOutboxOverview, *apperrors.AppError) {
	f.lastOverviewNow = now
	if f.overviewErr != nil {
		return dto.WebhookOutboxOverview{}, f.overviewErr
	}
	return f.overview, nil
}

func (f *fakeWebhookOutboxReadModel) ListDLQ(_ context.Context, limit int) ([]dto.WebhookDLQEvent, *apperrors.AppError) {
	f.lastDLQLimit = limit
	if f.dlqErr != nil {
		return nil, f.dlqErr
	}
	return f.dlqEvents, nil
}

type fakeWebhookOutboxOpsRepository struct {
	requeueResult   dto.WebhookOutboxMutationResult
	requeueErr      *apperrors.AppError
	cancelResult    dto.WebhookOutboxMutationResult
	cancelErr       *apperrors.AppError
	lastCancelError string
}

func (f *fakeWebhookOutboxOpsRepository) ClaimPendingForDispatch(
	_ context.Context,
	_ time.Time,
	_ int,
	_ string,
	_ time.Time,
) ([]dto.PendingWebhookOutboxEvent, *apperrors.AppError) {
	return nil, nil
}

func (f *fakeWebhookOutboxOpsRepository) MarkDelivered(
	_ context.Context,
	_ int64,
	_ string,
	_ time.Time,
) (bool, *apperrors.AppError) {
	return false, nil
}

func (f *fakeWebhookOutboxOpsRepository) MarkRetry(
	_ context.Context,
	_ int64,
	_ string,
	_ int,
	_ time.Time,
	_ string,
	_ time.Time,
) (bool, *apperrors.AppError) {
	return false, nil
}

func (f *fakeWebhookOutboxOpsRepository) MarkFailed(
	_ context.Context,
	_ int64,
	_ string,
	_ int,
	_ string,
	_ time.Time,
) (bool, *apperrors.AppError) {
	return false, nil
}

func (f *fakeWebhookOutboxOpsRepository) RenewLease(
	_ context.Context,
	_ int64,
	_ string,
	_ time.Time,
	_ time.Time,
) (bool, *apperrors.AppError) {
	return false, nil
}

func (f *fakeWebhookOutboxOpsRepository) RequeueFailedByEventID(
	_ context.Context,
	_ string,
	_ time.Time,
) (dto.WebhookOutboxMutationResult, *apperrors.AppError) {
	if f.requeueErr != nil {
		return dto.WebhookOutboxMutationResult{}, f.requeueErr
	}
	return f.requeueResult, nil
}

func (f *fakeWebhookOutboxOpsRepository) CancelByEventID(
	_ context.Context,
	_ string,
	lastError string,
	_ time.Time,
) (dto.WebhookOutboxMutationResult, *apperrors.AppError) {
	f.lastCancelError = lastError
	if f.cancelErr != nil {
		return dto.WebhookOutboxMutationResult{}, f.cancelErr
	}
	return f.cancelResult, nil
}
