package use_cases

import (
	"context"
	"strings"
	"time"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type reconcilePaymentRequestsUseCase struct {
	repository portsout.PaymentRequestReconciliationRepository
	observer   portsout.PaymentChainObserverGateway
}

func NewReconcilePaymentRequestsUseCase(
	repository portsout.PaymentRequestReconciliationRepository,
	observer portsout.PaymentChainObserverGateway,
) portsin.ReconcilePaymentRequestsUseCase {
	return &reconcilePaymentRequestsUseCase{repository: repository, observer: observer}
}

func (u *reconcilePaymentRequestsUseCase) Execute(
	ctx context.Context,
	command dto.ReconcilePaymentRequestsCommand,
) (dto.ReconcilePaymentRequestsOutput, *apperrors.AppError) {
	if u.repository == nil {
		return dto.ReconcilePaymentRequestsOutput{}, apperrors.NewInternal(
			"payment_request_reconciliation_repository_missing",
			"payment request reconciliation repository is required",
			nil,
		)
	}
	if u.observer == nil {
		return dto.ReconcilePaymentRequestsOutput{}, apperrors.NewInternal(
			"payment_chain_observer_gateway_missing",
			"payment chain observer gateway is required",
			nil,
		)
	}
	if command.BatchSize <= 0 {
		return dto.ReconcilePaymentRequestsOutput{}, apperrors.NewValidation(
			"reconcile_batch_size_invalid",
			"reconcile batch size must be greater than zero",
			map[string]any{"batch_size": command.BatchSize},
		)
	}
	workerID := strings.TrimSpace(command.WorkerID)
	if workerID == "" {
		return dto.ReconcilePaymentRequestsOutput{}, apperrors.NewValidation(
			"reconcile_worker_id_invalid",
			"reconcile worker id is required",
			nil,
		)
	}
	if command.LeaseDuration <= 0 {
		return dto.ReconcilePaymentRequestsOutput{}, apperrors.NewValidation(
			"reconcile_lease_duration_invalid",
			"reconcile lease duration must be greater than zero",
			map[string]any{"lease_duration": command.LeaseDuration.String()},
		)
	}

	now := command.Now.UTC()
	if command.Now.IsZero() {
		now = time.Now().UTC()
	}
	leaseUntil := now.Add(command.LeaseDuration)

	rows, appErr := u.repository.ClaimOpenForReconciliation(
		ctx,
		now,
		command.BatchSize,
		workerID,
		leaseUntil,
	)
	if appErr != nil {
		return dto.ReconcilePaymentRequestsOutput{}, appErr
	}

	output := dto.ReconcilePaymentRequestsOutput{
		Claimed: len(rows),
		Scanned: len(rows),
	}
	for _, row := range rows {
		currentStatus := strings.ToLower(strings.TrimSpace(row.Status))
		if !row.ExpiresAt.After(now) {
			updated, transitionErr := u.repository.TransitionStatusIfCurrent(
				ctx,
				row.ID,
				currentStatus,
				"expired",
				now,
				workerID,
				dto.ReconcileTransitionMetadata{UpdatedAt: now},
			)
			if transitionErr != nil {
				return output, transitionErr
			}
			if updated {
				output.Expired++
			} else {
				output.Skipped++
			}
			continue
		}

		observation, observeErr := u.observer.ObservePaymentRequest(ctx, dto.ObservePaymentRequestInput{
			RequestID:           row.ID,
			Chain:               row.Chain,
			Network:             row.Network,
			Asset:               row.Asset,
			ExpectedAmountMinor: row.ExpectedAmountMinor,
			AddressCanonical:    row.AddressCanonical,
			ChainID:             row.ChainID,
			TokenStandard:       row.TokenStandard,
			TokenContract:       row.TokenContract,
			TokenDecimals:       row.TokenDecimals,
		})
		if observeErr != nil {
			output.Errors++
			continue
		}
		if !observation.Supported {
			output.Skipped++
			continue
		}

		targetStatus := ""
		switch {
		case observation.Confirmed:
			targetStatus = "confirmed"
		case observation.Detected && currentStatus == "pending":
			targetStatus = "detected"
		default:
			output.Skipped++
			continue
		}

		updated, transitionErr := u.repository.TransitionStatusIfCurrent(
			ctx,
			row.ID,
			currentStatus,
			targetStatus,
			now,
			workerID,
			dto.ReconcileTransitionMetadata{
				ObservedAmountMinor: observation.ObservedAmount,
				ObservationSource:   observation.ObservationSource,
				ObservationDetails:  cloneMap(observation.ObservationDetails),
				UpdatedAt:           now,
			},
		)
		if transitionErr != nil {
			return output, transitionErr
		}
		if !updated {
			output.Skipped++
			continue
		}

		if targetStatus == "confirmed" {
			output.Confirmed++
		} else {
			output.Detected++
		}
	}

	return output, nil
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
