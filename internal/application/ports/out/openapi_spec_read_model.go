package out

import (
	"context"

	apperrors "chaintx/internal/shared_kernel/errors"
)

type OpenAPISpecReadModel interface {
	Read(ctx context.Context) ([]byte, string, *apperrors.AppError)
}
