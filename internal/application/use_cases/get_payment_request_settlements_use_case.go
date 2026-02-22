package use_cases

import (
	"context"
	"strings"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type getPaymentRequestSettlementsUseCase struct {
	readModel portsout.PaymentRequestReadModel
}

func NewGetPaymentRequestSettlementsUseCase(
	readModel portsout.PaymentRequestReadModel,
) portsin.GetPaymentRequestSettlementsUseCase {
	return &getPaymentRequestSettlementsUseCase{readModel: readModel}
}

func (u *getPaymentRequestSettlementsUseCase) Execute(
	ctx context.Context,
	query dto.GetPaymentRequestSettlementsQuery,
) (dto.PaymentRequestSettlementsResource, *apperrors.AppError) {
	if u.readModel == nil {
		return dto.PaymentRequestSettlementsResource{}, apperrors.NewInternal(
			"payment_request_read_model_missing",
			"payment request read model is required",
			nil,
		)
	}

	id := strings.TrimSpace(query.ID)
	if id == "" {
		return dto.PaymentRequestSettlementsResource{}, apperrors.NewValidation(
			"invalid_request",
			"payment request id is required",
			map[string]any{"field": "id"},
		)
	}

	settlements, found, appErr := u.readModel.ListSettlementsByPaymentRequestID(ctx, id)
	if appErr != nil {
		return dto.PaymentRequestSettlementsResource{}, appErr
	}
	if !found {
		return dto.PaymentRequestSettlementsResource{}, apperrors.NewNotFound(
			"payment_request_not_found",
			"payment request was not found",
			map[string]any{"id": id},
		)
	}
	if settlements == nil {
		settlements = []dto.PaymentRequestSettlementResource{}
	}

	return dto.PaymentRequestSettlementsResource{
		PaymentRequestID: id,
		Settlements:      settlements,
	}, nil
}
