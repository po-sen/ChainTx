package valueobjects

import (
	"regexp"
	"strings"

	apperrors "chaintx/internal/shared_kernel/errors"
)

var (
	chainPattern   = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)
	networkPattern = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)
	assetPattern   = regexp.MustCompile(`^[A-Z0-9_-]{1,32}$`)
)

func NormalizeChain(raw string) (string, *apperrors.AppError) {
	chain := strings.ToLower(strings.TrimSpace(raw))
	if chain == "" || !chainPattern.MatchString(chain) {
		return "", apperrors.NewValidation(
			"invalid_request",
			"chain is invalid",
			map[string]any{"field": "chain"},
		)
	}

	return chain, nil
}

func NormalizeNetwork(raw string) (string, *apperrors.AppError) {
	network := strings.ToLower(strings.TrimSpace(raw))
	if network == "" || !networkPattern.MatchString(network) {
		return "", apperrors.NewValidation(
			"invalid_request",
			"network is invalid",
			map[string]any{"field": "network"},
		)
	}

	return network, nil
}

func NormalizeAsset(raw string) (string, *apperrors.AppError) {
	asset := strings.ToUpper(strings.TrimSpace(raw))
	if asset == "" || !assetPattern.MatchString(asset) {
		return "", apperrors.NewValidation(
			"invalid_request",
			"asset is invalid",
			map[string]any{"field": "asset"},
		)
	}

	return asset, nil
}
