package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type GetOpenAPISpecUseCase interface {
	Execute(ctx context.Context, query dto.GetOpenAPISpecQuery) (dto.OpenAPISpecOutput, *apperrors.AppError)
}
