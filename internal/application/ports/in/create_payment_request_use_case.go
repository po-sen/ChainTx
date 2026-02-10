package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type CreatePaymentRequestUseCase interface {
	Execute(ctx context.Context, command dto.CreatePaymentRequestCommand) (dto.CreatePaymentRequestOutput, *apperrors.AppError)
}
