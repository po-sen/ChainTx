package controllers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	apperrors "chaintx/internal/shared_kernel/errors"
)

const (
	headerIdempotencyKey      = "Idempotency-Key"
	headerIdempotencyReplayed = "X-Idempotency-Replayed"
	headerPrincipalID         = "X-Principal-ID"
)

type PaymentRequestsController struct {
	createUseCase portsin.CreatePaymentRequestUseCase
	getUseCase    portsin.GetPaymentRequestUseCase
	logger        *log.Logger
}

type createPaymentRequestPayload struct {
	Chain               string         `json:"chain"`
	Network             string         `json:"network"`
	Asset               string         `json:"asset"`
	WebhookURL          string         `json:"webhook_url"`
	ExpectedAmountMinor *string        `json:"expected_amount_minor,omitempty"`
	ExpiresInSeconds    *int64         `json:"expires_in_seconds,omitempty"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

func NewPaymentRequestsController(
	createUseCase portsin.CreatePaymentRequestUseCase,
	getUseCase portsin.GetPaymentRequestUseCase,
	logger *log.Logger,
) *PaymentRequestsController {
	return &PaymentRequestsController{
		createUseCase: createUseCase,
		getUseCase:    getUseCase,
		logger:        logger,
	}
}

func (c *PaymentRequestsController) CreatePaymentRequest(w http.ResponseWriter, r *http.Request) {
	payload, appErr := parseCreatePaymentRequestPayload(r.Body)
	if appErr != nil {
		writeAppError(w, appErr)
		return
	}

	output, appErr := c.createUseCase.Execute(r.Context(), dto.CreatePaymentRequestCommand{
		IdempotencyScope: dto.IdempotencyScope{
			PrincipalID: strings.TrimSpace(r.Header.Get(headerPrincipalID)),
			HTTPMethod:  r.Method,
			HTTPPath:    "/v1/payment-requests",
		},
		IdempotencyKey:      strings.TrimSpace(r.Header.Get(headerIdempotencyKey)),
		Chain:               payload.Chain,
		Network:             payload.Network,
		Asset:               payload.Asset,
		WebhookURL:          payload.WebhookURL,
		ExpectedAmountMinor: payload.ExpectedAmountMinor,
		ExpiresInSeconds:    payload.ExpiresInSeconds,
		Metadata:            payload.Metadata,
	})
	if appErr != nil {
		c.logger.Printf("request error path=/v1/payment-requests method=%s code=%s message=%s", r.Method, appErr.Code, appErr.Message)
		writeAppError(w, appErr)
		return
	}

	location := "/v1/payment-requests/" + output.Resource.ID
	w.Header().Set("Location", location)
	if output.Replayed {
		w.Header().Set(headerIdempotencyReplayed, "true")
		writeJSON(w, http.StatusOK, output.Resource)
		return
	}

	writeJSON(w, http.StatusCreated, output.Resource)
}

func (c *PaymentRequestsController) GetPaymentRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	resource, appErr := c.getUseCase.Execute(r.Context(), dto.GetPaymentRequestQuery{ID: id})
	if appErr != nil {
		c.logger.Printf("request error path=/v1/payment-requests/{id} method=%s code=%s message=%s", r.Method, appErr.Code, appErr.Message)
		writeAppError(w, appErr)
		return
	}

	writeJSON(w, http.StatusOK, resource)
}

func parseCreatePaymentRequestPayload(body io.Reader) (createPaymentRequestPayload, *apperrors.AppError) {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	decoder.UseNumber()

	payload := createPaymentRequestPayload{}
	if err := decoder.Decode(&payload); err != nil {
		return createPaymentRequestPayload{}, apperrors.NewValidation(
			"invalid_request",
			"request body must be valid JSON",
			map[string]any{"error": err.Error()},
		)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return createPaymentRequestPayload{}, apperrors.NewValidation(
			"invalid_request",
			"request body must contain a single JSON object",
			nil,
		)
	}

	if payload.Metadata == nil {
		payload.Metadata = map[string]any{}
	}
	payload.WebhookURL = strings.TrimSpace(payload.WebhookURL)
	if payload.WebhookURL == "" {
		return createPaymentRequestPayload{}, apperrors.NewValidation(
			"invalid_request",
			"webhook_url is required",
			map[string]any{"field": "webhook_url"},
		)
	}

	return payload, nil
}
