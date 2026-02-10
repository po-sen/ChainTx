package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type GetPaymentRequestUseCase interface {
	Execute(ctx context.Context, query dto.GetPaymentRequestQuery) (dto.PaymentRequestResource, *apperrors.AppError)
}
