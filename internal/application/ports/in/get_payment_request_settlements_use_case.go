package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type GetPaymentRequestSettlementsUseCase interface {
	Execute(ctx context.Context, query dto.GetPaymentRequestSettlementsQuery) (dto.PaymentRequestSettlementsResource, *apperrors.AppError)
}
