package dto

import "time"

type GetWebhookOutboxOverviewQuery struct {
	Now time.Time
}

type WebhookOutboxOverview struct {
	PendingCount           int64      `json:"pending_count"`
	PendingReadyCount      int64      `json:"pending_ready_count"`
	RetryingCount          int64      `json:"retrying_count"`
	FailedCount            int64      `json:"failed_count"`
	DeliveredCount         int64      `json:"delivered_count"`
	OldestPendingCreatedAt *time.Time `json:"oldest_pending_created_at,omitempty"`
	OldestPendingAgeSec    *int64     `json:"oldest_pending_age_seconds,omitempty"`
}

type ListWebhookDLQEventsQuery struct {
	Limit int
}

type WebhookDLQEvent struct {
	EventID          string     `json:"event_id"`
	EventType        string     `json:"event_type"`
	PaymentRequestID string     `json:"payment_request_id"`
	DestinationURL   string     `json:"destination_url"`
	Attempts         int        `json:"attempts"`
	MaxAttempts      int        `json:"max_attempts"`
	LastError        *string    `json:"last_error,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeliveredAt      *time.Time `json:"delivered_at,omitempty"`
}

type ListWebhookDLQEventsOutput struct {
	Events []WebhookDLQEvent `json:"events"`
}

type RequeueWebhookDLQEventCommand struct {
	EventID    string
	OperatorID string
	Now        time.Time
}

type RequeueWebhookDLQEventOutput struct {
	EventID        string    `json:"event_id"`
	DeliveryStatus string    `json:"delivery_status"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CancelWebhookOutboxEventCommand struct {
	EventID    string
	OperatorID string
	Reason     string
	Now        time.Time
}

type CancelWebhookOutboxEventOutput struct {
	EventID        string    `json:"event_id"`
	DeliveryStatus string    `json:"delivery_status"`
	LastError      string    `json:"last_error"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type WebhookOutboxMutationResult struct {
	Found         bool
	Updated       bool
	CurrentStatus string
}
