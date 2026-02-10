package out

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type PaymentRequestReadModel interface {
	GetByID(ctx context.Context, id string) (dto.PaymentRequestResource, bool, *apperrors.AppError)
}
