package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type GetHealthUseCase interface {
	Execute(ctx context.Context, command dto.GetHealthCommand) (dto.HealthOutput, *apperrors.AppError)
}
