package valueobjects

import (
	"regexp"
	"strings"

	apperrors "chaintx/internal/shared_kernel/errors"
)

var amountMinorPattern = regexp.MustCompile(`^[0-9]{1,78}$`)

func NormalizeExpectedAmountMinor(raw string) (string, *apperrors.AppError) {
	value := strings.TrimSpace(raw)
	if !amountMinorPattern.MatchString(value) {
		return "", apperrors.NewValidation(
			"invalid_request",
			"expected_amount_minor must be an integer string with 1 to 78 digits",
			map[string]any{"field": "expected_amount_minor"},
		)
	}

	return value, nil
}
