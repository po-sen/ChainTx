//go:build !integration

package use_cases

import (
	"context"
	"testing"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

func TestInitializePersistenceUseCaseExecuteSuccess(t *testing.T) {
	fakeGateway := &fakePersistenceGateway{}
	useCase := NewInitializePersistenceUseCase(fakeGateway)

	appErr := useCase.Execute(context.Background(), dto.InitializePersistenceCommand{
		ReadinessTimeout:       50 * time.Millisecond,
		ReadinessRetryInterval: 5 * time.Millisecond,
	})

	if appErr != nil {
		t.Fatalf("expected no error, got %v", appErr)
	}

	if fakeGateway.readinessChecks != 1 {
		t.Fatalf("expected one readiness check, got %d", fakeGateway.readinessChecks)
	}

	if fakeGateway.migrationRuns != 1 {
		t.Fatalf("expected one migration run, got %d", fakeGateway.migrationRuns)
	}

	if fakeGateway.validationRuns != 1 {
		t.Fatalf("expected one validation run, got %d", fakeGateway.validationRuns)
	}
}

func TestInitializePersistenceUseCaseExecuteRetryThenSuccess(t *testing.T) {
	fakeGateway := &fakePersistenceGateway{
		readinessErrors: []*apperrors.AppError{
			apperrors.NewInternal("DB_CONNECT_FAILED", "failed", nil),
			nil,
		},
	}
	useCase := NewInitializePersistenceUseCase(fakeGateway)

	appErr := useCase.Execute(context.Background(), dto.InitializePersistenceCommand{
		ReadinessTimeout:       100 * time.Millisecond,
		ReadinessRetryInterval: 5 * time.Millisecond,
	})

	if appErr != nil {
		t.Fatalf("expected no error, got %v", appErr)
	}

	if fakeGateway.readinessChecks < 2 {
		t.Fatalf("expected at least two readiness checks, got %d", fakeGateway.readinessChecks)
	}
}

func TestInitializePersistenceUseCaseExecuteReadinessTimeout(t *testing.T) {
	fakeGateway := &fakePersistenceGateway{
		readinessErrors: []*apperrors.AppError{
			apperrors.NewInternal("DB_CONNECT_FAILED", "failed", nil),
		},
	}
	useCase := NewInitializePersistenceUseCase(fakeGateway)

	appErr := useCase.Execute(context.Background(), dto.InitializePersistenceCommand{
		ReadinessTimeout:       30 * time.Millisecond,
		ReadinessRetryInterval: 10 * time.Millisecond,
	})

	if appErr == nil {
		t.Fatalf("expected timeout error")
	}

	if appErr.Code != "DB_READINESS_TIMEOUT" {
		t.Fatalf("expected DB_READINESS_TIMEOUT, got %s", appErr.Code)
	}

	if fakeGateway.migrationRuns != 0 {
		t.Fatalf("expected migrations not to run on timeout, got %d", fakeGateway.migrationRuns)
	}

	if fakeGateway.validationRuns != 0 {
		t.Fatalf("expected validation not to run on timeout, got %d", fakeGateway.validationRuns)
	}
}

func TestInitializePersistenceUseCaseExecuteMigrationFailure(t *testing.T) {
	fakeGateway := &fakePersistenceGateway{
		runMigrationErr: apperrors.NewInternal("DB_MIGRATION_APPLY_FAILED", "failed", nil),
	}
	useCase := NewInitializePersistenceUseCase(fakeGateway)

	appErr := useCase.Execute(context.Background(), dto.InitializePersistenceCommand{
		ReadinessTimeout:       50 * time.Millisecond,
		ReadinessRetryInterval: 5 * time.Millisecond,
	})

	if appErr == nil {
		t.Fatalf("expected migration error")
	}

	if appErr.Code != "DB_MIGRATION_APPLY_FAILED" {
		t.Fatalf("expected DB_MIGRATION_APPLY_FAILED, got %s", appErr.Code)
	}

	if fakeGateway.validationRuns != 0 {
		t.Fatalf("expected validation not to run on migration failure, got %d", fakeGateway.validationRuns)
	}
}

func TestInitializePersistenceUseCaseExecuteValidationFailure(t *testing.T) {
	fakeGateway := &fakePersistenceGateway{
		validateCatalogErr: apperrors.NewInternal("ASSET_CATALOG_INVALID", "failed", nil),
	}
	useCase := NewInitializePersistenceUseCase(fakeGateway)

	appErr := useCase.Execute(context.Background(), dto.InitializePersistenceCommand{
		ReadinessTimeout:       50 * time.Millisecond,
		ReadinessRetryInterval: 5 * time.Millisecond,
	})

	if appErr == nil {
		t.Fatalf("expected validation error")
	}

	if appErr.Code != "ASSET_CATALOG_INVALID" {
		t.Fatalf("expected ASSET_CATALOG_INVALID, got %s", appErr.Code)
	}
}

func TestInitializePersistenceUseCaseExecuteInvalidCommand(t *testing.T) {
	fakeGateway := &fakePersistenceGateway{}
	useCase := NewInitializePersistenceUseCase(fakeGateway)

	appErr := useCase.Execute(context.Background(), dto.InitializePersistenceCommand{})
	if appErr == nil {
		t.Fatalf("expected validation error")
	}

	if appErr.Type != apperrors.TypeValidation {
		t.Fatalf("expected validation error type, got %s", appErr.Type)
	}
}

type fakePersistenceGateway struct {
	readinessErrors    []*apperrors.AppError
	runMigrationErr    *apperrors.AppError
	validateCatalogErr *apperrors.AppError
	readinessChecks    int
	migrationRuns      int
	validationRuns     int
}

func (f *fakePersistenceGateway) CheckReadiness(_ context.Context) *apperrors.AppError {
	f.readinessChecks++

	if len(f.readinessErrors) == 0 {
		return nil
	}

	index := f.readinessChecks - 1
	if index >= len(f.readinessErrors) {
		return f.readinessErrors[len(f.readinessErrors)-1]
	}

	return f.readinessErrors[index]
}

func (f *fakePersistenceGateway) RunMigrations(_ context.Context) *apperrors.AppError {
	f.migrationRuns++
	return f.runMigrationErr
}

func (f *fakePersistenceGateway) ValidateAssetCatalogIntegrity(_ context.Context) *apperrors.AppError {
	f.validationRuns++
	return f.validateCatalogErr
}
