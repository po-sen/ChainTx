//go:build !integration

package webhook

import (
	"context"
	"sync"
	"testing"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

func TestWorkerDisabled(t *testing.T) {
	fakeUseCase := &fakeDispatchUseCase{}
	worker := NewWorker(
		false,
		10*time.Millisecond,
		10,
		"worker-a",
		30*time.Second,
		5*time.Second,
		60*time.Second,
		2000,
		3,
		fakeUseCase,
		nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	worker.Start(ctx)

	if fakeUseCase.calls() != 0 {
		t.Fatalf("expected no calls for disabled worker, got %d", fakeUseCase.calls())
	}
}

func TestWorkerRunsCycleWithRetryConfig(t *testing.T) {
	fakeUseCase := &fakeDispatchUseCase{}
	worker := NewWorker(
		true,
		10*time.Millisecond,
		10,
		"worker-a",
		30*time.Second,
		5*time.Second,
		60*time.Second,
		1500,
		2,
		fakeUseCase,
		nil,
	)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	worker.Start(ctx)

	if fakeUseCase.calls() == 0 {
		t.Fatalf("expected at least one cycle call")
	}
	last := fakeUseCase.lastCommand()
	if last.WorkerID != "worker-a" {
		t.Fatalf("expected worker id worker-a, got %s", last.WorkerID)
	}
	if last.LeaseDuration != 30*time.Second {
		t.Fatalf("expected lease duration 30s, got %s", last.LeaseDuration)
	}
	if last.RetryJitterBPS != 1500 {
		t.Fatalf("expected retry jitter bps 1500, got %d", last.RetryJitterBPS)
	}
	if last.RetryBudget != 2 {
		t.Fatalf("expected retry budget 2, got %d", last.RetryBudget)
	}
}

type fakeDispatchUseCase struct {
	mu        sync.Mutex
	callCount int
	last      dto.DispatchWebhookEventsCommand
}

func (f *fakeDispatchUseCase) Execute(_ context.Context, command dto.DispatchWebhookEventsCommand) (dto.DispatchWebhookEventsOutput, *apperrors.AppError) {
	f.mu.Lock()
	f.callCount++
	f.last = command
	f.mu.Unlock()
	return dto.DispatchWebhookEventsOutput{}, nil
}

func (f *fakeDispatchUseCase) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount
}

func (f *fakeDispatchUseCase) lastCommand() dto.DispatchWebhookEventsCommand {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.last
}
