package policies

import "time"

const (
	idempotencyTTLBaseline = 7 * 24 * time.Hour
	idempotencyTTLMinimum  = 24 * time.Hour
)

func ResolveIdempotencyExpiry(createdAt, requestExpiresAt time.Time) time.Time {
	resolved := createdAt.Add(idempotencyTTLBaseline)
	if requestExpiresAt.After(resolved) {
		resolved = requestExpiresAt
	}

	minimum := createdAt.Add(idempotencyTTLMinimum)
	if minimum.After(resolved) {
		resolved = minimum
	}

	return resolved
}
