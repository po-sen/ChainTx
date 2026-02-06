package use_cases

import (
	"context"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type getOpenAPISpecUseCase struct {
	readModel portsout.OpenAPISpecReadModel
}

func NewGetOpenAPISpecUseCase(readModel portsout.OpenAPISpecReadModel) portsin.GetOpenAPISpecUseCase {
	return &getOpenAPISpecUseCase{
		readModel: readModel,
	}
}

func (u *getOpenAPISpecUseCase) Execute(ctx context.Context, _ dto.GetOpenAPISpecQuery) (dto.OpenAPISpecOutput, *apperrors.AppError) {
	content, contentType, appErr := u.readModel.Read(ctx)
	if appErr != nil {
		return dto.OpenAPISpecOutput{}, appErr
	}

	return dto.OpenAPISpecOutput{
		Content:     content,
		ContentType: contentType,
	}, nil
}
