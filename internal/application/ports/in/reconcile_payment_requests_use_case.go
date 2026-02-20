package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type ReconcilePaymentRequestsUseCase interface {
	Execute(
		ctx context.Context,
		command dto.ReconcilePaymentRequestsCommand,
	) (dto.ReconcilePaymentRequestsOutput, *apperrors.AppError)
}
