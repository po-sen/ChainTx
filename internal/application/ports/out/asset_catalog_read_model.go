package out

import (
	"context"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type AssetCatalogReadModel interface {
	ListEnabled(ctx context.Context) ([]dto.AssetCatalogEntry, *apperrors.AppError)
}
