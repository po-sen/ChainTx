package valueobjects

import (
	"regexp"
	"strings"

	apperrors "chaintx/internal/shared_kernel/errors"

	"golang.org/x/crypto/sha3"
)

var (
	evmAddressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	btcBech32Pattern  = regexp.MustCompile(`^(bc1|tb1|bcrt1)[qpzry9x8gf2tvdw0s3jn54khce6mua7l]{11,87}$`)
	btcBase58Pattern  = regexp.MustCompile(`^[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]{26,35}$`)
)

func NormalizeAddressForStorage(chain, address string) (string, *apperrors.AppError) {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return "", apperrors.NewValidation(
			"invalid_request",
			"address is required",
			map[string]any{"field": "address"},
		)
	}

	switch chain {
	case "ethereum":
		if !evmAddressPattern.MatchString(trimmed) {
			return "", apperrors.NewValidation(
				"invalid_request",
				"ethereum address is invalid",
				map[string]any{"field": "address"},
			)
		}
		return "0x" + strings.ToLower(strings.TrimPrefix(trimmed, "0x")), nil
	case "bitcoin":
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "bc1") || strings.HasPrefix(lower, "tb1") || strings.HasPrefix(lower, "bcrt1") {
			if !btcBech32Pattern.MatchString(lower) {
				return "", apperrors.NewValidation(
					"invalid_request",
					"bitcoin bech32 address is invalid",
					map[string]any{"field": "address"},
				)
			}
			return lower, nil
		}

		if !btcBase58Pattern.MatchString(trimmed) {
			return "", apperrors.NewValidation(
				"invalid_request",
				"bitcoin base58 address is invalid",
				map[string]any{"field": "address"},
			)
		}

		return trimmed, nil
	default:
		return "", apperrors.NewValidation(
			"unsupported_network",
			"unsupported chain for address canonicalization",
			map[string]any{"chain": chain},
		)
	}
}

func FormatAddressForResponse(chain, canonical string) (string, *apperrors.AppError) {
	switch chain {
	case "ethereum":
		return ToEIP55Checksum(canonical)
	case "bitcoin":
		return canonical, nil
	default:
		return "", apperrors.NewInternal(
			"address_chain_unsupported",
			"address chain is not supported for response formatting",
			map[string]any{"chain": chain},
		)
	}
}

func ToEIP55Checksum(canonical string) (string, *apperrors.AppError) {
	normalized := "0x" + strings.ToLower(strings.TrimSpace(strings.TrimPrefix(canonical, "0x")))
	if !evmAddressPattern.MatchString(normalized) {
		return "", apperrors.NewInternal(
			"address_canonical_invalid",
			"canonical ethereum address is invalid",
			map[string]any{"address": canonical},
		)
	}

	hexPart := strings.TrimPrefix(normalized, "0x")
	hash := sha3.NewLegacyKeccak256()
	if _, err := hash.Write([]byte(hexPart)); err != nil {
		return "", apperrors.NewInternal(
			"address_checksum_hash_failed",
			"failed to hash address for checksum",
			map[string]any{"error": err.Error()},
		)
	}
	checksumBytes := hash.Sum(nil)

	out := make([]byte, len(hexPart))
	for i := 0; i < len(hexPart); i++ {
		ch := hexPart[i]
		if ch >= '0' && ch <= '9' {
			out[i] = ch
			continue
		}

		var nibble byte
		if i%2 == 0 {
			nibble = (checksumBytes[i/2] >> 4) & 0x0f
		} else {
			nibble = checksumBytes[i/2] & 0x0f
		}

		if nibble >= 8 {
			out[i] = byte(strings.ToUpper(string(ch))[0])
		} else {
			out[i] = ch
		}
	}

	return "0x" + string(out), nil
}

func NormalizeTokenContract(raw string) (string, *apperrors.AppError) {
	normalized, appErr := NormalizeAddressForStorage("ethereum", raw)
	if appErr != nil {
		return "", apperrors.NewValidation(
			"invalid_request",
			"token_contract is invalid",
			map[string]any{"field": "token_contract"},
		)
	}

	return normalized, nil
}
