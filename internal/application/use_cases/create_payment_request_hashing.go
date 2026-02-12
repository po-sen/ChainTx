package use_cases

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	apperrors "chaintx/internal/shared_kernel/errors"
)

type createRequestHashInput struct {
	Chain               string
	Network             string
	Asset               string
	ExpectedAmountMinor *string
	ExpiresInSeconds    int64
	Metadata            map[string]any
}

func hashCreateRequest(input createRequestHashInput) (string, *apperrors.AppError) {
	payload := map[string]any{
		"chain":              input.Chain,
		"network":            input.Network,
		"asset":              input.Asset,
		"expires_in_seconds": input.ExpiresInSeconds,
	}
	if input.ExpectedAmountMinor != nil {
		payload["expected_amount_minor"] = *input.ExpectedAmountMinor
	}
	if len(input.Metadata) > 0 {
		payload["metadata"] = input.Metadata
	}

	canonicalBytes, err := marshalCanonicalJSON(payload)
	if err != nil {
		return "", apperrors.NewInternal(
			"idempotency_hash_payload_invalid",
			"failed to canonicalize request payload",
			map[string]any{"error": err.Error()},
		)
	}

	digest := sha256.Sum256(canonicalBytes)
	return hex.EncodeToString(digest[:]), nil
}

func marshalCanonicalJSON(value any) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := writeCanonicalJSON(buf, value); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func writeCanonicalJSON(buf *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if typed {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case string:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return err
		}
		buf.Write(encoded)
	case json.Number:
		buf.WriteString(typed.String())
	case float64, float32,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return err
		}
		buf.Write(encoded)
	case []any:
		buf.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonicalJSON(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		buf.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				buf.WriteByte(',')
			}

			encodedKey, err := json.Marshal(key)
			if err != nil {
				return err
			}
			buf.Write(encodedKey)
			buf.WriteByte(':')
			if err := writeCanonicalJSON(buf, typed[key]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return err
		}

		var normalized any
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		decoder.UseNumber()
		if decodeErr := decoder.Decode(&normalized); decodeErr != nil {
			return fmt.Errorf("failed to normalize payload for canonical JSON: %w", decodeErr)
		}

		return writeCanonicalJSON(buf, normalized)
	}

	return nil
}

func generateID(prefix string) (string, *apperrors.AppError) {
	randomBytes := make([]byte, 12)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", apperrors.NewInternal(
			"id_generation_failed",
			"failed to generate random identifier",
			map[string]any{"error": err.Error()},
		)
	}

	return prefix + strings.ToLower(hex.EncodeToString(randomBytes)), nil
}
