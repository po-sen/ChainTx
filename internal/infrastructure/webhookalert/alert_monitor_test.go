//go:build !integration

package webhookalert

import (
	"testing"
	"time"

	"chaintx/internal/application/dto"
)

func TestAlertMonitorLifecycle(t *testing.T) {
	monitor := newAlertMonitor(AlertConfig{
		Enabled:              true,
		Cooldown:             60 * time.Second,
		FailedCountThreshold: 2,
	})

	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	events := monitor.evaluate(now, dto.WebhookOutboxOverview{FailedCount: 3})
	assertSingleAlertEvent(t, events, "triggered", alertSignalFailedCount)

	events = monitor.evaluate(now.Add(30*time.Second), dto.WebhookOutboxOverview{FailedCount: 4})
	if len(events) != 0 {
		t.Fatalf("expected no event before cooldown, got %+v", events)
	}

	events = monitor.evaluate(now.Add(61*time.Second), dto.WebhookOutboxOverview{FailedCount: 4})
	assertSingleAlertEvent(t, events, "ongoing", alertSignalFailedCount)

	events = monitor.evaluate(now.Add(90*time.Second), dto.WebhookOutboxOverview{FailedCount: 1})
	assertSingleAlertEvent(t, events, "resolved", alertSignalFailedCount)
}

func TestAlertMonitorOldestPendingAgeNilDoesNotTrigger(t *testing.T) {
	monitor := newAlertMonitor(AlertConfig{
		Enabled:                  true,
		Cooldown:                 60 * time.Second,
		OldestPendingAgeSecLimit: 10,
	})

	events := monitor.evaluate(time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC), dto.WebhookOutboxOverview{})
	if len(events) != 0 {
		t.Fatalf("expected no event when oldest pending age is nil, got %+v", events)
	}
}

func assertSingleAlertEvent(t *testing.T, events []alertEvent, expectedState string, expectedSignal string) {
	t.Helper()
	if len(events) != 1 {
		t.Fatalf("expected exactly one event, got %+v", events)
	}
	if events[0].State != expectedState {
		t.Fatalf("expected state %s, got %s", expectedState, events[0].State)
	}
	if events[0].Signal != expectedSignal {
		t.Fatalf("expected signal %s, got %s", expectedSignal, events[0].Signal)
	}
}
