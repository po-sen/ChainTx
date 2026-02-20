//go:build !integration

package use_cases

import (
	"context"
	"testing"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

func TestReconcilePaymentRequestsUseCaseInvalidBatch(t *testing.T) {
	useCase := NewReconcilePaymentRequestsUseCase(&fakeReconcileRepository{}, &fakeObserverGateway{})

	_, appErr := useCase.Execute(context.Background(), dto.ReconcilePaymentRequestsCommand{BatchSize: 0})
	if appErr == nil {
		t.Fatalf("expected error")
	}
	if appErr.Code != "reconcile_batch_size_invalid" {
		t.Fatalf("expected reconcile_batch_size_invalid, got %s", appErr.Code)
	}
}

func TestReconcilePaymentRequestsUseCaseRequiresWorkerID(t *testing.T) {
	useCase := NewReconcilePaymentRequestsUseCase(&fakeReconcileRepository{}, &fakeObserverGateway{})

	_, appErr := useCase.Execute(context.Background(), dto.ReconcilePaymentRequestsCommand{
		BatchSize:     1,
		LeaseDuration: 5 * time.Second,
	})
	if appErr == nil {
		t.Fatalf("expected error")
	}
	if appErr.Code != "reconcile_worker_id_invalid" {
		t.Fatalf("expected reconcile_worker_id_invalid, got %s", appErr.Code)
	}
}

func TestReconcilePaymentRequestsUseCaseRequiresLeaseDuration(t *testing.T) {
	useCase := NewReconcilePaymentRequestsUseCase(&fakeReconcileRepository{}, &fakeObserverGateway{})

	_, appErr := useCase.Execute(context.Background(), dto.ReconcilePaymentRequestsCommand{
		BatchSize: 1,
		WorkerID:  "worker-a",
	})
	if appErr == nil {
		t.Fatalf("expected error")
	}
	if appErr.Code != "reconcile_lease_duration_invalid" {
		t.Fatalf("expected reconcile_lease_duration_invalid, got %s", appErr.Code)
	}
}

func TestReconcilePaymentRequestsUseCaseExpiryTransition(t *testing.T) {
	now := time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC)
	repo := &fakeReconcileRepository{
		rows: []dto.OpenPaymentRequestForReconciliation{
			{
				ID:        "pr_1",
				Status:    "pending",
				Chain:     "bitcoin",
				Network:   "regtest",
				Asset:     "BTC",
				ExpiresAt: now.Add(-1 * time.Minute),
			},
		},
	}
	useCase := NewReconcilePaymentRequestsUseCase(repo, &fakeObserverGateway{})

	output, appErr := useCase.Execute(context.Background(), dto.ReconcilePaymentRequestsCommand{
		Now:           now,
		BatchSize:     50,
		WorkerID:      "worker-a",
		LeaseDuration: 30 * time.Second,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if output.Expired != 1 {
		t.Fatalf("expected expired=1, got %d", output.Expired)
	}
	if len(repo.transitions) != 1 || repo.transitions[0].nextStatus != "expired" {
		t.Fatalf("expected expired transition, got %+v", repo.transitions)
	}
}

func TestReconcilePaymentRequestsUseCaseDetectedAndConfirmed(t *testing.T) {
	now := time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC)
	repo := &fakeReconcileRepository{
		rows: []dto.OpenPaymentRequestForReconciliation{
			{
				ID:                  "pr_detect",
				Status:              "pending",
				Chain:               "bitcoin",
				Network:             "regtest",
				Asset:               "BTC",
				ExpectedAmountMinor: ptrString("1000"),
				AddressCanonical:    "bcrt1x",
				ExpiresAt:           now.Add(10 * time.Minute),
			},
			{
				ID:                  "pr_confirm",
				Status:              "detected",
				Chain:               "ethereum",
				Network:             "local",
				Asset:               "ETH",
				ExpectedAmountMinor: ptrString("100"),
				AddressCanonical:    "0xabc",
				ExpiresAt:           now.Add(10 * time.Minute),
			},
		},
	}
	observer := &fakeObserverGateway{
		responses: map[string]dto.ObservePaymentRequestOutput{
			"pr_detect": {
				Supported:         true,
				ObservedAmount:    "1000",
				Detected:          true,
				Confirmed:         false,
				ObservationSource: "btc_esplora",
			},
			"pr_confirm": {
				Supported:         true,
				ObservedAmount:    "200",
				Detected:          true,
				Confirmed:         true,
				ObservationSource: "evm_rpc",
			},
		},
	}
	useCase := NewReconcilePaymentRequestsUseCase(repo, observer)

	output, appErr := useCase.Execute(context.Background(), dto.ReconcilePaymentRequestsCommand{
		Now:           now,
		BatchSize:     50,
		WorkerID:      "worker-a",
		LeaseDuration: 30 * time.Second,
	})
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if output.Detected != 1 {
		t.Fatalf("expected detected=1, got %d", output.Detected)
	}
	if output.Confirmed != 1 {
		t.Fatalf("expected confirmed=1, got %d", output.Confirmed)
	}
	if len(repo.transitions) != 2 {
		t.Fatalf("expected 2 transitions, got %d", len(repo.transitions))
	}
}

func TestReconcilePaymentRequestsUseCaseObserverErrorContinues(t *testing.T) {
	now := time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC)
	repo := &fakeReconcileRepository{
		rows: []dto.OpenPaymentRequestForReconciliation{
			{
				ID:                  "pr_err",
				Status:              "pending",
				Chain:               "bitcoin",
				Network:             "regtest",
				Asset:               "BTC",
				AddressCanonical:    "bcrt1x",
				ExpectedAmountMinor: ptrString("1000"),
				ExpiresAt:           now.Add(10 * time.Minute),
			},
		},
	}
	observer := &fakeObserverGateway{
		errors: map[string]*apperrors.AppError{
			"pr_err": apperrors.NewInternal("observer_failed", "failed", nil),
		},
	}
	useCase := NewReconcilePaymentRequestsUseCase(repo, observer)

	output, appErr := useCase.Execute(context.Background(), dto.ReconcilePaymentRequestsCommand{
		Now:           now,
		BatchSize:     10,
		WorkerID:      "worker-a",
		LeaseDuration: 30 * time.Second,
	})
	if appErr != nil {
		t.Fatalf("expected no fatal error, got %+v", appErr)
	}
	if output.Errors != 1 {
		t.Fatalf("expected errors=1, got %d", output.Errors)
	}
	if len(repo.transitions) != 0 {
		t.Fatalf("expected no transitions, got %d", len(repo.transitions))
	}
}

type fakeReconcileRepository struct {
	rows               []dto.OpenPaymentRequestForReconciliation
	claimErr           *apperrors.AppError
	transitionErr      *apperrors.AppError
	transitionAccepted bool
	transitions        []fakeTransition
	leaseOwners        []string
}

type fakeTransition struct {
	id            string
	currentStatus string
	nextStatus    string
}

func (f *fakeReconcileRepository) ClaimOpenForReconciliation(
	_ context.Context,
	_ time.Time,
	_ int,
	leaseOwner string,
	_ time.Time,
) ([]dto.OpenPaymentRequestForReconciliation, *apperrors.AppError) {
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	f.leaseOwners = append(f.leaseOwners, leaseOwner)
	return f.rows, nil
}

func (f *fakeReconcileRepository) TransitionStatusIfCurrent(
	_ context.Context,
	id string,
	currentStatus string,
	nextStatus string,
	_ time.Time,
	leaseOwner string,
	_ dto.ReconcileTransitionMetadata,
) (bool, *apperrors.AppError) {
	if f.transitionErr != nil {
		return false, f.transitionErr
	}
	f.transitions = append(f.transitions, fakeTransition{
		id:            id,
		currentStatus: currentStatus,
		nextStatus:    nextStatus,
	})
	f.leaseOwners = append(f.leaseOwners, leaseOwner)
	if f.transitionAccepted {
		return true, nil
	}
	return true, nil
}

type fakeObserverGateway struct {
	responses map[string]dto.ObservePaymentRequestOutput
	errors    map[string]*apperrors.AppError
}

func (f *fakeObserverGateway) ObservePaymentRequest(_ context.Context, input dto.ObservePaymentRequestInput) (dto.ObservePaymentRequestOutput, *apperrors.AppError) {
	if f.errors != nil {
		if appErr, exists := f.errors[input.RequestID]; exists {
			return dto.ObservePaymentRequestOutput{}, appErr
		}
	}
	if f.responses != nil {
		if response, exists := f.responses[input.RequestID]; exists {
			return response, nil
		}
	}
	return dto.ObservePaymentRequestOutput{Supported: false}, nil
}

func ptrString(value string) *string {
	return &value
}
