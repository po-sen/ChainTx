package reconciler

import (
	"context"
	"log"
	"time"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
)

type Worker struct {
	enabled       bool
	pollInterval  time.Duration
	batchSize     int
	workerID      string
	leaseDuration time.Duration
	useCase       portsin.ReconcilePaymentRequestsUseCase
	logger        *log.Logger
}

func NewWorker(
	enabled bool,
	pollInterval time.Duration,
	batchSize int,
	workerID string,
	leaseDuration time.Duration,
	useCase portsin.ReconcilePaymentRequestsUseCase,
	logger *log.Logger,
) *Worker {
	return &Worker{
		enabled:       enabled,
		pollInterval:  pollInterval,
		batchSize:     batchSize,
		workerID:      workerID,
		leaseDuration: leaseDuration,
		useCase:       useCase,
		logger:        logger,
	}
}

func (w *Worker) Enabled() bool {
	return w != nil && w.enabled
}

func (w *Worker) Start(ctx context.Context) {
	if w == nil || !w.enabled || w.useCase == nil {
		return
	}

	w.logf(
		"payment request reconciler started worker_id=%s poll_interval=%s batch_size=%d lease_duration=%s",
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
			w.logf("payment request reconciler stopped")
			return
		case <-ticker.C:
			w.runCycle(ctx)
		}
	}
}

func (w *Worker) runCycle(ctx context.Context) {
	startedAt := time.Now().UTC()
	output, appErr := w.useCase.Execute(ctx, dto.ReconcilePaymentRequestsCommand{
		Now:           startedAt,
		BatchSize:     w.batchSize,
		WorkerID:      w.workerID,
		LeaseDuration: w.leaseDuration,
	})
	if appErr != nil {
		w.logf(
			"payment request reconcile cycle failed code=%s message=%s details=%v",
			appErr.Code,
			appErr.Message,
			appErr.Details,
		)
		return
	}

	w.logf(
		"payment request reconcile cycle completed worker_id=%s claimed=%d scanned=%d confirmed=%d detected=%d expired=%d skipped=%d errors=%d latency_ms=%d",
		w.workerID,
		output.Claimed,
		output.Scanned,
		output.Confirmed,
		output.Detected,
		output.Expired,
		output.Skipped,
		output.Errors,
		time.Since(startedAt).Milliseconds(),
	)
}

func (w *Worker) logf(format string, args ...any) {
	if w.logger == nil {
		return
	}
	w.logger.Printf(format, args...)
}
