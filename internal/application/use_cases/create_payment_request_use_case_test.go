package use_cases

import (
	"context"
	"strings"
	"testing"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

func TestCreatePaymentRequestUseCaseExecuteSuccessWithDefaults(t *testing.T) {
	readModel := fakeAssetCatalogReadModel{
		entries: []dto.AssetCatalogEntry{
			{
				Chain:                   "bitcoin",
				Network:                 "mainnet",
				Asset:                   "BTC",
				MinorUnit:               "sats",
				Decimals:                8,
				AddressScheme:           "bip84_p2wpkh",
				DefaultExpiresInSeconds: 3600,
				WalletAccountID:         "wa_btc",
			},
		},
	}

	clock := fixedClock{now: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)}
	repository := &fakePaymentRequestRepository{
		onCreate: func(command dto.CreatePaymentRequestPersistenceCommand) {
			if command.Chain != "bitcoin" || command.Network != "mainnet" || command.Asset != "BTC" {
				t.Fatalf("unexpected normalized tuple: %+v", command)
			}
			if command.IdempotencyScope.PrincipalID != "anonymous" {
				t.Fatalf("expected default principal anonymous, got %q", command.IdempotencyScope.PrincipalID)
			}
			if command.IdempotencyScope.HTTPMethod != "POST" {
				t.Fatalf("expected POST method, got %q", command.IdempotencyScope.HTTPMethod)
			}
			if command.IdempotencyScope.HTTPPath != "/v1/payment-requests" {
				t.Fatalf("expected normalized path, got %q", command.IdempotencyScope.HTTPPath)
			}
			if command.ExpiresAt.Sub(command.CreatedAt) != 3600*time.Second {
				t.Fatalf("unexpected resolved expiry duration: %s", command.ExpiresAt.Sub(command.CreatedAt))
			}
			if command.IdempotencyExpiresAt.Before(command.ExpiresAt) {
				t.Fatalf("expected idempotency expiry >= request expiry")
			}
		},
		result: dto.CreatePaymentRequestPersistenceResult{
			Resource: dto.PaymentRequestResource{ID: "pr_test", Status: "pending"},
			Replayed: false,
		},
	}

	useCase := NewCreatePaymentRequestUseCase(readModel, repository, clock)
	output, appErr := useCase.Execute(context.Background(), dto.CreatePaymentRequestCommand{
		Chain:   "Bitcoin",
		Network: "Mainnet",
		Asset:   "btc",
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}

	if output.Resource.ID != "pr_test" {
		t.Fatalf("expected pr_test id, got %q", output.Resource.ID)
	}
	if repository.createCalls != 1 {
		t.Fatalf("expected one create call, got %d", repository.createCalls)
	}
}

func TestCreatePaymentRequestUseCaseExecuteUnsupportedAsset(t *testing.T) {
	useCase := NewCreatePaymentRequestUseCase(fakeAssetCatalogReadModel{
		entries: []dto.AssetCatalogEntry{
			{Chain: "bitcoin", Network: "mainnet", Asset: "BTC", DefaultExpiresInSeconds: 3600},
		},
	}, &fakePaymentRequestRepository{}, fixedClock{now: time.Now().UTC()})

	_, appErr := useCase.Execute(context.Background(), dto.CreatePaymentRequestCommand{
		Chain:   "bitcoin",
		Network: "mainnet",
		Asset:   "USDT",
	})
	if appErr == nil {
		t.Fatalf("expected unsupported asset error")
	}
	if appErr.Code != "unsupported_asset" {
		t.Fatalf("expected unsupported_asset, got %s", appErr.Code)
	}
}

func TestCreatePaymentRequestUseCaseExecuteExpectedAmountValidation(t *testing.T) {
	amount := "1.25"
	useCase := NewCreatePaymentRequestUseCase(fakeAssetCatalogReadModel{}, &fakePaymentRequestRepository{}, fixedClock{now: time.Now().UTC()})

	_, appErr := useCase.Execute(context.Background(), dto.CreatePaymentRequestCommand{
		Chain:               "bitcoin",
		Network:             "mainnet",
		Asset:               "BTC",
		ExpectedAmountMinor: &amount,
	})
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
	if appErr.Code != "invalid_request" {
		t.Fatalf("expected invalid_request, got %s", appErr.Code)
	}
}

func TestCreatePaymentRequestUseCaseExecuteMetadataTooLarge(t *testing.T) {
	useCase := NewCreatePaymentRequestUseCase(fakeAssetCatalogReadModel{}, &fakePaymentRequestRepository{}, fixedClock{now: time.Now().UTC()})

	metadata := map[string]any{
		"blob": strings.Repeat("a", 5000),
	}
	_, appErr := useCase.Execute(context.Background(), dto.CreatePaymentRequestCommand{
		Chain:    "bitcoin",
		Network:  "mainnet",
		Asset:    "BTC",
		Metadata: metadata,
	})
	if appErr == nil {
		t.Fatalf("expected validation error")
	}
	if appErr.Code != "invalid_request" {
		t.Fatalf("expected invalid_request, got %s", appErr.Code)
	}
}

func TestHashCreateRequestDeterministicForEquivalentJSON(t *testing.T) {
	first, appErr := hashCreateRequest(createRequestHashInput{
		Chain:            "bitcoin",
		Network:          "mainnet",
		Asset:            "BTC",
		ExpiresInSeconds: 3600,
		Metadata: map[string]any{
			"order": map[string]any{"id": "A1", "amount": "10"},
			"tags":  []any{"x", "y"},
		},
	})
	if appErr != nil {
		t.Fatalf("expected no hash error, got %+v", appErr)
	}

	second, appErr := hashCreateRequest(createRequestHashInput{
		Chain:            "bitcoin",
		Network:          "mainnet",
		Asset:            "BTC",
		ExpiresInSeconds: 3600,
		Metadata: map[string]any{
			"tags":  []any{"x", "y"},
			"order": map[string]any{"amount": "10", "id": "A1"},
		},
	})
	if appErr != nil {
		t.Fatalf("expected no hash error, got %+v", appErr)
	}

	if first != second {
		t.Fatalf("expected deterministic hash, got first=%s second=%s", first, second)
	}
}

type fakeAssetCatalogReadModel struct {
	entries []dto.AssetCatalogEntry
	appErr  *apperrors.AppError
}

func (f fakeAssetCatalogReadModel) ListEnabled(_ context.Context) ([]dto.AssetCatalogEntry, *apperrors.AppError) {
	if f.appErr != nil {
		return nil, f.appErr
	}

	return f.entries, nil
}

type fakePaymentRequestRepository struct {
	onCreate    func(command dto.CreatePaymentRequestPersistenceCommand)
	result      dto.CreatePaymentRequestPersistenceResult
	appErr      *apperrors.AppError
	createCalls int
}

func (f *fakePaymentRequestRepository) Create(_ context.Context, command dto.CreatePaymentRequestPersistenceCommand) (dto.CreatePaymentRequestPersistenceResult, *apperrors.AppError) {
	f.createCalls++
	if f.onCreate != nil {
		f.onCreate(command)
	}
	if f.appErr != nil {
		return dto.CreatePaymentRequestPersistenceResult{}, f.appErr
	}

	if f.result.Resource.ID == "" {
		f.result.Resource = dto.PaymentRequestResource{ID: command.ResourceID, Status: "pending"}
	}
	return f.result, nil
}

type fixedClock struct {
	now time.Time
}

func (f fixedClock) NowUTC() time.Time {
	return f.now.UTC()
}
