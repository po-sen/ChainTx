package dto

import "time"

type InitializePersistenceCommand struct {
	ReadinessTimeout       time.Duration
	ReadinessRetryInterval time.Duration
}
