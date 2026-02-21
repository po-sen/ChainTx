package dto

import "time"

type DispatchWebhookEventsCommand struct {
	Now            time.Time
	BatchSize      int
	WorkerID       string
	LeaseDuration  time.Duration
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

type DispatchWebhookEventsOutput struct {
	Claimed   int
	Sent      int
	Retried   int
	Failed    int
	Skipped   int
	Errors    int
	LatencyMS int64
}

type PendingWebhookOutboxEvent struct {
	ID             int64
	EventID        string
	EventType      string
	DestinationURL string
	Payload        []byte
	Attempts       int
	MaxAttempts    int
}

type SendWebhookEventInput struct {
	EventID         string
	EventType       string
	DeliveryAttempt int
	DestinationURL  string
	Payload         []byte
}

type SendWebhookEventOutput struct {
	StatusCode int
}
