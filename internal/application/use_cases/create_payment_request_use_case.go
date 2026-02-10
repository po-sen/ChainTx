package use_cases

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"chaintx/internal/application/dto"
	portsin "chaintx/internal/application/ports/in"
	portsout "chaintx/internal/application/ports/out"
	"chaintx/internal/domain/policies"
	valueobjects "chaintx/internal/domain/value_objects"
	apperrors "chaintx/internal/shared_kernel/errors"
)

const (
	defaultPrincipalID = "anonymous"
	defaultHTTPMethod  = "POST"
	defaultHTTPPath    = "/v1/payment-requests"

	maxMetadataBytes = 4096

	hashAlgorithmSHA256 = "sha256"
)

type createPaymentRequestUseCase struct {
	assetCatalogReadModel portsout.AssetCatalogReadModel
	repository            portsout.PaymentRequestRepository
	clock                 Clock
}

func NewCreatePaymentRequestUseCase(
	assetCatalogReadModel portsout.AssetCatalogReadModel,
	repository portsout.PaymentRequestRepository,
	clock Clock,
) portsin.CreatePaymentRequestUseCase {
	if clock == nil {
		clock = NewSystemClock()
	}

	return &createPaymentRequestUseCase{
		assetCatalogReadModel: assetCatalogReadModel,
		repository:            repository,
		clock:                 clock,
	}
}

func (u *createPaymentRequestUseCase) Execute(ctx context.Context, command dto.CreatePaymentRequestCommand) (dto.CreatePaymentRequestOutput, *apperrors.AppError) {
	if u.assetCatalogReadModel == nil {
		return dto.CreatePaymentRequestOutput{}, apperrors.NewInternal(
			"asset_catalog_read_model_missing",
			"asset catalog read model is required",
			nil,
		)
	}
	if u.repository == nil {
		return dto.CreatePaymentRequestOutput{}, apperrors.NewInternal(
			"payment_request_repository_missing",
			"payment request repository is required",
			nil,
		)
	}

	chain, appErr := valueobjects.NormalizeChain(command.Chain)
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}
	network, appErr := valueobjects.NormalizeNetwork(command.Network)
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}
	asset, appErr := valueobjects.NormalizeAsset(command.Asset)
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}

	var expectedAmountMinor *string
	if command.ExpectedAmountMinor != nil {
		normalizedAmount, normalizeErr := valueobjects.NormalizeExpectedAmountMinor(*command.ExpectedAmountMinor)
		if normalizeErr != nil {
			return dto.CreatePaymentRequestOutput{}, normalizeErr
		}
		expectedAmountMinor = &normalizedAmount
	}

	metadata, appErr := normalizeMetadata(command.Metadata)
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}

	assetEntries, appErr := u.assetCatalogReadModel.ListEnabled(ctx)
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}

	assetEntry, found := findAssetCatalogEntry(assetEntries, chain, network, asset)
	if !found {
		return dto.CreatePaymentRequestOutput{}, classifyUnsupportedTuple(assetEntries, chain, network, asset)
	}

	resolvedExpiresInSeconds, appErr := valueobjects.ResolveExpiresInSeconds(command.ExpiresInSeconds, assetEntry.DefaultExpiresInSeconds)
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}

	idempotencyScope := normalizeIdempotencyScope(command.IdempotencyScope)
	idempotencyKey := strings.TrimSpace(command.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey, appErr = generateID("auto_idem_")
		if appErr != nil {
			return dto.CreatePaymentRequestOutput{}, appErr
		}
	}

	createdAt := u.clock.NowUTC()
	expiresAt := createdAt.Add(time.Duration(resolvedExpiresInSeconds) * time.Second)
	idempotencyExpiresAt := policies.ResolveIdempotencyExpiry(createdAt, expiresAt)

	requestHash, appErr := hashCreateRequest(createRequestHashInput{
		Chain:               chain,
		Network:             network,
		Asset:               asset,
		ExpectedAmountMinor: expectedAmountMinor,
		ExpiresInSeconds:    resolvedExpiresInSeconds,
		Metadata:            metadata,
	})
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}

	resourceID, appErr := generateID("pr_")
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}
	status := valueobjects.NewPendingPaymentRequestStatus().String()

	result, appErr := u.repository.Create(ctx, dto.CreatePaymentRequestPersistenceCommand{
		ResourceID:           resourceID,
		IdempotencyScope:     idempotencyScope,
		IdempotencyKey:       idempotencyKey,
		RequestHash:          requestHash,
		HashAlgorithm:        hashAlgorithmSHA256,
		Status:               status,
		Chain:                chain,
		Network:              network,
		Asset:                asset,
		ExpectedAmountMinor:  expectedAmountMinor,
		Metadata:             metadata,
		ExpiresAt:            expiresAt,
		IdempotencyExpiresAt: idempotencyExpiresAt,
		CreatedAt:            createdAt,
		AssetCatalogSnapshot: assetEntry,
	})
	if appErr != nil {
		return dto.CreatePaymentRequestOutput{}, appErr
	}

	return dto.CreatePaymentRequestOutput(result), nil
}

func findAssetCatalogEntry(entries []dto.AssetCatalogEntry, chain, network, asset string) (dto.AssetCatalogEntry, bool) {
	for _, entry := range entries {
		if entry.Chain == chain && entry.Network == network && strings.EqualFold(entry.Asset, asset) {
			return entry, true
		}
	}

	return dto.AssetCatalogEntry{}, false
}

func classifyUnsupportedTuple(entries []dto.AssetCatalogEntry, chain, network, asset string) *apperrors.AppError {
	networkSupported := false
	for _, entry := range entries {
		if entry.Chain == chain && entry.Network == network {
			networkSupported = true
			if strings.EqualFold(entry.Asset, asset) {
				break
			}
		}
	}

	if !networkSupported {
		return apperrors.NewValidation(
			"unsupported_network",
			"network is not supported for the selected chain",
			map[string]any{
				"chain":   chain,
				"network": network,
			},
		)
	}

	return apperrors.NewValidation(
		"unsupported_asset",
		"asset is not supported for the selected chain and network",
		map[string]any{
			"chain":   chain,
			"network": network,
			"asset":   asset,
		},
	)
}

func normalizeIdempotencyScope(scope dto.IdempotencyScope) dto.IdempotencyScope {
	principalID := strings.TrimSpace(scope.PrincipalID)
	if principalID == "" {
		principalID = defaultPrincipalID
	}

	httpMethod := strings.ToUpper(strings.TrimSpace(scope.HTTPMethod))
	if httpMethod == "" {
		httpMethod = defaultHTTPMethod
	}

	httpPath := strings.TrimSpace(scope.HTTPPath)
	if httpPath == "" {
		httpPath = defaultHTTPPath
	}
	httpPath = normalizePath(httpPath)

	return dto.IdempotencyScope{
		PrincipalID: principalID,
		HTTPMethod:  httpMethod,
		HTTPPath:    httpPath,
	}
}

func normalizePath(raw string) string {
	normalized := path.Clean("/" + strings.TrimLeft(strings.TrimSpace(raw), "/"))
	if normalized == "." {
		return defaultHTTPPath
	}

	return normalized
}

func normalizeMetadata(input map[string]any) (map[string]any, *apperrors.AppError) {
	if len(input) == 0 {
		return map[string]any{}, nil
	}

	encoded, err := json.Marshal(input)
	if err != nil {
		return nil, apperrors.NewValidation(
			"invalid_request",
			"metadata must be valid JSON object",
			map[string]any{"field": "metadata"},
		)
	}

	if len(encoded) > maxMetadataBytes {
		return nil, apperrors.NewValidation(
			"invalid_request",
			"metadata payload exceeds 4KB",
			map[string]any{"field": "metadata", "max_bytes": maxMetadataBytes},
		)
	}

	var normalized map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if decodeErr := decoder.Decode(&normalized); decodeErr != nil {
		return nil, apperrors.NewValidation(
			"invalid_request",
			"metadata must be valid JSON object",
			map[string]any{"field": "metadata"},
		)
	}

	if normalized == nil {
		normalized = map[string]any{}
	}

	return normalized, nil
}

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
