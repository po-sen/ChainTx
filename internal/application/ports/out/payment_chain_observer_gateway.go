package out

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type PaymentChainObserverGateway interface {
	ObservePaymentRequest(
		ctx context.Context,
		input dto.ObservePaymentRequestInput,
	) (dto.ObservePaymentRequestOutput, *apperrors.AppError)
}
