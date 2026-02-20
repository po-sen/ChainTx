package dto

import "time"

type ReconcilePaymentRequestsCommand struct {
	Now           time.Time
	BatchSize     int
	WorkerID      string
	LeaseDuration time.Duration
}

type ReconcilePaymentRequestsOutput struct {
	Claimed   int
	Scanned   int
	Confirmed int
	Detected  int
	Expired   int
	Skipped   int
	Errors    int
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
	ObservationSource  string
	ObservationDetails map[string]any
}

type ReconcileTransitionMetadata struct {
	ObservedAmountMinor string         `json:"observed_amount_minor,omitempty"`
	ObservationSource   string         `json:"observation_source,omitempty"`
	ObservationDetails  map[string]any `json:"observation_details,omitempty"`
	UpdatedAt           time.Time      `json:"updated_at"`
}
