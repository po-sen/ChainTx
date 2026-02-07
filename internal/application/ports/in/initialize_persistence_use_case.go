package in

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type InitializePersistenceUseCase interface {
	Execute(ctx context.Context, command dto.InitializePersistenceCommand) *apperrors.AppError
}
