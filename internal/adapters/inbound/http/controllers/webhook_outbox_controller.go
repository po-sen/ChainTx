package controllers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type WebhookOutboxController struct {
	overviewUseCase portsin.GetWebhookOutboxOverviewUseCase
	listDLQUseCase  portsin.ListWebhookDLQEventsUseCase
	requeueUseCase  portsin.RequeueWebhookDLQEventUseCase
	cancelUseCase   portsin.CancelWebhookOutboxEventUseCase
	logger          *log.Logger
}

type webhookCancelPayload struct {
	Reason string `json:"reason,omitempty"`
}

func NewWebhookOutboxController(
	overviewUseCase portsin.GetWebhookOutboxOverviewUseCase,
	listDLQUseCase portsin.ListWebhookDLQEventsUseCase,
	requeueUseCase portsin.RequeueWebhookDLQEventUseCase,
	cancelUseCase portsin.CancelWebhookOutboxEventUseCase,
	logger *log.Logger,
) *WebhookOutboxController {
	return &WebhookOutboxController{
		overviewUseCase: overviewUseCase,
		listDLQUseCase:  listDLQUseCase,
		requeueUseCase:  requeueUseCase,
		cancelUseCase:   cancelUseCase,
		logger:          logger,
	}
}

func (c *WebhookOutboxController) GetOverview(w http.ResponseWriter, r *http.Request) {
	if c.overviewUseCase == nil {
		writeAppError(w, apperrors.NewInternal(
			"webhook_outbox_overview_use_case_missing",
			"webhook outbox overview use case is required",
			nil,
		))
		return
	}

	output, appErr := c.overviewUseCase.Execute(r.Context(), dto.GetWebhookOutboxOverviewQuery{Now: time.Now().UTC()})
	if appErr != nil {
		c.logRequestError(r.Method, "/v1/webhook-outbox/overview", appErr)
		writeAppError(w, appErr)
		return
	}

	writeJSON(w, http.StatusOK, output)
}

func (c *WebhookOutboxController) ListDLQ(w http.ResponseWriter, r *http.Request) {
	if c.listDLQUseCase == nil {
		writeAppError(w, apperrors.NewInternal(
			"webhook_outbox_dlq_use_case_missing",
			"webhook outbox dlq use case is required",
			nil,
		))
		return
	}

	limit := 0
	rawLimit := strings.TrimSpace(r.URL.Query().Get("limit"))
	if rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeAppError(w, apperrors.NewValidation(
				"invalid_request",
				"limit must be an integer",
				map[string]any{"field": "limit"},
			))
			return
		}
		limit = parsed
	}

	output, appErr := c.listDLQUseCase.Execute(r.Context(), dto.ListWebhookDLQEventsQuery{Limit: limit})
	if appErr != nil {
		c.logRequestError(r.Method, "/v1/webhook-outbox/dlq", appErr)
		writeAppError(w, appErr)
		return
	}

	writeJSON(w, http.StatusOK, output)
}

func (c *WebhookOutboxController) RequeueDLQEvent(w http.ResponseWriter, r *http.Request) {
	if c.requeueUseCase == nil {
		writeAppError(w, apperrors.NewInternal(
			"webhook_outbox_requeue_use_case_missing",
			"webhook outbox requeue use case is required",
			nil,
		))
		return
	}

	eventID := strings.TrimSpace(r.PathValue("event_id"))
	output, appErr := c.requeueUseCase.Execute(r.Context(), dto.RequeueWebhookDLQEventCommand{
		EventID: eventID,
		Now:     time.Now().UTC(),
	})
	if appErr != nil {
		c.logRequestError(r.Method, "/v1/webhook-outbox/dlq/{event_id}/requeue", appErr)
		writeAppError(w, appErr)
		return
	}

	writeJSON(w, http.StatusOK, output)
}

func (c *WebhookOutboxController) CancelEvent(w http.ResponseWriter, r *http.Request) {
	if c.cancelUseCase == nil {
		writeAppError(w, apperrors.NewInternal(
			"webhook_outbox_cancel_use_case_missing",
			"webhook outbox cancel use case is required",
			nil,
		))
		return
	}

	payload, appErr := parseWebhookCancelPayload(r.Body)
	if appErr != nil {
		writeAppError(w, appErr)
		return
	}

	eventID := strings.TrimSpace(r.PathValue("event_id"))
	output, appErr := c.cancelUseCase.Execute(r.Context(), dto.CancelWebhookOutboxEventCommand{
		EventID: eventID,
		Reason:  payload.Reason,
		Now:     time.Now().UTC(),
	})
	if appErr != nil {
		c.logRequestError(r.Method, "/v1/webhook-outbox/events/{event_id}/cancel", appErr)
		writeAppError(w, appErr)
		return
	}

	writeJSON(w, http.StatusOK, output)
}

func (c *WebhookOutboxController) logRequestError(method string, path string, appErr *apperrors.AppError) {
	if c == nil || c.logger == nil || appErr == nil {
		return
	}
	c.logger.Printf("request error path=%s method=%s code=%s message=%s", path, method, appErr.Code, appErr.Message)
}

func parseWebhookCancelPayload(body io.Reader) (webhookCancelPayload, *apperrors.AppError) {
	if body == nil {
		return webhookCancelPayload{}, nil
	}

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	decoder.UseNumber()

	payload := webhookCancelPayload{}
	if err := decoder.Decode(&payload); err != nil {
		if err == io.EOF {
			return webhookCancelPayload{}, nil
		}
		return webhookCancelPayload{}, apperrors.NewValidation(
			"invalid_request",
			"request body must be valid JSON",
			map[string]any{"error": err.Error()},
		)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return webhookCancelPayload{}, apperrors.NewValidation(
			"invalid_request",
			"request body must contain a single JSON object",
			nil,
		)
	}

	payload.Reason = strings.TrimSpace(payload.Reason)
	return payload, nil
}
