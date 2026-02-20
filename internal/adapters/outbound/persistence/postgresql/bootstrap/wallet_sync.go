package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"chaintx/internal/infrastructure/walletkeys"
	apperrors "chaintx/internal/shared_kernel/errors"
)

const (
	derivationPathTemplate = "0/{index}"

	syncActionReused      = "reused"
	syncActionReactivated = "reactivated"
	syncActionRotated     = "rotated"

	matchSourceActive   = "active"
	matchSourceLegacy   = "legacy"
	matchSourceUnhashed = "unhashed"
)

type catalogSyncTarget struct {
	Chain    string
	Network  string
	KeysetID string
}

func (t catalogSyncTarget) key() string {
	return t.Chain + "|" + t.Network + "|" + t.KeysetID
}

type hashCandidate struct {
	Hash   string
	Source string
}

type walletAccountRef struct {
	ID   string
	Hash string
}

type walletSyncOutcome struct {
	Action          string
	MatchSource     string
	WalletAccountID string
	KeyHash         string
}

func (g *Gateway) syncDevtestWalletAllocationState(ctx context.Context) *apperrors.AppError {
	keysetHashAlgo := strings.ToLower(strings.TrimSpace(g.validationRules.KeysetHashAlgorithm))
	if keysetHashAlgo == "" {
		keysetHashAlgo = "hmac-sha256"
	}
	if keysetHashAlgo != "hmac-sha256" {
		return apperrors.NewInternal(
			"invalid_configuration",
			"unsupported keyset hash algorithm",
			map[string]any{"key_material_hash_algo": keysetHashAlgo},
		)
	}

	activeSecret := strings.TrimSpace(g.validationRules.KeysetHashHMACSecret)
	if activeSecret == "" {
		return apperrors.NewInternal(
			"invalid_configuration",
			"missing keyset hash hmac secret for startup wallet sync",
			nil,
		)
	}

	preflightByTarget := map[string]DevtestKeysetPreflightEntry{}
	for _, entry := range g.validationRules.DevtestKeysetPreflight {
		normalizedEntry := DevtestKeysetPreflightEntry{
			Chain:                 strings.ToLower(strings.TrimSpace(entry.Chain)),
			Network:               strings.ToLower(strings.TrimSpace(entry.Network)),
			KeysetID:              strings.TrimSpace(entry.KeysetID),
			ExtendedPublicKey:     strings.TrimSpace(entry.ExtendedPublicKey),
			ExpectedIndex0Address: strings.TrimSpace(entry.ExpectedIndex0Address),
		}
		if normalizedEntry.Chain == "" || normalizedEntry.Network == "" || normalizedEntry.KeysetID == "" || normalizedEntry.ExtendedPublicKey == "" || normalizedEntry.ExpectedIndex0Address == "" {
			continue
		}
		preflightByTarget[normalizedEntry.Chain+"|"+normalizedEntry.Network+"|"+normalizedEntry.KeysetID] = normalizedEntry
	}
	if len(preflightByTarget) == 0 {
		return apperrors.NewInternal(
			"invalid_configuration",
			"devtest startup preflight requires nested keyset entries with expected_index0_address",
			nil,
		)
	}

	db, err := sql.Open("pgx", g.databaseURL)
	if err != nil {
		return apperrors.NewInternal(
			"DB_CONNECT_INIT_FAILED",
			"failed to initialize database connection",
			map[string]any{"database_target": g.databaseTarget},
		)
	}
	defer db.Close()

	targets, appErr := g.loadEnabledCatalogSyncTargets(ctx, db)
	if appErr != nil {
		return appErr
	}
	if len(targets) == 0 {
		g.logf("wallet-account startup sync skipped: no enabled catalog targets")
		return nil
	}

	for _, target := range targets {
		keyMaterial, exists := g.validationRules.DevtestKeysets[target.KeysetID]
		if !exists || strings.TrimSpace(keyMaterial) == "" {
			return apperrors.NewInternal(
				"invalid_configuration",
				"devtest keyset is missing for startup wallet sync",
				map[string]any{
					"chain":     target.Chain,
					"network":   target.Network,
					"keyset_id": target.KeysetID,
				},
			)
		}

		preflightEntry, exists := preflightByTarget[target.key()]
		if !exists {
			return apperrors.NewInternal(
				"invalid_configuration",
				"missing preflight entry for enabled wallet target",
				map[string]any{
					"chain":     target.Chain,
					"network":   target.Network,
					"keyset_id": target.KeysetID,
				},
			)
		}
		if preflightEntry.ExtendedPublicKey != strings.TrimSpace(keyMaterial) {
			return apperrors.NewInternal(
				"invalid_configuration",
				"preflight key material does not match keyset_id value",
				map[string]any{
					"chain":     target.Chain,
					"network":   target.Network,
					"keyset_id": target.KeysetID,
				},
			)
		}

		if appErr := g.verifyIndexZeroPreflight(target, keyMaterial, preflightEntry.ExpectedIndex0Address); appErr != nil {
			return appErr
		}

		outcome, appErr := g.syncWalletAccountTarget(ctx, db, target, keyMaterial, keysetHashAlgo, activeSecret)
		if appErr != nil {
			return appErr
		}

		g.logf(
			"wallet-account startup sync action=%s chain=%s network=%s keyset_id=%s wallet_account_id=%s hash_prefix=%s match_source=%s",
			outcome.Action,
			target.Chain,
			target.Network,
			target.KeysetID,
			outcome.WalletAccountID,
			hashPrefix(outcome.KeyHash),
			outcome.MatchSource,
		)
	}

	g.logf("wallet-account startup sync completed targets=%d", len(targets))
	return nil
}

func (g *Gateway) loadEnabledCatalogSyncTargets(ctx context.Context, db *sql.DB) ([]catalogSyncTarget, *apperrors.AppError) {
	const query = `
SELECT DISTINCT ac.chain, ac.network, wa.keyset_id
FROM app.asset_catalog ac
JOIN app.wallet_accounts wa ON wa.id = ac.wallet_account_id
WHERE ac.enabled = TRUE
ORDER BY ac.chain, ac.network, wa.keyset_id
`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, apperrors.NewInternal(
			"invalid_configuration",
			"failed to query enabled wallet sync targets",
			map[string]any{"error": err.Error()},
		)
	}
	defer rows.Close()

	targets := []catalogSyncTarget{}
	for rows.Next() {
		var chain string
		var network string
		var keysetID string
		if err := rows.Scan(&chain, &network, &keysetID); err != nil {
			return nil, apperrors.NewInternal(
				"invalid_configuration",
				"failed to parse wallet sync target row",
				map[string]any{"error": err.Error()},
			)
		}

		target := catalogSyncTarget{
			Chain:    strings.ToLower(strings.TrimSpace(chain)),
			Network:  strings.ToLower(strings.TrimSpace(network)),
			KeysetID: strings.TrimSpace(keysetID),
		}
		if target.Chain == "" || target.Network == "" || target.KeysetID == "" {
			return nil, apperrors.NewInternal(
				"invalid_configuration",
				"wallet sync target row is missing chain/network/keyset_id",
				map[string]any{
					"chain":     target.Chain,
					"network":   target.Network,
					"keyset_id": target.KeysetID,
				},
			)
		}

		targets = append(targets, target)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternal(
			"invalid_configuration",
			"failed while iterating wallet sync target rows",
			map[string]any{"error": err.Error()},
		)
	}

	return targets, nil
}

func (g *Gateway) verifyIndexZeroPreflight(target catalogSyncTarget, rawKey string, expectedAddress string) *apperrors.AppError {
	normalizer, exists := g.validationRules.DevtestKeyNormalizers[target.Chain]
	if !exists {
		return apperrors.NewInternal(
			"invalid_configuration",
			"unsupported chain for keyset preflight",
			map[string]any{
				"chain":     target.Chain,
				"network":   target.Network,
				"keyset_id": target.KeysetID,
			},
		)
	}

	key, _, keyErr := normalizer(rawKey)
	if keyErr != nil {
		return mapWalletKeyError(keyErr, map[string]any{
			"chain":     target.Chain,
			"network":   target.Network,
			"keyset_id": target.KeysetID,
		})
	}
	if keyErr := walletkeys.ValidateAccountLevelPolicy(key); keyErr != nil {
		return mapWalletKeyError(keyErr, map[string]any{
			"chain":     target.Chain,
			"network":   target.Network,
			"keyset_id": target.KeysetID,
		})
	}

	var derivedAddress string
	switch target.Chain {
	case "bitcoin":
		derived, deriveErr := walletkeys.DeriveBitcoinP2WPKHAddress(key, target.Network, derivationPathTemplate, 0)
		if deriveErr != nil {
			return mapWalletKeyError(deriveErr, map[string]any{
				"chain":     target.Chain,
				"network":   target.Network,
				"keyset_id": target.KeysetID,
			})
		}
		derivedAddress = derived
	case "ethereum":
		switch target.Network {
		case "sepolia", "local", "mainnet":
		default:
			return apperrors.NewInternal(
				"invalid_configuration",
				"unsupported ethereum network for keyset preflight",
				map[string]any{
					"network":   target.Network,
					"keyset_id": target.KeysetID,
				},
			)
		}
		derived, deriveErr := walletkeys.DeriveEVMAddress(key, derivationPathTemplate, 0)
		if deriveErr != nil {
			return mapWalletKeyError(deriveErr, map[string]any{
				"chain":     target.Chain,
				"network":   target.Network,
				"keyset_id": target.KeysetID,
			})
		}
		derivedAddress = derived
	default:
		return apperrors.NewInternal(
			"invalid_configuration",
			"unsupported chain for keyset preflight",
			map[string]any{
				"chain":     target.Chain,
				"network":   target.Network,
				"keyset_id": target.KeysetID,
			},
		)
	}

	normalizedExpected := strings.ToLower(strings.TrimSpace(expectedAddress))
	normalizedDerived := strings.ToLower(strings.TrimSpace(derivedAddress))
	if normalizedExpected != normalizedDerived {
		return apperrors.NewInternal(
			"invalid_configuration",
			"devtest keyset preflight mismatch",
			map[string]any{
				"chain":            target.Chain,
				"network":          target.Network,
				"keyset_id":        target.KeysetID,
				"expected_address": normalizedExpected,
				"derived_address":  normalizedDerived,
			},
		)
	}

	g.logf(
		"startup keyset preflight passed chain=%s network=%s keyset_id=%s derived_index=0 address=%s",
		target.Chain,
		target.Network,
		target.KeysetID,
		normalizedDerived,
	)
	return nil
}

func (g *Gateway) syncWalletAccountTarget(
	ctx context.Context,
	db *sql.DB,
	target catalogSyncTarget,
	keyMaterial string,
	keysetHashAlgo string,
	activeSecret string,
) (walletSyncOutcome, *apperrors.AppError) {
	activeHash := computeHMACSHA256(activeSecret, keyMaterial)
	legacyCandidates := make([]hashCandidate, 0, len(g.validationRules.KeysetHashHMACLegacy))
	legacyHashSet := map[string]struct{}{}
	seenHashes := map[string]struct{}{activeHash: {}}
	for _, secret := range g.validationRules.KeysetHashHMACLegacy {
		legacyHash := computeHMACSHA256(secret, keyMaterial)
		if _, exists := seenHashes[legacyHash]; exists {
			continue
		}
		seenHashes[legacyHash] = struct{}{}
		legacyHashSet[legacyHash] = struct{}{}
		legacyCandidates = append(legacyCandidates, hashCandidate{Hash: legacyHash, Source: matchSourceLegacy})
	}
	allCandidates := append([]hashCandidate{{Hash: activeHash, Source: matchSourceActive}}, legacyCandidates...)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return walletSyncOutcome{}, apperrors.NewInternal(
			"wallet_account_sync_failed",
			"failed to begin wallet-account sync transaction",
			map[string]any{"error": err.Error()},
		)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = tx.Rollback()
	}()

	activeRef, appErr := getActiveWalletAccountForUpdate(ctx, tx, target)
	if appErr != nil {
		return walletSyncOutcome{}, appErr
	}

	selectedRef := walletAccountRef{}
	action := ""
	matchSource := ""
	details := map[string]any{}

	if activeRef.ID != "" {
		details["previous_active_wallet_account_id"] = activeRef.ID
	}

	if source, matches := classifyHashMatch(activeRef.Hash, activeHash, legacyHashSet); activeRef.ID != "" && matches {
		if appErr := upsertWalletAccountHash(ctx, tx, activeRef.ID, activeHash, keysetHashAlgo, true); appErr != nil {
			return walletSyncOutcome{}, appErr
		}
		if appErr := deactivateOtherActiveWallets(ctx, tx, target, activeRef.ID); appErr != nil {
			return walletSyncOutcome{}, appErr
		}
		selectedRef = activeRef
		action = syncActionReused
		matchSource = source
		details["selected_from"] = "active"
	} else {
		historicalRef, historicalSource, found, appErr := findHistoricalWalletAccountForUpdate(ctx, tx, target, allCandidates)
		if appErr != nil {
			return walletSyncOutcome{}, appErr
		}
		if found {
			if appErr := deactivateAllActiveWallets(ctx, tx, target); appErr != nil {
				return walletSyncOutcome{}, appErr
			}
			if appErr := upsertWalletAccountHash(ctx, tx, historicalRef.ID, activeHash, keysetHashAlgo, true); appErr != nil {
				return walletSyncOutcome{}, appErr
			}
			selectedRef = historicalRef
			action = syncActionReactivated
			matchSource = historicalSource
			details["selected_from"] = "historical"
		} else {
			if appErr := deactivateAllActiveWallets(ctx, tx, target); appErr != nil {
				return walletSyncOutcome{}, appErr
			}

			newWalletAccountID := generateWalletAccountID(target.Chain, target.Network, activeHash)
			if appErr := insertWalletAccount(ctx, tx, newWalletAccountID, target, activeHash, keysetHashAlgo); appErr != nil {
				return walletSyncOutcome{}, appErr
			}
			selectedRef = walletAccountRef{ID: newWalletAccountID, Hash: activeHash}
			action = syncActionRotated
			matchSource = matchSourceActive
			details["selected_from"] = "new"
		}
	}

	if appErr := bindAssetCatalogWalletAccount(ctx, tx, target, selectedRef.ID); appErr != nil {
		return walletSyncOutcome{}, appErr
	}

	details["hash_prefix"] = hashPrefix(activeHash)
	if appErr := insertWalletSyncEvent(ctx, tx, target, selectedRef.ID, action, matchSource, activeHash, keysetHashAlgo, details); appErr != nil {
		return walletSyncOutcome{}, appErr
	}

	if err := tx.Commit(); err != nil {
		return walletSyncOutcome{}, apperrors.NewInternal(
			"wallet_account_sync_failed",
			"failed to commit wallet-account sync transaction",
			map[string]any{"error": err.Error()},
		)
	}
	committed = true

	return walletSyncOutcome{
		Action:          action,
		MatchSource:     matchSource,
		WalletAccountID: selectedRef.ID,
		KeyHash:         activeHash,
	}, nil
}

func getActiveWalletAccountForUpdate(ctx context.Context, tx *sql.Tx, target catalogSyncTarget) (walletAccountRef, *apperrors.AppError) {
	const query = `
SELECT id, COALESCE(key_material_hash, '')
FROM app.wallet_accounts
WHERE chain = $1
  AND network = $2
  AND keyset_id = $3
  AND is_active = TRUE
ORDER BY updated_at DESC, created_at DESC
LIMIT 1
FOR UPDATE
`

	row := tx.QueryRowContext(ctx, query, target.Chain, target.Network, target.KeysetID)
	ref := walletAccountRef{}
	err := row.Scan(&ref.ID, &ref.Hash)
	if err == sql.ErrNoRows {
		return walletAccountRef{}, nil
	}
	if err != nil {
		return walletAccountRef{}, apperrors.NewInternal(
			"wallet_account_sync_failed",
			"failed to query active wallet account",
			map[string]any{"error": err.Error()},
		)
	}
	ref.ID = strings.TrimSpace(ref.ID)
	ref.Hash = strings.ToLower(strings.TrimSpace(ref.Hash))
	return ref, nil
}

func findHistoricalWalletAccountForUpdate(
	ctx context.Context,
	tx *sql.Tx,
	target catalogSyncTarget,
	candidates []hashCandidate,
) (walletAccountRef, string, bool, *apperrors.AppError) {
	const query = `
SELECT id, COALESCE(key_material_hash, '')
FROM app.wallet_accounts
WHERE chain = $1
  AND network = $2
  AND keyset_id = $3
  AND is_active = FALSE
  AND key_material_hash = $4
ORDER BY updated_at DESC, created_at DESC
LIMIT 1
FOR UPDATE
`

	for _, candidate := range candidates {
		if candidate.Hash == "" {
			continue
		}
		ref := walletAccountRef{}
		err := tx.QueryRowContext(ctx, query, target.Chain, target.Network, target.KeysetID, candidate.Hash).Scan(&ref.ID, &ref.Hash)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return walletAccountRef{}, "", false, apperrors.NewInternal(
				"wallet_account_sync_failed",
				"failed to query historical wallet account",
				map[string]any{"error": err.Error()},
			)
		}
		ref.ID = strings.TrimSpace(ref.ID)
		ref.Hash = strings.ToLower(strings.TrimSpace(ref.Hash))
		return ref, candidate.Source, true, nil
	}

	return walletAccountRef{}, "", false, nil
}

func deactivateAllActiveWallets(ctx context.Context, tx *sql.Tx, target catalogSyncTarget) *apperrors.AppError {
	const query = `
UPDATE app.wallet_accounts
SET is_active = FALSE,
    updated_at = now()
WHERE chain = $1
  AND network = $2
  AND keyset_id = $3
  AND is_active = TRUE
`
	if _, err := tx.ExecContext(ctx, query, target.Chain, target.Network, target.KeysetID); err != nil {
		return apperrors.NewInternal(
			"wallet_account_sync_failed",
			"failed to deactivate active wallet accounts",
			map[string]any{"error": err.Error()},
		)
	}
	return nil
}

func deactivateOtherActiveWallets(ctx context.Context, tx *sql.Tx, target catalogSyncTarget, keepID string) *apperrors.AppError {
	const query = `
UPDATE app.wallet_accounts
SET is_active = FALSE,
    updated_at = now()
WHERE chain = $1
  AND network = $2
  AND keyset_id = $3
  AND is_active = TRUE
  AND id <> $4
`
	if _, err := tx.ExecContext(ctx, query, target.Chain, target.Network, target.KeysetID, keepID); err != nil {
		return apperrors.NewInternal(
			"wallet_account_sync_failed",
			"failed to deactivate stale active wallet accounts",
			map[string]any{"error": err.Error()},
		)
	}
	return nil
}

func upsertWalletAccountHash(
	ctx context.Context,
	tx *sql.Tx,
	walletAccountID string,
	keyHash string,
	keyHashAlgo string,
	isActive bool,
) *apperrors.AppError {
	const query = `
UPDATE app.wallet_accounts
SET derivation_path_template = $2,
    key_material_hash = $3,
    key_material_hash_algo = $4,
    is_active = $5,
    updated_at = now()
WHERE id = $1
`
	result, err := tx.ExecContext(ctx, query, walletAccountID, derivationPathTemplate, keyHash, keyHashAlgo, isActive)
	if err != nil {
		return apperrors.NewInternal(
			"wallet_account_sync_failed",
			"failed to update wallet account hash",
			map[string]any{"error": err.Error(), "wallet_account_id": walletAccountID},
		)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternal(
			"wallet_account_sync_failed",
			"failed to read wallet-account hash update result",
			map[string]any{"error": err.Error(), "wallet_account_id": walletAccountID},
		)
	}
	if rows != 1 {
		return apperrors.NewInternal(
			"wallet_account_sync_failed",
			"wallet-account hash update affected unexpected rows",
			map[string]any{"wallet_account_id": walletAccountID, "rows": rows},
		)
	}
	return nil
}

func insertWalletAccount(
	ctx context.Context,
	tx *sql.Tx,
	walletAccountID string,
	target catalogSyncTarget,
	keyHash string,
	keyHashAlgo string,
) *apperrors.AppError {
	const query = `
INSERT INTO app.wallet_accounts (
  id,
  chain,
  network,
  keyset_id,
  derivation_path_template,
  next_index,
  is_active,
  key_material_hash,
  key_material_hash_algo,
  created_at,
  updated_at
)
VALUES ($1, $2, $3, $4, $5, 0, TRUE, $6, $7, now(), now())
`

	if _, err := tx.ExecContext(ctx, query, walletAccountID, target.Chain, target.Network, target.KeysetID, derivationPathTemplate, keyHash, keyHashAlgo); err != nil {
		return apperrors.NewInternal(
			"wallet_account_sync_failed",
			"failed to insert rotated wallet account",
			map[string]any{"error": err.Error(), "wallet_account_id": walletAccountID},
		)
	}

	return nil
}

func bindAssetCatalogWalletAccount(
	ctx context.Context,
	tx *sql.Tx,
	target catalogSyncTarget,
	walletAccountID string,
) *apperrors.AppError {
	const query = `
WITH updated AS (
  UPDATE app.asset_catalog ac
  SET wallet_account_id = $1,
      updated_at = now()
  FROM app.wallet_accounts wa
  WHERE ac.wallet_account_id = wa.id
    AND ac.chain = $2
    AND ac.network = $3
    AND wa.keyset_id = $4
  RETURNING 1
)
SELECT count(*) FROM updated
`

	var count int64
	if err := tx.QueryRowContext(ctx, query, walletAccountID, target.Chain, target.Network, target.KeysetID).Scan(&count); err != nil {
		return apperrors.NewInternal(
			"wallet_account_sync_failed",
			"failed to bind asset catalog wallet account",
			map[string]any{"error": err.Error(), "wallet_account_id": walletAccountID},
		)
	}
	if count == 0 {
		return apperrors.NewInternal(
			"invalid_configuration",
			"no asset catalog row found for wallet sync target",
			map[string]any{
				"chain":     target.Chain,
				"network":   target.Network,
				"keyset_id": target.KeysetID,
			},
		)
	}

	return nil
}

func insertWalletSyncEvent(
	ctx context.Context,
	tx *sql.Tx,
	target catalogSyncTarget,
	walletAccountID string,
	action string,
	matchSource string,
	keyHash string,
	keyHashAlgo string,
	details map[string]any,
) *apperrors.AppError {
	const query = `
INSERT INTO app.wallet_account_sync_events (
  chain,
  network,
  keyset_id,
  wallet_account_id,
  action,
  match_source,
  key_material_hash,
  key_material_hash_algo,
  details,
  created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
`

	detailsPayload := []byte("{}")
	if len(details) > 0 {
		encoded, err := json.Marshal(details)
		if err != nil {
			return apperrors.NewInternal(
				"wallet_account_sync_failed",
				"failed to encode wallet-account sync event details",
				map[string]any{"error": err.Error()},
			)
		}
		detailsPayload = encoded
	}

	if _, err := tx.ExecContext(
		ctx,
		query,
		target.Chain,
		target.Network,
		target.KeysetID,
		walletAccountID,
		action,
		matchSource,
		keyHash,
		keyHashAlgo,
		detailsPayload,
	); err != nil {
		return apperrors.NewInternal(
			"wallet_account_sync_failed",
			"failed to insert wallet-account sync event",
			map[string]any{"error": err.Error()},
		)
	}

	return nil
}

func classifyHashMatch(storedHash string, activeHash string, legacyHashes map[string]struct{}) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(storedHash))
	if normalized == "" {
		return matchSourceUnhashed, true
	}
	if normalized == activeHash {
		return matchSourceActive, true
	}
	if _, exists := legacyHashes[normalized]; exists {
		return matchSourceLegacy, true
	}
	return "", false
}

func computeHMACSHA256(secret string, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateWalletAccountID(chain string, network string, keyHash string) string {
	timestamp := time.Now().UTC().Format("20060102150405")
	nanos := time.Now().UTC().UnixNano() % 1000000
	return fmt.Sprintf("wa_%s_%s_%s_%06d_%s", chain, network, hashPrefix(keyHash), nanos, timestamp)
}

func hashPrefix(fullHash string) string {
	normalized := strings.ToLower(strings.TrimSpace(fullHash))
	if len(normalized) < 12 {
		return normalized
	}
	return normalized[:12]
}
