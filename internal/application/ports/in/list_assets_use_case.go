package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type ListAssetsUseCase interface {
	Execute(ctx context.Context, query dto.ListAssetsQuery) (dto.ListAssetsOutput, *apperrors.AppError)
}
