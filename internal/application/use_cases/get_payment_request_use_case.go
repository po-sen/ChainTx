package use_cases

import (
	"context"
	"strings"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type getPaymentRequestUseCase struct {
	readModel portsout.PaymentRequestReadModel
}

func NewGetPaymentRequestUseCase(readModel portsout.PaymentRequestReadModel) portsin.GetPaymentRequestUseCase {
	return &getPaymentRequestUseCase{readModel: readModel}
}

func (u *getPaymentRequestUseCase) Execute(ctx context.Context, query dto.GetPaymentRequestQuery) (dto.PaymentRequestResource, *apperrors.AppError) {
	if u.readModel == nil {
		return dto.PaymentRequestResource{}, apperrors.NewInternal(
			"payment_request_read_model_missing",
			"payment request read model is required",
			nil,
		)
	}

	id := strings.TrimSpace(query.ID)
	if id == "" {
		return dto.PaymentRequestResource{}, apperrors.NewValidation(
			"invalid_request",
			"payment request id is required",
			map[string]any{"field": "id"},
		)
	}

	resource, found, appErr := u.readModel.GetByID(ctx, id)
	if appErr != nil {
		return dto.PaymentRequestResource{}, appErr
	}
	if !found {
		return dto.PaymentRequestResource{}, apperrors.NewNotFound(
			"payment_request_not_found",
			"payment request was not found",
			map[string]any{"id": id},
		)
	}

	return resource, nil
}
