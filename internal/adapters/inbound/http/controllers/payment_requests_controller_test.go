package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

func TestPaymentRequestsControllerCreatePaymentRequestCreated(t *testing.T) {
	controller := NewPaymentRequestsController(
		stubCreateUseCase{replayed: false},
		stubGetUseCase{},
		log.New(io.Discard, "", 0),
	)

	body := bytes.NewBufferString(`{"chain":"bitcoin","network":"mainnet","asset":"BTC"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/payment-requests", body)
	rec := httptest.NewRecorder()

	controller.CreatePaymentRequest(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Location") == "" {
		t.Fatalf("expected Location header")
	}
}

func TestPaymentRequestsControllerCreatePaymentRequestReplayed(t *testing.T) {
	controller := NewPaymentRequestsController(
		stubCreateUseCase{replayed: true},
		stubGetUseCase{},
		log.New(io.Discard, "", 0),
	)

	body := bytes.NewBufferString(`{"chain":"bitcoin","network":"mainnet","asset":"BTC"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/payment-requests", body)
	rec := httptest.NewRecorder()

	controller.CreatePaymentRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Idempotency-Replayed") != "true" {
		t.Fatalf("expected X-Idempotency-Replayed=true")
	}
}

func TestPaymentRequestsControllerCreatePaymentRequestInvalidJSON(t *testing.T) {
	controller := NewPaymentRequestsController(
		stubCreateUseCase{replayed: false},
		stubGetUseCase{},
		log.New(io.Discard, "", 0),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/payment-requests", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()

	controller.CreatePaymentRequest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json: %v", err)
	}
	if _, ok := payload["error"]; !ok {
		t.Fatalf("expected error envelope in response: %v", payload)
	}
}

func TestPaymentRequestsControllerGetPaymentRequest(t *testing.T) {
	controller := NewPaymentRequestsController(
		stubCreateUseCase{replayed: false},
		stubGetUseCase{},
		log.New(io.Discard, "", 0),
	)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment-requests/pr_test", nil)
	req.SetPathValue("id", "pr_test")
	rec := httptest.NewRecorder()

	controller.GetPaymentRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"id":"pr_test"`)) {
		t.Fatalf("expected id in payload, got %s", rec.Body.String())
	}
}

type stubCreateUseCase struct {
	replayed bool
}

func (s stubCreateUseCase) Execute(_ context.Context, _ dto.CreatePaymentRequestCommand) (dto.CreatePaymentRequestOutput, *apperrors.AppError) {
	createdAt := time.Unix(0, 0).UTC()
	expiresAt := createdAt.Add(time.Hour)

	return dto.CreatePaymentRequestOutput{
		Resource: dto.PaymentRequestResource{
			ID:        "pr_test",
			Status:    "pending",
			Chain:     "bitcoin",
			Network:   "mainnet",
			Asset:     "BTC",
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
			PaymentInstructions: dto.PaymentInstructions{
				Address:         "bc1qexample",
				AddressScheme:   "bip84_p2wpkh",
				DerivationIndex: 1,
			},
		},
		Replayed: s.replayed,
	}, nil
}

type stubGetUseCase struct{}

func (stubGetUseCase) Execute(_ context.Context, query dto.GetPaymentRequestQuery) (dto.PaymentRequestResource, *apperrors.AppError) {
	createdAt := time.Unix(0, 0).UTC()
	expiresAt := createdAt.Add(time.Hour)

	return dto.PaymentRequestResource{
		ID:        query.ID,
		Status:    "pending",
		Chain:     "bitcoin",
		Network:   "mainnet",
		Asset:     "BTC",
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
		PaymentInstructions: dto.PaymentInstructions{
			Address:         "bc1qexample",
			AddressScheme:   "bip84_p2wpkh",
			DerivationIndex: 1,
		},
	}, nil
}
