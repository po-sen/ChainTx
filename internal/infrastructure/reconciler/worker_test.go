//go:build !integration

package reconciler

import (
	"context"
	"sync"
	"testing"
	"time"

	"chaintx/internal/application/dto"
	apperrors "chaintx/internal/shared_kernel/errors"
)

func TestWorkerDisabled(t *testing.T) {
	fakeUseCase := &fakeReconcileUseCase{}
	worker := NewWorker(
		false,
		10*time.Millisecond,
		10,
		"worker-a",
		30*time.Second,
		24*time.Hour,
		2,
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

func TestWorkerRunsCycle(t *testing.T) {
	fakeUseCase := &fakeReconcileUseCase{}
	worker := NewWorker(
		true,
		10*time.Millisecond,
		10,
		"worker-a",
		30*time.Second,
		24*time.Hour,
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
	if last.ReorgObserveWindow != 24*time.Hour {
		t.Fatalf("expected reorg observe window 24h, got %s", last.ReorgObserveWindow)
	}
	if last.StabilityCycles != 2 {
		t.Fatalf("expected stability cycles 2, got %d", last.StabilityCycles)
	}
}

type fakeReconcileUseCase struct {
	mu        sync.Mutex
	callCount int
	last      dto.ReconcilePaymentRequestsCommand
}

func (f *fakeReconcileUseCase) Execute(_ context.Context, command dto.ReconcilePaymentRequestsCommand) (dto.ReconcilePaymentRequestsOutput, *apperrors.AppError) {
	f.mu.Lock()
	f.callCount++
	f.last = command
	f.mu.Unlock()
	return dto.ReconcilePaymentRequestsOutput{}, nil
}

func (f *fakeReconcileUseCase) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount
}

func (f *fakeReconcileUseCase) lastCommand() dto.ReconcilePaymentRequestsCommand {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.last
}
