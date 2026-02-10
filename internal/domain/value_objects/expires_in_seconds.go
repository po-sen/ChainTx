package valueobjects

import apperrors "chaintx/internal/shared_kernel/errors"

const (
	MinExpiresInSeconds int64 = 60
	MaxExpiresInSeconds int64 = 2_592_000
)

func ResolveExpiresInSeconds(requested *int64, defaultValue int64) (int64, *apperrors.AppError) {
	if !isExpiryInRange(defaultValue) {
		return 0, apperrors.NewInternal(
			"asset_catalog_invalid",
			"asset catalog default_expires_in_seconds is out of range",
			map[string]any{"default_expires_in_seconds": defaultValue},
		)
	}

	if requested == nil {
		return defaultValue, nil
	}

	if !isExpiryInRange(*requested) {
		return 0, apperrors.NewValidation(
			"invalid_request",
			"expires_in_seconds must be between 60 and 2592000",
			map[string]any{"field": "expires_in_seconds"},
		)
	}

	return *requested, nil
}

func isExpiryInRange(value int64) bool {
	return value >= MinExpiresInSeconds && value <= MaxExpiresInSeconds
}
