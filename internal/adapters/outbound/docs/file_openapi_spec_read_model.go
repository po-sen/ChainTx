package docs

import (
	"context"
	"os"

	apperrors "chaintx/internal/shared_kernel/errors"
)

type FileOpenAPISpecReadModel struct {
	path string
}

func NewFileOpenAPISpecReadModel(path string) *FileOpenAPISpecReadModel {
	return &FileOpenAPISpecReadModel{
		path: path,
	}
}

func (r *FileOpenAPISpecReadModel) Read(_ context.Context) ([]byte, string, *apperrors.AppError) {
	content, err := os.ReadFile(r.path)
	if err != nil {
		return nil, "", apperrors.NewInternal(
			"OPENAPI_FILE_READ_FAILED",
			"failed to read OpenAPI spec file",
			map[string]any{"path": r.path},
		)
	}

	return content, "application/yaml; charset=utf-8", nil
}
