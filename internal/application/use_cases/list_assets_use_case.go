package use_cases

import (
	"context"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type listAssetsUseCase struct {
	readModel portsout.AssetCatalogReadModel
}

func NewListAssetsUseCase(readModel portsout.AssetCatalogReadModel) portsin.ListAssetsUseCase {
	return &listAssetsUseCase{readModel: readModel}
}

func (u *listAssetsUseCase) Execute(ctx context.Context, _ dto.ListAssetsQuery) (dto.ListAssetsOutput, *apperrors.AppError) {
	if u.readModel == nil {
		return dto.ListAssetsOutput{}, apperrors.NewInternal(
			"asset_catalog_read_model_missing",
			"asset catalog read model is required",
			nil,
		)
	}

	assets, appErr := u.readModel.ListEnabled(ctx)
	if appErr != nil {
		return dto.ListAssetsOutput{}, appErr
	}

	return dto.ListAssetsOutput{Assets: assets}, nil
}
