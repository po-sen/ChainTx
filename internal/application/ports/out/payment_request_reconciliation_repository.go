package out

import (
	"context"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type PaymentRequestReconciliationRepository interface {
	ClaimOpenForReconciliation(
		ctx context.Context,
		now time.Time,
		observeWindow time.Duration,
		limit int,
		leaseOwner string,
		leaseUntil time.Time,
	) ([]dto.OpenPaymentRequestForReconciliation, *apperrors.AppError)
	SyncObservedSettlements(
		ctx context.Context,
		requestID string,
		chain string,
		network string,
		asset string,
		observedAt time.Time,
		settlements []dto.ObservedSettlementEvidence,
	) (dto.ReconcileSettlementSyncResult, *apperrors.AppError)
	TransitionStatusIfCurrent(
		ctx context.Context,
		id string,
		currentStatus string,
		nextStatus string,
		updatedAt time.Time,
		leaseOwner string,
		metadata dto.ReconcileTransitionMetadata,
	) (bool, *apperrors.AppError)
}
