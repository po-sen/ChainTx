//go:build !integration

package controllers

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

func TestWebhookOutboxControllerGetOverview(t *testing.T) {
	controller := NewWebhookOutboxController(
		stubOverviewUseCase{},
		stubListDLQUseCase{},
		stubRequeueDLQUseCase{},
		stubCancelEventUseCase{},
		log.New(io.Discard, "", 0),
	)

	req := httptest.NewRequest(http.MethodGet, "/v1/webhook-outbox/overview", nil)
	rec := httptest.NewRecorder()

	controller.GetOverview(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"pending_count":2`)) {
		t.Fatalf("expected pending_count in response, got %s", rec.Body.String())
	}
}

func TestWebhookOutboxControllerListDLQRejectsInvalidLimit(t *testing.T) {
	controller := NewWebhookOutboxController(
		stubOverviewUseCase{},
		stubListDLQUseCase{},
		stubRequeueDLQUseCase{},
		stubCancelEventUseCase{},
		log.New(io.Discard, "", 0),
	)

	req := httptest.NewRequest(http.MethodGet, "/v1/webhook-outbox/dlq?limit=abc", nil)
	rec := httptest.NewRecorder()

	controller.ListDLQ(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWebhookOutboxControllerRequeueDLQEvent(t *testing.T) {
	controller := NewWebhookOutboxController(
		stubOverviewUseCase{},
		stubListDLQUseCase{},
		stubRequeueDLQUseCase{},
		stubCancelEventUseCase{},
		log.New(io.Discard, "", 0),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhook-outbox/dlq/evt_1/requeue", nil)
	req.SetPathValue("event_id", "evt_1")
	rec := httptest.NewRecorder()

	controller.RequeueDLQEvent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"delivery_status":"pending"`)) {
		t.Fatalf("expected pending status, got %s", rec.Body.String())
	}
}

func TestWebhookOutboxControllerCancelEventInvalidJSON(t *testing.T) {
	controller := NewWebhookOutboxController(
		stubOverviewUseCase{},
		stubListDLQUseCase{},
		stubRequeueDLQUseCase{},
		stubCancelEventUseCase{},
		log.New(io.Discard, "", 0),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhook-outbox/events/evt_1/cancel", bytes.NewBufferString("{"))
	req.SetPathValue("event_id", "evt_1")
	rec := httptest.NewRecorder()

	controller.CancelEvent(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWebhookOutboxControllerCancelEventSuccess(t *testing.T) {
	cancelUseCase := &stubCancelEventCaptureUseCase{}
	controller := NewWebhookOutboxController(
		stubOverviewUseCase{},
		stubListDLQUseCase{},
		stubRequeueDLQUseCase{},
		cancelUseCase,
		log.New(io.Discard, "", 0),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/webhook-outbox/events/evt_1/cancel",
		bytes.NewBufferString(`{"reason":"operator action"}`),
	)
	req.SetPathValue("event_id", "evt_1")
	rec := httptest.NewRecorder()

	controller.CancelEvent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if cancelUseCase.lastCommand.EventID != "evt_1" {
		t.Fatalf("expected event id evt_1, got %+v", cancelUseCase.lastCommand)
	}
	if cancelUseCase.lastCommand.Reason != "operator action" {
		t.Fatalf("expected reason operator action, got %+v", cancelUseCase.lastCommand)
	}
}

type stubOverviewUseCase struct{}

func (stubOverviewUseCase) Execute(_ context.Context, _ dto.GetWebhookOutboxOverviewQuery) (dto.WebhookOutboxOverview, *apperrors.AppError) {
	return dto.WebhookOutboxOverview{
		PendingCount:      2,
		PendingReadyCount: 1,
		RetryingCount:     1,
		FailedCount:       3,
		DeliveredCount:    5,
	}, nil
}

type stubListDLQUseCase struct{}

func (stubListDLQUseCase) Execute(_ context.Context, query dto.ListWebhookDLQEventsQuery) (dto.ListWebhookDLQEventsOutput, *apperrors.AppError) {
	if query.Limit > 200 {
		return dto.ListWebhookDLQEventsOutput{}, apperrors.NewValidation("invalid_request", "limit must be between 1 and 200", nil)
	}
	return dto.ListWebhookDLQEventsOutput{
		Events: []dto.WebhookDLQEvent{{EventID: "evt_1"}},
	}, nil
}

type stubRequeueDLQUseCase struct{}

func (stubRequeueDLQUseCase) Execute(_ context.Context, command dto.RequeueWebhookDLQEventCommand) (dto.RequeueWebhookDLQEventOutput, *apperrors.AppError) {
	return dto.RequeueWebhookDLQEventOutput{
		EventID:        command.EventID,
		DeliveryStatus: "pending",
		UpdatedAt:      time.Now().UTC(),
	}, nil
}

type stubCancelEventUseCase struct{}

func (stubCancelEventUseCase) Execute(_ context.Context, command dto.CancelWebhookOutboxEventCommand) (dto.CancelWebhookOutboxEventOutput, *apperrors.AppError) {
	return dto.CancelWebhookOutboxEventOutput{
		EventID:        command.EventID,
		DeliveryStatus: "failed",
		LastError:      "manual_cancelled",
		UpdatedAt:      time.Now().UTC(),
	}, nil
}

type stubCancelEventCaptureUseCase struct {
	lastCommand dto.CancelWebhookOutboxEventCommand
}

func (s *stubCancelEventCaptureUseCase) Execute(_ context.Context, command dto.CancelWebhookOutboxEventCommand) (dto.CancelWebhookOutboxEventOutput, *apperrors.AppError) {
	s.lastCommand = command
	return dto.CancelWebhookOutboxEventOutput{
		EventID:        command.EventID,
		DeliveryStatus: "failed",
		LastError:      "manual_cancelled",
		UpdatedAt:      time.Now().UTC(),
	}, nil
}
