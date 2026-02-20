package use_cases

import (
	"context"
	"strconv"
	"time"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type initializePersistenceUseCase struct {
	gateway portsout.PersistenceBootstrapGateway
}

func NewInitializePersistenceUseCase(gateway portsout.PersistenceBootstrapGateway) portsin.InitializePersistenceUseCase {
	return &initializePersistenceUseCase{
		gateway: gateway,
	}
}

func (u *initializePersistenceUseCase) Execute(ctx context.Context, command dto.InitializePersistenceCommand) *apperrors.AppError {
	if u.gateway == nil {
		return apperrors.NewInternal(
			"PERSISTENCE_GATEWAY_MISSING",
			"persistence gateway is required",
			nil,
		)
	}

	if command.ReadinessTimeout <= 0 {
		return apperrors.NewValidation(
			"READINESS_TIMEOUT_INVALID",
			"readiness timeout must be greater than zero",
			nil,
		)
	}

	if command.ReadinessRetryInterval <= 0 {
		return apperrors.NewValidation(
			"READINESS_RETRY_INTERVAL_INVALID",
			"readiness retry interval must be greater than zero",
			nil,
		)
	}

	readinessCtx, cancel := context.WithTimeout(ctx, command.ReadinessTimeout)
	defer cancel()

	attempts := 0
	for {
		attempts++
		appErr := u.gateway.CheckReadiness(readinessCtx)
		if appErr == nil {
			break
		}

		if readinessCtx.Err() != nil {
			return apperrors.NewInternal(
				"DB_READINESS_TIMEOUT",
				"database readiness check timed out",
				map[string]any{
					"attempts":  strconv.Itoa(attempts),
					"timeout":   command.ReadinessTimeout.String(),
					"last_code": appErr.Code,
				},
			)
		}

		timer := time.NewTimer(command.ReadinessRetryInterval)
		select {
		case <-readinessCtx.Done():
			timer.Stop()
			return apperrors.NewInternal(
				"DB_READINESS_TIMEOUT",
				"database readiness check timed out",
				map[string]any{
					"attempts": strconv.Itoa(attempts),
					"timeout":  command.ReadinessTimeout.String(),
				},
			)
		case <-timer.C:
		}
	}

	if migrationErr := u.gateway.RunMigrations(ctx); migrationErr != nil {
		return migrationErr
	}

	if syncErr := u.gateway.SyncWalletAllocationState(ctx); syncErr != nil {
		return syncErr
	}

	return u.gateway.ValidateAssetCatalogIntegrity(ctx)
}
