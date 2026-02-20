package devtest

import (
	"encoding/json"
	"math/big"
	"strings"

	apperrors "chaintx/internal/shared_kernel/errors"
)

func parseExpectedThreshold(expectedAmountMinor *string) (*big.Int, *apperrors.AppError) {
	if expectedAmountMinor == nil {
		return big.NewInt(1), nil
	}

	trimmed := strings.TrimSpace(*expectedAmountMinor)
	if trimmed == "" {
		return big.NewInt(1), nil
	}

	value := new(big.Int)
	if _, ok := value.SetString(trimmed, 10); !ok || value.Sign() < 0 {
		return nil, apperrors.NewValidation(
			"invalid_request",
			"expected_amount_minor must be a non-negative integer",
			map[string]any{"expected_amount_minor": trimmed},
		)
	}

	return value, nil
}

func parseHexQuantity(raw json.RawMessage) (*big.Int, *apperrors.AppError) {
	var hexValue string
	if err := json.Unmarshal(raw, &hexValue); err != nil {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"failed to parse hex quantity",
			map[string]any{"error": err.Error()},
		)
	}

	trimmed := strings.TrimSpace(hexValue)
	if trimmed == "" {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"hex quantity is empty",
			nil,
		)
	}

	value := new(big.Int)
	if strings.HasPrefix(trimmed, "0x") || strings.HasPrefix(trimmed, "0X") {
		trimmed = trimmed[2:]
	}
	if trimmed == "" {
		trimmed = "0"
	}
	if _, ok := value.SetString(trimmed, 16); !ok {
		return nil, apperrors.NewInternal(
			"chain_observation_failed",
			"invalid hex quantity",
			map[string]any{"value": hexValue},
		)
	}

	return value, nil
}

func buildERC20BalanceOfData(address string) (string, *apperrors.AppError) {
	normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(address)), "0x")
	if len(normalized) != 40 {
		return "", apperrors.NewInternal(
			"chain_observation_failed",
			"invalid address for ERC20 balanceOf call",
			map[string]any{"address": address},
		)
	}

	return "0x70a08231" + strings.Repeat("0", 24) + normalized, nil
}

func normalizeHexAddress(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if strings.HasPrefix(trimmed, "0x") {
		return trimmed
	}
	return "0x" + trimmed
}
