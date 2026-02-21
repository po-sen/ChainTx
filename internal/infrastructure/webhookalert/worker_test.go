//go:build !integration

package webhookalert

import (
	"bytes"
	"context"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

func TestWorkerDisabled(t *testing.T) {
	fakeOverviewUseCase := &fakeOverviewUseCase{}
	worker := NewWorker(
		false,
		10*time.Millisecond,
		"worker-a",
		fakeOverviewUseCase,
		AlertConfig{Enabled: true, FailedCountThreshold: 1},
		nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	worker.Start(ctx)

	if fakeOverviewUseCase.calls() != 0 {
		t.Fatalf("expected no calls for disabled worker, got %d", fakeOverviewUseCase.calls())
	}
}

func TestWorkerRunsCycleWithAlertLifecycleLogs(t *testing.T) {
	fakeOverviewUseCase := &fakeOverviewUseCase{
		overviews: []dto.WebhookOutboxOverview{
			{FailedCount: 3},
			{FailedCount: 4},
			{FailedCount: 5},
			{FailedCount: 1},
		},
	}
	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", 0)

	worker := NewWorker(
		true,
		10*time.Millisecond,
		"worker-a",
		fakeOverviewUseCase,
		AlertConfig{
			Enabled:              true,
			Cooldown:             60 * time.Second,
			FailedCountThreshold: 2,
		},
		logger,
	)

	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	worker.now = func() time.Time { return now }

	ctx := context.Background()
	worker.runCycle(ctx)
	now = now.Add(30 * time.Second)
	worker.runCycle(ctx)
	now = now.Add(31 * time.Second)
	worker.runCycle(ctx)
	now = now.Add(30 * time.Second)
	worker.runCycle(ctx)

	logs := logBuffer.String()
	if strings.Count(logs, "webhook alert triggered") != 1 {
		t.Fatalf("expected one triggered log, got logs:\n%s", logs)
	}
	if strings.Count(logs, "webhook alert ongoing") != 1 {
		t.Fatalf("expected one ongoing log after cooldown, got logs:\n%s", logs)
	}
	if strings.Count(logs, "webhook alert resolved") != 1 {
		t.Fatalf("expected one resolved log, got logs:\n%s", logs)
	}
}

func TestWorkerOverviewFailureDoesNotStopCycle(t *testing.T) {
	fakeOverviewUseCase := &fakeOverviewUseCase{
		err: apperrors.NewInternal("overview_failed", "overview failed", nil),
	}
	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", 0)

	worker := NewWorker(
		true,
		10*time.Millisecond,
		"worker-a",
		fakeOverviewUseCase,
		AlertConfig{
			Enabled:              true,
			Cooldown:             60 * time.Second,
			FailedCountThreshold: 1,
		},
		logger,
	)

	worker.runCycle(context.Background())
	worker.runCycle(context.Background())

	if fakeOverviewUseCase.calls() != 2 {
		t.Fatalf("expected two overview calls, got %d", fakeOverviewUseCase.calls())
	}
	if strings.Count(logBuffer.String(), "webhook alert evaluation failed") != 2 {
		t.Fatalf("expected failure logs for each cycle, got logs:\n%s", logBuffer.String())
	}
}

type fakeOverviewUseCase struct {
	mu        sync.Mutex
	callCount int
	overviews []dto.WebhookOutboxOverview
	err       *apperrors.AppError
}

func (f *fakeOverviewUseCase) Execute(_ context.Context, _ dto.GetWebhookOutboxOverviewQuery) (dto.WebhookOutboxOverview, *apperrors.AppError) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callCount++
	if f.err != nil {
		return dto.WebhookOutboxOverview{}, f.err
	}
	if len(f.overviews) == 0 {
		return dto.WebhookOutboxOverview{}, nil
	}
	index := f.callCount - 1
	if index >= len(f.overviews) {
		index = len(f.overviews) - 1
	}
	return f.overviews[index], nil
}

func (f *fakeOverviewUseCase) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount
}
