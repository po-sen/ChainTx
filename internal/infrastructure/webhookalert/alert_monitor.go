package webhookalert

import (
	"time"

	"chaintx/internal/application/dto"
)

const (
	alertSignalFailedCount   = "failed_count"
	alertSignalPendingReady  = "pending_ready_count"
	alertSignalOldestPending = "oldest_pending_age_seconds"
	defaultAlertCooldown     = 300 * time.Second
)

type AlertConfig struct {
	Enabled                  bool
	Cooldown                 time.Duration
	FailedCountThreshold     int64
	PendingReadyThreshold    int64
	OldestPendingAgeSecLimit int64
}

type alertMonitor struct {
	cfg    AlertConfig
	states map[string]alertSignalState
}

type alertSignalState struct {
	active         bool
	triggeredAt    time.Time
	lastNotifiedAt time.Time
}

type alertEvent struct {
	State     string
	Signal    string
	Current   int64
	Threshold int64
	Cooldown  time.Duration
}

func newAlertMonitor(cfg AlertConfig) *alertMonitor {
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = defaultAlertCooldown
	}
	return &alertMonitor{
		cfg:    cfg,
		states: map[string]alertSignalState{},
	}
}

func (m *alertMonitor) enabled() bool {
	return m != nil && m.cfg.Enabled
}

func (m *alertMonitor) evaluate(now time.Time, overview dto.WebhookOutboxOverview) []alertEvent {
	if !m.enabled() {
		return nil
	}

	events := []alertEvent{}
	if m.cfg.FailedCountThreshold > 0 {
		events = append(events, m.evaluateSignal(
			now,
			alertSignalFailedCount,
			overview.FailedCount,
			m.cfg.FailedCountThreshold,
			overview.FailedCount >= m.cfg.FailedCountThreshold,
		)...)
	}
	if m.cfg.PendingReadyThreshold > 0 {
		events = append(events, m.evaluateSignal(
			now,
			alertSignalPendingReady,
			overview.PendingReadyCount,
			m.cfg.PendingReadyThreshold,
			overview.PendingReadyCount >= m.cfg.PendingReadyThreshold,
		)...)
	}
	if m.cfg.OldestPendingAgeSecLimit > 0 {
		oldestAge := int64(0)
		breached := false
		if overview.OldestPendingAgeSec != nil {
			oldestAge = *overview.OldestPendingAgeSec
			breached = oldestAge >= m.cfg.OldestPendingAgeSecLimit
		}
		events = append(events, m.evaluateSignal(
			now,
			alertSignalOldestPending,
			oldestAge,
			m.cfg.OldestPendingAgeSecLimit,
			breached,
		)...)
	}

	return events
}

func (m *alertMonitor) evaluateSignal(
	now time.Time,
	signal string,
	current int64,
	threshold int64,
	breached bool,
) []alertEvent {
	state := m.states[signal]

	if breached {
		if !state.active {
			state.active = true
			state.triggeredAt = now
			state.lastNotifiedAt = now
			m.states[signal] = state
			return []alertEvent{{
				State:     "triggered",
				Signal:    signal,
				Current:   current,
				Threshold: threshold,
				Cooldown:  m.cfg.Cooldown,
			}}
		}

		if now.Sub(state.lastNotifiedAt) < m.cfg.Cooldown {
			return nil
		}
		state.lastNotifiedAt = now
		m.states[signal] = state
		return []alertEvent{{
			State:     "ongoing",
			Signal:    signal,
			Current:   current,
			Threshold: threshold,
			Cooldown:  m.cfg.Cooldown,
		}}
	}

	if state.active {
		delete(m.states, signal)
		return []alertEvent{{
			State:     "resolved",
			Signal:    signal,
			Current:   current,
			Threshold: threshold,
			Cooldown:  m.cfg.Cooldown,
		}}
	}

	return nil
}
