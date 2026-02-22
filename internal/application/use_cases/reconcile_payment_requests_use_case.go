package use_cases

import (
	"context"
	"strconv"
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
	if command.ReorgObserveWindow <= 0 {
		return dto.ReconcilePaymentRequestsOutput{}, apperrors.NewValidation(
			"reconcile_reorg_observe_window_invalid",
			"reconcile reorg observe window must be greater than zero",
			map[string]any{"reorg_observe_window": command.ReorgObserveWindow.String()},
		)
	}
	if command.StabilityCycles <= 0 {
		return dto.ReconcilePaymentRequestsOutput{}, apperrors.NewValidation(
			"reconcile_stability_cycles_invalid",
			"reconcile stability cycles must be greater than zero",
			map[string]any{"stability_cycles": command.StabilityCycles},
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
		command.ReorgObserveWindow,
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
		state := parseReconciliationState(row.Metadata)
		if shouldExpireRequest(currentStatus, now, row.ExpiresAt) {
			updated, transitionErr := u.repository.TransitionStatusIfCurrent(
				ctx,
				row.ID,
				currentStatus,
				"expired",
				now,
				workerID,
				dto.ReconcileTransitionMetadata{
					TransitionReason: "payment_expired",
					UpdatedAt:        now,
				},
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

		if shouldSkipConfirmedObservation(currentStatus, state, now, command.ReorgObserveWindow) {
			updated, transitionErr := u.repository.TransitionStatusIfCurrent(
				ctx,
				row.ID,
				currentStatus,
				currentStatus,
				now,
				workerID,
				buildObservationMetadata(
					now,
					dto.ObservePaymentRequestOutput{
						Supported:         true,
						ObservedAmount:    "",
						ObservationSource: "reconcile_policy",
						ObservationDetails: map[string]any{
							"observation_skipped":  true,
							"reason":               "observe_window_elapsed",
							"reorg_observe_window": command.ReorgObserveWindow.String(),
						},
					},
					dto.ReconcileSettlementSyncResult{},
					state,
					currentStatus,
					"",
					false,
					false,
					command.StabilityCycles,
				),
			)
			if transitionErr != nil {
				return output, transitionErr
			}
			if updated {
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
			_, transitionErr := u.repository.TransitionStatusIfCurrent(
				ctx,
				row.ID,
				currentStatus,
				currentStatus,
				now,
				workerID,
				dto.ReconcileTransitionMetadata{
					ObservationSource: "reconcile_policy",
					ObservationDetails: map[string]any{
						"observation_error": observeErr.Code,
					},
					UpdatedAt: now,
				},
			)
			if transitionErr != nil {
				return output, transitionErr
			}
			continue
		}
		if !observation.Supported {
			_, transitionErr := u.repository.TransitionStatusIfCurrent(
				ctx,
				row.ID,
				currentStatus,
				currentStatus,
				now,
				workerID,
				dto.ReconcileTransitionMetadata{
					ObservationSource: "reconcile_policy",
					ObservationDetails: map[string]any{
						"supported": false,
					},
					UpdatedAt: now,
				},
			)
			if transitionErr != nil {
				return output, transitionErr
			}
			output.Skipped++
			continue
		}

		settlementSummary, settlementErr := u.repository.SyncObservedSettlements(
			ctx,
			row.ID,
			row.Chain,
			row.Network,
			row.Asset,
			now,
			observation.Settlements,
		)
		if settlementErr != nil {
			return output, settlementErr
		}

		targetStatus := currentStatus
		transitionReason := ""
		allowImmediateDemote := settlementSummary.NewlyOrphanedCount > 0
		isReconfirm := false

		switch {
		case currentStatus == "confirmed" && (!observation.Confirmed || allowImmediateDemote):
			nextState := nextStabilityState(state, "demote")
			if allowImmediateDemote || nextState.demoteStreak >= command.StabilityCycles {
				targetStatus = "reorged"
				transitionReason = "payment_reorged"
			}
		case observation.Confirmed && (currentStatus == "pending" || currentStatus == "detected" || currentStatus == "reorged"):
			nextState := nextStabilityState(state, "promote")
			if nextState.promoteStreak >= command.StabilityCycles {
				targetStatus = "confirmed"
				if currentStatus == "reorged" {
					transitionReason = "payment_reconfirmed"
					isReconfirm = true
				} else {
					transitionReason = "payment_confirmed"
				}
			}
		case observation.Detected && currentStatus == "pending":
			targetStatus = "detected"
			transitionReason = "payment_detected"
		}

		updated, transitionErr := u.repository.TransitionStatusIfCurrent(
			ctx,
			row.ID,
			currentStatus,
			targetStatus,
			now,
			workerID,
			buildObservationMetadata(
				now,
				observation,
				settlementSummary,
				state,
				currentStatus,
				transitionReason,
				targetStatus == "confirmed",
				targetStatus == "reorged",
				command.StabilityCycles,
			),
		)
		if transitionErr != nil {
			return output, transitionErr
		}
		if !updated {
			output.Skipped++
			continue
		}

		switch targetStatus {
		case "confirmed":
			output.Confirmed++
			if isReconfirm {
				output.Reconfirmed++
			}
		case "detected":
			output.Detected++
		case "reorged":
			output.Reorged++
		case "expired":
			output.Expired++
		default:
			output.Skipped++
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

type reconcileState struct {
	firstConfirmedAt *time.Time
	finalityAt       *time.Time
	stabilitySignal  string
	promoteStreak    int
	demoteStreak     int
}

func parseReconciliationState(metadata map[string]any) reconcileState {
	state := reconcileState{}
	if len(metadata) == 0 {
		return state
	}

	raw, exists := metadata["reconciliation"]
	if !exists {
		return state
	}
	reconciliation, ok := raw.(map[string]any)
	if !ok {
		return state
	}

	state.firstConfirmedAt = parseMetadataTime(reconciliation["first_confirmed_at"])
	state.finalityAt = parseMetadataTime(reconciliation["finality_reached_at"])
	state.stabilitySignal = strings.ToLower(strings.TrimSpace(parseMetadataString(reconciliation["stability_signal"])))
	state.promoteStreak = parseMetadataInt(reconciliation["stability_promote_streak"])
	state.demoteStreak = parseMetadataInt(reconciliation["stability_demote_streak"])
	return state
}

func shouldExpireRequest(currentStatus string, now time.Time, expiresAt time.Time) bool {
	if currentStatus != "pending" && currentStatus != "detected" && currentStatus != "reorged" {
		return false
	}
	return !expiresAt.After(now)
}

func shouldSkipConfirmedObservation(
	currentStatus string,
	state reconcileState,
	now time.Time,
	observeWindow time.Duration,
) bool {
	if currentStatus != "confirmed" || observeWindow <= 0 {
		return false
	}
	if state.firstConfirmedAt == nil || state.finalityAt == nil {
		return false
	}
	return !now.Before(state.firstConfirmedAt.Add(observeWindow))
}

func nextStabilityState(state reconcileState, signal string) reconcileState {
	next := state
	signal = strings.ToLower(strings.TrimSpace(signal))
	switch signal {
	case "promote":
		if state.stabilitySignal == "promote" {
			next.promoteStreak++
		} else {
			next.promoteStreak = 1
		}
		next.demoteStreak = 0
		next.stabilitySignal = "promote"
	case "demote":
		if state.stabilitySignal == "demote" {
			next.demoteStreak++
		} else {
			next.demoteStreak = 1
		}
		next.promoteStreak = 0
		next.stabilitySignal = "demote"
	default:
		next.promoteStreak = 0
		next.demoteStreak = 0
		next.stabilitySignal = ""
	}
	return next
}

func buildObservationMetadata(
	now time.Time,
	observation dto.ObservePaymentRequestOutput,
	settlementSummary dto.ReconcileSettlementSyncResult,
	state reconcileState,
	currentStatus string,
	transitionReason string,
	nextIsConfirmed bool,
	nextIsReorged bool,
	stabilityCycles int,
) dto.ReconcileTransitionMetadata {
	nextState := state
	shouldReset := transitionReason != ""
	if shouldReset {
		nextState = nextStabilityState(state, "")
	} else {
		switch {
		case currentStatus == "confirmed" && (!observation.Confirmed || settlementSummary.NewlyOrphanedCount > 0):
			nextState = nextStabilityState(state, "demote")
		case observation.Confirmed && (currentStatus == "pending" || currentStatus == "detected" || currentStatus == "reorged"):
			nextState = nextStabilityState(state, "promote")
		default:
			nextState = nextStabilityState(state, "")
		}
	}

	if nextIsConfirmed {
		if nextState.firstConfirmedAt == nil {
			ts := now
			nextState.firstConfirmedAt = &ts
		}
		if observation.FinalityReached {
			ts := now
			nextState.finalityAt = &ts
		}
	} else if nextIsReorged {
		nextState.finalityAt = nil
	} else if currentStatus == "confirmed" && observation.Confirmed {
		if nextState.firstConfirmedAt == nil {
			ts := now
			nextState.firstConfirmedAt = &ts
		}
		if observation.FinalityReached && nextState.finalityAt == nil {
			ts := now
			nextState.finalityAt = &ts
		}
	}

	finalityReached := observation.FinalityReached
	metadata := dto.ReconcileTransitionMetadata{
		ObservedAmountMinor: observation.ObservedAmount,
		ObservationSource:   observation.ObservationSource,
		ObservationDetails:  cloneMap(observation.ObservationDetails),
		TransitionReason:    transitionReason,
		FinalityReached:     &finalityReached,
		EvidenceSummary: &dto.ReconcileEvidenceSummary{
			CanonicalCount:     settlementSummary.CanonicalCount,
			NonCanonicalCount:  settlementSummary.NonCanonicalCount,
			NewlyOrphanedCount: settlementSummary.NewlyOrphanedCount,
		},
		FirstConfirmedAt:       nextState.firstConfirmedAt,
		FinalityReachedAt:      nextState.finalityAt,
		StabilitySignal:        nextState.stabilitySignal,
		StabilityPromoteStreak: nextState.promoteStreak,
		StabilityDemoteStreak:  nextState.demoteStreak,
		UpdatedAt:              now,
	}
	if stabilityCycles > 0 {
		if metadata.ObservationDetails == nil {
			metadata.ObservationDetails = map[string]any{}
		}
		metadata.ObservationDetails["stability_cycles_required"] = stabilityCycles
	}
	return metadata
}

func parseMetadataString(value any) string {
	raw, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(raw)
}

func parseMetadataInt(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0
		}
		out, _ := strconv.Atoi(trimmed)
		return out
	default:
		return 0
	}
}

func parseMetadataTime(value any) *time.Time {
	raw, ok := value.(string)
	if !ok {
		return nil
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return nil
	}
	utc := parsed.UTC()
	return &utc
}
