package out

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type PaymentRequestRepository interface {
	Create(
		ctx context.Context,
		command dto.CreatePaymentRequestPersistenceCommand,
		resolveAddress dto.ResolvePaymentAddressFunc,
	) (dto.CreatePaymentRequestPersistenceResult, *apperrors.AppError)
}
