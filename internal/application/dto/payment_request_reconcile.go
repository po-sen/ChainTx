package dto

import "time"

type ReconcilePaymentRequestsCommand struct {
	Now                time.Time
	BatchSize          int
	WorkerID           string
	LeaseDuration      time.Duration
	ReorgObserveWindow time.Duration
	StabilityCycles    int
}

type ReconcilePaymentRequestsOutput struct {
	Claimed     int
	Scanned     int
	Confirmed   int
	Detected    int
	Reorged     int
	Reconfirmed int
	Expired     int
	Skipped     int
	Errors      int
}

type OpenPaymentRequestForReconciliation struct {
	ID                  string
	Status              string
	Chain               string
	Network             string
	Asset               string
	ExpectedAmountMinor *string
	AddressCanonical    string
	ExpiresAt           time.Time
	ChainID             *int64
	TokenStandard       *string
	TokenContract       *string
	TokenDecimals       *int
	Metadata            map[string]any
}

type ObservePaymentRequestInput struct {
	RequestID           string
	Chain               string
	Network             string
	Asset               string
	ExpectedAmountMinor *string
	AddressCanonical    string
	ChainID             *int64
	TokenStandard       *string
	TokenContract       *string
	TokenDecimals       *int
}

type ObservePaymentRequestOutput struct {
	Supported          bool
	ObservedAmount     string
	Detected           bool
	Confirmed          bool
	FinalityReached    bool
	ObservationSource  string
	ObservationDetails map[string]any
	Settlements        []ObservedSettlementEvidence
}

type ObservedSettlementEvidence struct {
	EvidenceRef   string
	AmountMinor   string
	Confirmations int
	IsCanonical   bool
	BlockHeight   *int64
	BlockHash     *string
	Metadata      map[string]any
}

type ReconcileSettlementSyncResult struct {
	CanonicalCount     int
	NonCanonicalCount  int
	NewlyOrphanedCount int
}

type ReconcileTransitionMetadata struct {
	ObservedAmountMinor    string                    `json:"observed_amount_minor,omitempty"`
	ObservationSource      string                    `json:"observation_source,omitempty"`
	ObservationDetails     map[string]any            `json:"observation_details,omitempty"`
	TransitionReason       string                    `json:"transition_reason,omitempty"`
	FinalityReached        *bool                     `json:"finality_reached,omitempty"`
	EvidenceSummary        *ReconcileEvidenceSummary `json:"evidence_summary,omitempty"`
	FirstConfirmedAt       *time.Time                `json:"first_confirmed_at,omitempty"`
	FinalityReachedAt      *time.Time                `json:"finality_reached_at,omitempty"`
	StabilitySignal        string                    `json:"stability_signal,omitempty"`
	StabilityPromoteStreak int                       `json:"stability_promote_streak,omitempty"`
	StabilityDemoteStreak  int                       `json:"stability_demote_streak,omitempty"`
	UpdatedAt              time.Time                 `json:"updated_at"`
}

type ReconcileEvidenceSummary struct {
	CanonicalCount     int `json:"canonical_count,omitempty"`
	NonCanonicalCount  int `json:"non_canonical_count,omitempty"`
	NewlyOrphanedCount int `json:"newly_orphaned_count,omitempty"`
}
