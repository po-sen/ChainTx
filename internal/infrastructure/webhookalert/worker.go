package webhookalert

import (
	"context"
	"log"
	"time"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
)

const defaultPollInterval = 10 * time.Second

type Worker struct {
	enabled         bool
	pollInterval    time.Duration
	workerID        string
	overviewUseCase portsin.GetWebhookOutboxOverviewUseCase
	alertMonitor    *alertMonitor
	now             func() time.Time
	logger          *log.Logger
}

func NewWorker(
	enabled bool,
	pollInterval time.Duration,
	workerID string,
	overviewUseCase portsin.GetWebhookOutboxOverviewUseCase,
	alertConfig AlertConfig,
	logger *log.Logger,
) *Worker {
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	return &Worker{
		enabled:         enabled,
		pollInterval:    pollInterval,
		workerID:        workerID,
		overviewUseCase: overviewUseCase,
		alertMonitor:    newAlertMonitor(alertConfig),
		now:             time.Now,
		logger:          logger,
	}
}

func (w *Worker) Enabled() bool {
	return w != nil && w.enabled
}

func (w *Worker) Start(ctx context.Context) {
	if w == nil || !w.enabled || w.overviewUseCase == nil || w.alertMonitor == nil || !w.alertMonitor.enabled() {
		return
	}

	w.logf(
		"webhook alert worker started worker_id=%s poll_interval=%s",
		w.workerID,
		w.pollInterval,
	)

	w.runCycle(ctx)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logf("webhook alert worker stopped")
			return
		case <-ticker.C:
			w.runCycle(ctx)
		}
	}
}

func (w *Worker) runCycle(ctx context.Context) {
	now := w.nowUTC()
	overview, appErr := w.overviewUseCase.Execute(ctx, dto.GetWebhookOutboxOverviewQuery{Now: now})
	if appErr != nil {
		w.logf(
			"webhook alert evaluation failed worker_id=%s code=%s message=%s details=%v",
			w.workerID,
			appErr.Code,
			appErr.Message,
			appErr.Details,
		)
		return
	}

	for _, event := range w.alertMonitor.evaluate(now, overview) {
		w.logf(
			"webhook alert %s worker_id=%s signal=%s current=%d threshold=%d cooldown=%s",
			event.State,
			w.workerID,
			event.Signal,
			event.Current,
			event.Threshold,
			event.Cooldown,
		)
	}
}

func (w *Worker) nowUTC() time.Time {
	if w == nil || w.now == nil {
		return time.Now().UTC()
	}
	return w.now().UTC()
}

func (w *Worker) logf(format string, args ...any) {
	if w.logger == nil {
		return
	}
	w.logger.Printf(format, args...)
}
