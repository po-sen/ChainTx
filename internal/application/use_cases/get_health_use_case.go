package use_cases

import (
	"context"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	"chaintx/internal/domain/value_objects"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type getHealthUseCase struct{}

func NewGetHealthUseCase() portsin.GetHealthUseCase {
	return &getHealthUseCase{}
}

func (u *getHealthUseCase) Execute(_ context.Context, _ dto.GetHealthCommand) (dto.HealthOutput, *apperrors.AppError) {
	status := valueobjects.NewHealthyStatus()

	return dto.HealthOutput{
		Status: status.String(),
	}, nil
}
