package webhook

import (
	"context"
	"log"
	"time"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
)

type Worker struct {
	enabled         bool
	pollInterval    time.Duration
	batchSize       int
	workerID        string
	leaseDuration   time.Duration
	initialBackoff  time.Duration
	maxBackoff      time.Duration
	retryJitterBPS  int
	retryBudget     int
	dispatchUseCase portsin.DispatchWebhookEventsUseCase
	logger          *log.Logger
}

func NewWorker(
	enabled bool,
	pollInterval time.Duration,
	batchSize int,
	workerID string,
	leaseDuration time.Duration,
	initialBackoff time.Duration,
	maxBackoff time.Duration,
	retryJitterBPS int,
	retryBudget int,
	dispatchUseCase portsin.DispatchWebhookEventsUseCase,
	logger *log.Logger,
) *Worker {
	return &Worker{
		enabled:         enabled,
		pollInterval:    pollInterval,
		batchSize:       batchSize,
		workerID:        workerID,
		leaseDuration:   leaseDuration,
		initialBackoff:  initialBackoff,
		maxBackoff:      maxBackoff,
		retryJitterBPS:  retryJitterBPS,
		retryBudget:     retryBudget,
		dispatchUseCase: dispatchUseCase,
		logger:          logger,
	}
}

func (w *Worker) Enabled() bool {
	return w != nil && w.enabled
}

func (w *Worker) Start(ctx context.Context) {
	if w == nil || !w.enabled || w.dispatchUseCase == nil {
		return
	}

	w.logf(
		"webhook dispatcher started worker_id=%s poll_interval=%s batch_size=%d lease_duration=%s",
		w.workerID,
		w.pollInterval,
		w.batchSize,
		w.leaseDuration,
	)

	w.runCycle(ctx)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logf("webhook dispatcher stopped")
			return
		case <-ticker.C:
			w.runCycle(ctx)
		}
	}
}

func (w *Worker) runCycle(ctx context.Context) {
	startedAt := time.Now().UTC()
	output, appErr := w.dispatchUseCase.Execute(ctx, dto.DispatchWebhookEventsCommand{
		Now:            startedAt,
		BatchSize:      w.batchSize,
		WorkerID:       w.workerID,
		LeaseDuration:  w.leaseDuration,
		InitialBackoff: w.initialBackoff,
		MaxBackoff:     w.maxBackoff,
		RetryJitterBPS: w.retryJitterBPS,
		RetryBudget:    w.retryBudget,
	})
	if appErr != nil {
		w.logf(
			"webhook dispatch cycle failed code=%s message=%s details=%v",
			appErr.Code,
			appErr.Message,
			appErr.Details,
		)
		return
	}

	w.logf(
		"webhook dispatch cycle completed worker_id=%s claimed=%d sent=%d retried=%d failed=%d skipped=%d errors=%d http_2xx=%d http_4xx=%d http_5xx=%d network_error=%d latency_ms=%d",
		w.workerID,
		output.Claimed,
		output.Sent,
		output.Retried,
		output.Failed,
		output.Skipped,
		output.Errors,
		output.HTTP2xxCount,
		output.HTTP4xxCount,
		output.HTTP5xxCount,
		output.NetworkErrorCount,
		time.Since(startedAt).Milliseconds(),
	)
}

func (w *Worker) logf(format string, args ...any) {
	if w.logger == nil {
		return
	}
	w.logger.Printf(format, args...)
}
