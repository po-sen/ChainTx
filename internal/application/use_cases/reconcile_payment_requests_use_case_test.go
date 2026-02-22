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
		BatchSize:          1,
		WorkerID:           "worker-a",
		ReorgObserveWindow: 24 * time.Hour,
		StabilityCycles:    1,
	})
	if appErr == nil {
		t.Fatalf("expected error")
	}
	if appErr.Code != "reconcile_lease_duration_invalid" {
		t.Fatalf("expected reconcile_lease_duration_invalid, got %s", appErr.Code)
	}
}

func TestReconcilePaymentRequestsUseCaseRequiresObserveWindow(t *testing.T) {
	useCase := NewReconcilePaymentRequestsUseCase(&fakeReconcileRepository{}, &fakeObserverGateway{})

	_, appErr := useCase.Execute(context.Background(), dto.ReconcilePaymentRequestsCommand{
		BatchSize:       1,
		WorkerID:        "worker-a",
		LeaseDuration:   30 * time.Second,
		StabilityCycles: 1,
	})
	if appErr == nil {
		t.Fatalf("expected error")
	}
	if appErr.Code != "reconcile_reorg_observe_window_invalid" {
		t.Fatalf("expected reconcile_reorg_observe_window_invalid, got %s", appErr.Code)
	}
}

func TestReconcilePaymentRequestsUseCaseRequiresStabilityCycles(t *testing.T) {
	useCase := NewReconcilePaymentRequestsUseCase(&fakeReconcileRepository{}, &fakeObserverGateway{})

	_, appErr := useCase.Execute(context.Background(), dto.ReconcilePaymentRequestsCommand{
		BatchSize:          1,
		WorkerID:           "worker-a",
		LeaseDuration:      30 * time.Second,
		ReorgObserveWindow: 24 * time.Hour,
		StabilityCycles:    0,
	})
	if appErr == nil {
		t.Fatalf("expected error")
	}
	if appErr.Code != "reconcile_stability_cycles_invalid" {
		t.Fatalf("expected reconcile_stability_cycles_invalid, got %s", appErr.Code)
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
		Now:                now,
		BatchSize:          50,
		WorkerID:           "worker-a",
		LeaseDuration:      30 * time.Second,
		ReorgObserveWindow: 24 * time.Hour,
		StabilityCycles:    1,
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
				FinalityReached:   false,
				ObservationSource: "btc_esplora",
				Settlements: []dto.ObservedSettlementEvidence{
					{EvidenceRef: "btc:chain_stats", AmountMinor: "1000", Confirmations: 1, IsCanonical: true},
				},
			},
			"pr_confirm": {
				Supported:         true,
				ObservedAmount:    "200",
				Detected:          true,
				Confirmed:         true,
				FinalityReached:   true,
				ObservationSource: "evm_rpc",
				Settlements: []dto.ObservedSettlementEvidence{
					{EvidenceRef: "evm:confirmed_snapshot", AmountMinor: "200", Confirmations: 3, IsCanonical: true},
				},
			},
		},
	}
	useCase := NewReconcilePaymentRequestsUseCase(repo, observer)

	output, appErr := useCase.Execute(context.Background(), dto.ReconcilePaymentRequestsCommand{
		Now:                now,
		BatchSize:          50,
		WorkerID:           "worker-a",
		LeaseDuration:      30 * time.Second,
		ReorgObserveWindow: 24 * time.Hour,
		StabilityCycles:    1,
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
		Now:                now,
		BatchSize:          10,
		WorkerID:           "worker-a",
		LeaseDuration:      30 * time.Second,
		ReorgObserveWindow: 24 * time.Hour,
		StabilityCycles:    1,
	})
	if appErr != nil {
		t.Fatalf("expected no fatal error, got %+v", appErr)
	}
	if output.Errors != 1 {
		t.Fatalf("expected errors=1, got %d", output.Errors)
	}
	if len(repo.transitions) != 1 || repo.transitions[0].currentStatus != repo.transitions[0].nextStatus {
		t.Fatalf("expected metadata-only transition, got %+v", repo.transitions)
	}
}

func TestReconcilePaymentRequestsUseCaseReorgAndReconfirm(t *testing.T) {
	now := time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC)
	reconcileMeta := map[string]any{
		"reconciliation": map[string]any{
			"first_confirmed_at":       now.Add(-5 * time.Minute).Format(time.RFC3339Nano),
			"stability_signal":         "demote",
			"stability_demote_streak":  1,
			"stability_promote_streak": 0,
			"finality_reached_at":      now.Add(-5 * time.Minute).Format(time.RFC3339Nano),
		},
	}
	repo := &fakeReconcileRepository{
		rows: []dto.OpenPaymentRequestForReconciliation{
			{
				ID:               "pr_reorg",
				Status:           "confirmed",
				Chain:            "bitcoin",
				Network:          "regtest",
				Asset:            "BTC",
				AddressCanonical: "bcrt1x",
				ExpiresAt:        now.Add(10 * time.Minute),
				Metadata:         reconcileMeta,
			},
			{
				ID:               "pr_reconfirm",
				Status:           "reorged",
				Chain:            "bitcoin",
				Network:          "regtest",
				Asset:            "BTC",
				AddressCanonical: "bcrt1y",
				ExpiresAt:        now.Add(10 * time.Minute),
			},
		},
		settlementSummaries: map[string]dto.ReconcileSettlementSyncResult{
			"pr_reorg":     {CanonicalCount: 0, NonCanonicalCount: 1, NewlyOrphanedCount: 1},
			"pr_reconfirm": {CanonicalCount: 1, NonCanonicalCount: 0, NewlyOrphanedCount: 0},
		},
	}
	observer := &fakeObserverGateway{
		responses: map[string]dto.ObservePaymentRequestOutput{
			"pr_reorg": {
				Supported:         true,
				ObservedAmount:    "0",
				Detected:          false,
				Confirmed:         false,
				FinalityReached:   false,
				ObservationSource: "btc_esplora",
				Settlements: []dto.ObservedSettlementEvidence{
					{EvidenceRef: "btc:chain_stats", AmountMinor: "0", Confirmations: 0, IsCanonical: true},
				},
			},
			"pr_reconfirm": {
				Supported:         true,
				ObservedAmount:    "1000",
				Detected:          false,
				Confirmed:         true,
				FinalityReached:   true,
				ObservationSource: "btc_esplora",
				Settlements: []dto.ObservedSettlementEvidence{
					{EvidenceRef: "btc:chain_stats", AmountMinor: "1000", Confirmations: 2, IsCanonical: true},
				},
			},
		},
	}
	useCase := NewReconcilePaymentRequestsUseCase(repo, observer)

	output, appErr := useCase.Execute(context.Background(), dto.ReconcilePaymentRequestsCommand{
		Now:                now,
		BatchSize:          10,
		WorkerID:           "worker-a",
		LeaseDuration:      30 * time.Second,
		ReorgObserveWindow: 24 * time.Hour,
		StabilityCycles:    1,
	})
	if appErr != nil {
		t.Fatalf("expected no fatal error, got %+v", appErr)
	}
	if output.Reorged != 1 {
		t.Fatalf("expected reorged=1, got %+v", output)
	}
	if output.Reconfirmed != 1 {
		t.Fatalf("expected reconfirmed=1, got %+v", output)
	}
}

type fakeReconcileRepository struct {
	rows                []dto.OpenPaymentRequestForReconciliation
	claimErr            *apperrors.AppError
	settlementErr       *apperrors.AppError
	transitionErr       *apperrors.AppError
	transitionAccepted  bool
	transitions         []fakeTransition
	leaseOwners         []string
	settlementSummaries map[string]dto.ReconcileSettlementSyncResult
}

type fakeTransition struct {
	id            string
	currentStatus string
	nextStatus    string
}

func (f *fakeReconcileRepository) ClaimOpenForReconciliation(
	_ context.Context,
	_ time.Time,
	_ time.Duration,
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

func (f *fakeReconcileRepository) SyncObservedSettlements(
	_ context.Context,
	requestID string,
	_ string,
	_ string,
	_ string,
	_ time.Time,
	_ []dto.ObservedSettlementEvidence,
) (dto.ReconcileSettlementSyncResult, *apperrors.AppError) {
	if f.settlementErr != nil {
		return dto.ReconcileSettlementSyncResult{}, f.settlementErr
	}
	if f.settlementSummaries != nil {
		if summary, exists := f.settlementSummaries[requestID]; exists {
			return summary, nil
		}
	}
	return dto.ReconcileSettlementSyncResult{}, nil
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
