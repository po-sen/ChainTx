package out

import (
	"context"

	apperrors "chaintx/internal/shared_kernel/errors"
)

type PersistenceBootstrapGateway interface {
	CheckReadiness(ctx context.Context) *apperrors.AppError
	RunMigrations(ctx context.Context) *apperrors.AppError
	ValidateAssetCatalogIntegrity(ctx context.Context) *apperrors.AppError
}
