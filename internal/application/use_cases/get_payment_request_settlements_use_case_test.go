package use_cases

import (
	"context"
	"testing"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

func TestGetPaymentRequestSettlementsUseCaseExecuteValidation(t *testing.T) {
	useCase := NewGetPaymentRequestSettlementsUseCase(stubPaymentRequestReadModelForSettlements{})

	_, appErr := useCase.Execute(context.Background(), dto.GetPaymentRequestSettlementsQuery{ID: "   "})
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
	if appErr.Code != "invalid_request" {
		t.Fatalf("expected invalid_request, got %s", appErr.Code)
	}
}

func TestGetPaymentRequestSettlementsUseCaseExecuteNotFound(t *testing.T) {
	useCase := NewGetPaymentRequestSettlementsUseCase(stubPaymentRequestReadModelForSettlements{})

	_, appErr := useCase.Execute(context.Background(), dto.GetPaymentRequestSettlementsQuery{ID: "pr_missing"})
	if appErr == nil {
		t.Fatalf("expected not found error")
	}
	if appErr.Code != "payment_request_not_found" {
		t.Fatalf("expected payment_request_not_found, got %s", appErr.Code)
	}
}

func TestGetPaymentRequestSettlementsUseCaseExecuteEmptySettlements(t *testing.T) {
	useCase := NewGetPaymentRequestSettlementsUseCase(stubPaymentRequestReadModelForSettlements{
		found:       true,
		settlements: nil,
	})

	output, appErr := useCase.Execute(context.Background(), dto.GetPaymentRequestSettlementsQuery{ID: "pr_test"})
	if appErr != nil {
		t.Fatalf("expected success, got %+v", appErr)
	}
	if output.PaymentRequestID != "pr_test" {
		t.Fatalf("expected payment_request_id=pr_test, got %s", output.PaymentRequestID)
	}
	if output.Settlements == nil {
		t.Fatalf("expected non-nil settlements slice")
	}
	if len(output.Settlements) != 0 {
		t.Fatalf("expected empty settlements, got %d", len(output.Settlements))
	}
}

func TestGetPaymentRequestSettlementsUseCaseExecuteSuccess(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	useCase := NewGetPaymentRequestSettlementsUseCase(stubPaymentRequestReadModelForSettlements{
		found: true,
		settlements: []dto.PaymentRequestSettlementResource{
			{
				EvidenceRef:   "btc:tx:1",
				AmountMinor:   "1000",
				Confirmations: 3,
				IsCanonical:   true,
				Metadata:      map[string]any{"source": "chain_stats"},
				FirstSeenAt:   now,
				LastSeenAt:    now,
				UpdatedAt:     now,
			},
		},
	})

	output, appErr := useCase.Execute(context.Background(), dto.GetPaymentRequestSettlementsQuery{ID: "pr_test"})
	if appErr != nil {
		t.Fatalf("expected success, got %+v", appErr)
	}
	if len(output.Settlements) != 1 {
		t.Fatalf("expected one settlement, got %d", len(output.Settlements))
	}
	if output.Settlements[0].EvidenceRef != "btc:tx:1" {
		t.Fatalf("expected evidence_ref btc:tx:1, got %s", output.Settlements[0].EvidenceRef)
	}
}

func TestGetPaymentRequestSettlementsUseCaseExecuteReadModelError(t *testing.T) {
	useCase := NewGetPaymentRequestSettlementsUseCase(stubPaymentRequestReadModelForSettlements{
		listErr: apperrors.NewInternal(
			"payment_request_query_failed",
			"failed",
			nil,
		),
	})

	_, appErr := useCase.Execute(context.Background(), dto.GetPaymentRequestSettlementsQuery{ID: "pr_test"})
	if appErr == nil {
		t.Fatalf("expected error")
	}
	if appErr.Code != "payment_request_query_failed" {
		t.Fatalf("expected payment_request_query_failed, got %s", appErr.Code)
	}
}

type stubPaymentRequestReadModelForSettlements struct {
	found       bool
	settlements []dto.PaymentRequestSettlementResource
	listErr     *apperrors.AppError
}

func (s stubPaymentRequestReadModelForSettlements) GetByID(
	_ context.Context,
	_ string,
) (dto.PaymentRequestResource, bool, *apperrors.AppError) {
	return dto.PaymentRequestResource{}, false, nil
}

func (s stubPaymentRequestReadModelForSettlements) ListSettlementsByPaymentRequestID(
	_ context.Context,
	_ string,
) ([]dto.PaymentRequestSettlementResource, bool, *apperrors.AppError) {
	if s.listErr != nil {
		return nil, false, s.listErr
	}
	return s.settlements, s.found, nil
}
