package use_cases

import (
	"bytes"
	"context"
	"encoding/json"
	"path"
	"strings"
	"time"

	"chaintx/internal/application/dto"
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

type normalizedCreatePaymentRequestInput struct {
	Chain               string
	Network             string
	Asset               string
	WebhookURL          string
	ExpectedAmountMinor *string
	ExpiresInSeconds    *int64
	Metadata            map[string]any
	IdempotencyScope    dto.IdempotencyScope
	IdempotencyKey      string
}

func (u *createPaymentRequestUseCase) validateDependencies() *apperrors.AppError {
	if u.assetCatalogReadModel == nil {
		return apperrors.NewInternal(
			"asset_catalog_read_model_missing",
			"asset catalog read model is required",
			nil,
		)
	}

	if u.repository == nil {
		return apperrors.NewInternal(
			"payment_request_repository_missing",
			"payment request repository is required",
			nil,
		)
	}

	if u.walletGateway == nil {
		return apperrors.NewInternal(
			"wallet_allocation_gateway_missing",
			"wallet allocation gateway is required",
			nil,
		)
	}
	if len(u.webhookURLAllowList) == 0 {
		return apperrors.NewInternal(
			"webhook_url_allowlist_missing",
			"webhook url allowlist is required",
			nil,
		)
	}

	return nil
}

func (u *createPaymentRequestUseCase) normalizeCommand(command dto.CreatePaymentRequestCommand) (normalizedCreatePaymentRequestInput, *apperrors.AppError) {
	chain, appErr := valueobjects.NormalizeChain(command.Chain)
	if appErr != nil {
		return normalizedCreatePaymentRequestInput{}, appErr
	}

	network, appErr := valueobjects.NormalizeNetwork(command.Network)
	if appErr != nil {
		return normalizedCreatePaymentRequestInput{}, appErr
	}

	asset, appErr := valueobjects.NormalizeAsset(command.Asset)
	if appErr != nil {
		return normalizedCreatePaymentRequestInput{}, appErr
	}
	webhookURL, webhookHost, appErr := valueobjects.NormalizeWebhookURL(command.WebhookURL)
	if appErr != nil {
		return normalizedCreatePaymentRequestInput{}, appErr
	}
	if !valueobjects.IsWebhookHostAllowed(webhookHost, u.webhookURLAllowList) {
		return normalizedCreatePaymentRequestInput{}, apperrors.NewValidation(
			"webhook_url_not_allowed",
			"webhook_url host is not allowlisted",
			map[string]any{"field": "webhook_url"},
		)
	}

	var expectedAmountMinor *string
	if command.ExpectedAmountMinor != nil {
		normalizedAmount, normalizeErr := valueobjects.NormalizeExpectedAmountMinor(*command.ExpectedAmountMinor)
		if normalizeErr != nil {
			return normalizedCreatePaymentRequestInput{}, normalizeErr
		}
		expectedAmountMinor = &normalizedAmount
	}

	metadata, appErr := normalizeMetadata(command.Metadata)
	if appErr != nil {
		return normalizedCreatePaymentRequestInput{}, appErr
	}

	idempotencyScope := normalizeIdempotencyScope(command.IdempotencyScope)
	idempotencyKey := strings.TrimSpace(command.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey, appErr = generateID("auto_idem_")
		if appErr != nil {
			return normalizedCreatePaymentRequestInput{}, appErr
		}
	}

	return normalizedCreatePaymentRequestInput{
		Chain:               chain,
		Network:             network,
		Asset:               asset,
		WebhookURL:          webhookURL,
		ExpectedAmountMinor: expectedAmountMinor,
		ExpiresInSeconds:    command.ExpiresInSeconds,
		Metadata:            metadata,
		IdempotencyScope:    idempotencyScope,
		IdempotencyKey:      idempotencyKey,
	}, nil
}

func (u *createPaymentRequestUseCase) loadAssetCatalogEntry(ctx context.Context, chain, network, asset string) (dto.AssetCatalogEntry, *apperrors.AppError) {
	assetEntries, appErr := u.assetCatalogReadModel.ListEnabled(ctx)
	if appErr != nil {
		return dto.AssetCatalogEntry{}, appErr
	}

	assetEntry, found := findAssetCatalogEntry(assetEntries, chain, network, asset)
	if !found {
		return dto.AssetCatalogEntry{}, classifyUnsupportedTuple(assetEntries, chain, network, asset)
	}

	return assetEntry, nil
}

func (u *createPaymentRequestUseCase) buildPersistenceCommand(
	input normalizedCreatePaymentRequestInput,
	assetEntry dto.AssetCatalogEntry,
) (dto.CreatePaymentRequestPersistenceCommand, *apperrors.AppError) {
	resolvedExpiresInSeconds, appErr := valueobjects.ResolveExpiresInSeconds(input.ExpiresInSeconds, assetEntry.DefaultExpiresInSeconds)
	if appErr != nil {
		return dto.CreatePaymentRequestPersistenceCommand{}, appErr
	}

	createdAt := u.clock.NowUTC()
	expiresAt := createdAt.Add(time.Duration(resolvedExpiresInSeconds) * time.Second)
	idempotencyExpiresAt := policies.ResolveIdempotencyExpiry(createdAt, expiresAt)

	requestHash, appErr := hashCreateRequest(createRequestHashInput{
		Chain:               input.Chain,
		Network:             input.Network,
		Asset:               input.Asset,
		WebhookURL:          input.WebhookURL,
		ExpectedAmountMinor: input.ExpectedAmountMinor,
		ExpiresInSeconds:    resolvedExpiresInSeconds,
		Metadata:            input.Metadata,
	})
	if appErr != nil {
		return dto.CreatePaymentRequestPersistenceCommand{}, appErr
	}

	resourceID, appErr := generateID("pr_")
	if appErr != nil {
		return dto.CreatePaymentRequestPersistenceCommand{}, appErr
	}

	status := valueobjects.NewPendingPaymentRequestStatus().String()
	return dto.CreatePaymentRequestPersistenceCommand{
		ResourceID:           resourceID,
		IdempotencyScope:     input.IdempotencyScope,
		IdempotencyKey:       input.IdempotencyKey,
		RequestHash:          requestHash,
		HashAlgorithm:        hashAlgorithmSHA256,
		Status:               status,
		Chain:                input.Chain,
		Network:              input.Network,
		Asset:                input.Asset,
		WebhookURL:           input.WebhookURL,
		ExpectedAmountMinor:  input.ExpectedAmountMinor,
		Metadata:             input.Metadata,
		ExpiresAt:            expiresAt,
		IdempotencyExpiresAt: idempotencyExpiresAt,
		CreatedAt:            createdAt,
		AssetCatalogSnapshot: assetEntry,
		AllocationMode:       u.allocationMode,
	}, nil
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
