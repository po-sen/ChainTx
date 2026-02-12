package paymentrequest

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"log"
	"strings"
	"time"

	"chaintx/internal/application/dto"
	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"

	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct {
	db     *sql.DB
	logger *log.Logger
}

var _ portsout.PaymentRequestRepository = (*Repository)(nil)

func NewRepository(db *sql.DB, logger *log.Logger) *Repository {
	return &Repository{db: db, logger: logger}
}

func (r *Repository) Create(
	ctx context.Context,
	command dto.CreatePaymentRequestPersistenceCommand,
	resolveAddress dto.ResolvePaymentAddressFunc,
) (result dto.CreatePaymentRequestPersistenceResult, appErr *apperrors.AppError) {
	startedAt := time.Now()
	attemptResult := "failure"
	attemptReason := "unknown"
	derivationIndex := int64(-1)
	walletAccountID := command.AssetCatalogSnapshot.WalletAccountID
	defer func() {
		if appErr != nil {
			attemptReason = appErr.Code
		} else if attemptReason == "unknown" {
			attemptReason = attemptResult
		}

		latency := time.Since(startedAt)

		if r.logger != nil {
			r.logger.Printf(
				"wallet allocation attempt mode=%s chain=%s network=%s asset=%s wallet_account_id=%s derivation_index=%d result=%s reason=%s retry_count=0 latency_ms=%d",
				command.AllocationMode,
				command.Chain,
				command.Network,
				command.Asset,
				walletAccountID,
				derivationIndex,
				attemptResult,
				attemptReason,
				latency.Milliseconds(),
			)
		}
	}()

	if resolveAddress == nil {
		appErr = apperrors.NewInternal(
			"payment_address_resolver_missing",
			"payment address resolver is required",
			nil,
		)
		return result, appErr
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		appErr = apperrors.NewInternal(
			"payment_request_tx_begin_failed",
			"failed to start payment request transaction",
			map[string]any{"error": err.Error()},
		)
		return result, appErr
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	record, found, appErr := r.findIdempotencyRecordForUpdate(ctx, tx, command.IdempotencyScope, command.IdempotencyKey)
	if appErr != nil {
		return result, appErr
	}
	if found {
		if record.RequestHash != command.RequestHash {
			appErr = apperrors.NewConflict(
				"idempotency_key_conflict",
				"Idempotency key reused with different request payload",
				map[string]any{"idempotency_key": command.IdempotencyKey},
			)
			return result, appErr
		}

		var resource dto.PaymentRequestResource
		if unmarshalErr := json.Unmarshal(record.ResponsePayload, &resource); unmarshalErr != nil {
			appErr = apperrors.NewInternal(
				"idempotency_payload_invalid",
				"stored idempotency payload is invalid",
				map[string]any{"error": unmarshalErr.Error(), "resource_id": record.ResourceID},
			)
			return result, appErr
		}

		if commitErr := tx.Commit(); commitErr != nil {
			appErr = apperrors.NewInternal(
				"payment_request_tx_commit_failed",
				"failed to commit idempotency replay transaction",
				map[string]any{"error": commitErr.Error()},
			)
			return result, appErr
		}
		committed = true

		attemptResult = "replayed"
		attemptReason = "idempotency_replay"
		result = dto.CreatePaymentRequestPersistenceResult{Resource: resource, Replayed: true}
		return result, nil
	}

	wallet, appErr := r.lockWalletAccountForUpdate(ctx, tx, command.AssetCatalogSnapshot.WalletAccountID)
	if appErr != nil {
		return result, appErr
	}
	walletAccountID = wallet.ID
	derivationIndex = wallet.NextIndex

	if !wallet.IsActive {
		appErr = apperrors.NewInternal(
			"wallet_account_inactive",
			"wallet account is inactive",
			map[string]any{"wallet_account_id": wallet.ID},
		)
		return result, appErr
	}
	if wallet.Chain != command.Chain || wallet.Network != command.Network {
		appErr = apperrors.NewInternal(
			"asset_catalog_wallet_mismatch",
			"asset catalog mapping does not match wallet account chain/network",
			map[string]any{
				"wallet_account_id": wallet.ID,
				"wallet_chain":      wallet.Chain,
				"wallet_network":    wallet.Network,
				"request_chain":     command.Chain,
				"request_network":   command.Network,
			},
		)
		return result, appErr
	}

	allocation, appErr := resolveAddress(ctx, dto.ResolvePaymentAddressInput{
		Chain:                  command.Chain,
		Network:                command.Network,
		AddressScheme:          command.AssetCatalogSnapshot.AddressScheme,
		KeysetID:               wallet.KeysetID,
		DerivationPathTemplate: wallet.DerivationPathTemplate,
		DerivationIndex:        wallet.NextIndex,
		ChainID:                command.AssetCatalogSnapshot.ChainID,
	})
	if appErr != nil {
		return result, appErr
	}

	if appErr := r.insertPaymentRequest(ctx, tx, command, wallet.ID, wallet.NextIndex, allocation.AddressCanonical); appErr != nil {
		return result, appErr
	}

	resource := dto.PaymentRequestResource{
		ID:                  command.ResourceID,
		Status:              command.Status,
		Chain:               command.Chain,
		Network:             command.Network,
		Asset:               command.Asset,
		ExpectedAmountMinor: command.ExpectedAmountMinor,
		ExpiresAt:           command.ExpiresAt,
		CreatedAt:           command.CreatedAt,
		PaymentInstructions: dto.PaymentInstructions{
			Address:         allocation.Address,
			AddressScheme:   command.AssetCatalogSnapshot.AddressScheme,
			DerivationIndex: wallet.NextIndex,
			ChainID:         command.AssetCatalogSnapshot.ChainID,
			TokenStandard:   command.AssetCatalogSnapshot.TokenStandard,
			TokenContract:   command.AssetCatalogSnapshot.TokenContract,
			TokenDecimals:   command.AssetCatalogSnapshot.TokenDecimals,
		},
	}

	responsePayload, marshalErr := json.Marshal(resource)
	if marshalErr != nil {
		appErr = apperrors.NewInternal(
			"payment_request_payload_encode_failed",
			"failed to encode payment request payload",
			map[string]any{"error": marshalErr.Error()},
		)
		return result, appErr
	}

	if appErr := r.insertIdempotencyRecord(ctx, tx, command, responsePayload, command.IdempotencyExpiresAt); appErr != nil {
		if appErr.Code == "idempotency_key_conflict" {
			_ = tx.Rollback()
			committed = true

			replayedResource, replayFound, replayErr := r.loadReplayResource(ctx, command.IdempotencyScope, command.IdempotencyKey, command.RequestHash)
			if replayErr != nil {
				appErr = replayErr
				return result, appErr
			}
			if replayFound {
				attemptResult = "replayed"
				attemptReason = "idempotency_replay_race"
				result = dto.CreatePaymentRequestPersistenceResult{
					Resource: replayedResource,
					Replayed: true,
				}
				return result, nil
			}
		}

		return result, appErr
	}

	if appErr := r.bumpWalletNextIndex(ctx, tx, wallet.ID, wallet.NextIndex, command.CreatedAt); appErr != nil {
		return result, appErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		appErr = apperrors.NewInternal(
			"payment_request_tx_commit_failed",
			"failed to commit payment request transaction",
			map[string]any{"error": commitErr.Error()},
		)
		return result, appErr
	}
	committed = true

	attemptResult = "success"
	attemptReason = "created"
	result = dto.CreatePaymentRequestPersistenceResult{Resource: resource, Replayed: false}
	return result, nil
}

type idempotencyRecord struct {
	RequestHash     string
	ResourceID      string
	ResponsePayload []byte
}

func (r *Repository) findIdempotencyRecordForUpdate(
	ctx context.Context,
	tx *sql.Tx,
	scope dto.IdempotencyScope,
	idempotencyKey string,
) (idempotencyRecord, bool, *apperrors.AppError) {
	const query = `
SELECT request_hash, resource_id, response_payload
FROM app.idempotency_records
WHERE scope_principal = $1
  AND scope_method = $2
  AND scope_path = $3
  AND idempotency_key = $4
FOR UPDATE
`

	record := idempotencyRecord{}
	err := tx.QueryRowContext(
		ctx,
		query,
		scope.PrincipalID,
		scope.HTTPMethod,
		scope.HTTPPath,
		idempotencyKey,
	).Scan(&record.RequestHash, &record.ResourceID, &record.ResponsePayload)
	if err == nil {
		return record, true, nil
	}
	if stderrors.Is(err, sql.ErrNoRows) {
		return idempotencyRecord{}, false, nil
	}

	return idempotencyRecord{}, false, apperrors.NewInternal(
		"idempotency_record_query_failed",
		"failed to query idempotency record",
		map[string]any{"error": err.Error()},
	)
}

func (r *Repository) loadReplayResource(
	ctx context.Context,
	scope dto.IdempotencyScope,
	idempotencyKey string,
	expectedHash string,
) (dto.PaymentRequestResource, bool, *apperrors.AppError) {
	const query = `
SELECT request_hash, resource_id, response_payload
FROM app.idempotency_records
WHERE scope_principal = $1
  AND scope_method = $2
  AND scope_path = $3
  AND idempotency_key = $4
`

	record := idempotencyRecord{}
	err := r.db.QueryRowContext(ctx, query, scope.PrincipalID, scope.HTTPMethod, scope.HTTPPath, idempotencyKey).Scan(
		&record.RequestHash,
		&record.ResourceID,
		&record.ResponsePayload,
	)
	if stderrors.Is(err, sql.ErrNoRows) {
		return dto.PaymentRequestResource{}, false, nil
	}
	if err != nil {
		return dto.PaymentRequestResource{}, false, apperrors.NewInternal(
			"idempotency_record_query_failed",
			"failed to query idempotency record",
			map[string]any{"error": err.Error()},
		)
	}
	if record.RequestHash != expectedHash {
		return dto.PaymentRequestResource{}, false, apperrors.NewConflict(
			"idempotency_key_conflict",
			"Idempotency key reused with different request payload",
			map[string]any{"idempotency_key": idempotencyKey},
		)
	}

	var resource dto.PaymentRequestResource
	if unmarshalErr := json.Unmarshal(record.ResponsePayload, &resource); unmarshalErr != nil {
		return dto.PaymentRequestResource{}, false, apperrors.NewInternal(
			"idempotency_payload_invalid",
			"stored idempotency payload is invalid",
			map[string]any{"error": unmarshalErr.Error(), "resource_id": record.ResourceID},
		)
	}

	return resource, true, nil
}

type walletAccountRow struct {
	ID                     string
	Chain                  string
	Network                string
	KeysetID               string
	DerivationPathTemplate string
	NextIndex              int64
	IsActive               bool
}

func (r *Repository) lockWalletAccountForUpdate(ctx context.Context, tx *sql.Tx, walletAccountID string) (walletAccountRow, *apperrors.AppError) {
	const query = `
SELECT id, chain, network, keyset_id, derivation_path_template, next_index, is_active
FROM app.wallet_accounts
WHERE id = $1
FOR UPDATE
`

	walletAccount := walletAccountRow{}
	err := tx.QueryRowContext(ctx, query, walletAccountID).Scan(
		&walletAccount.ID,
		&walletAccount.Chain,
		&walletAccount.Network,
		&walletAccount.KeysetID,
		&walletAccount.DerivationPathTemplate,
		&walletAccount.NextIndex,
		&walletAccount.IsActive,
	)
	if stderrors.Is(err, sql.ErrNoRows) {
		return walletAccountRow{}, apperrors.NewInternal(
			"wallet_account_not_found",
			"wallet account mapping is invalid",
			map[string]any{"wallet_account_id": walletAccountID},
		)
	}
	if err != nil {
		return walletAccountRow{}, apperrors.NewInternal(
			"wallet_account_query_failed",
			"failed to query wallet account",
			map[string]any{"error": err.Error(), "wallet_account_id": walletAccountID},
		)
	}

	walletAccount.Chain = strings.ToLower(walletAccount.Chain)
	walletAccount.Network = strings.ToLower(walletAccount.Network)
	return walletAccount, nil
}

func (r *Repository) insertPaymentRequest(
	ctx context.Context,
	tx *sql.Tx,
	command dto.CreatePaymentRequestPersistenceCommand,
	walletAccountID string,
	derivationIndex int64,
	addressCanonical string,
) *apperrors.AppError {
	const insertSQL = `
INSERT INTO app.payment_requests (
  id,
  wallet_account_id,
  chain,
  network,
  asset,
  status,
  expected_amount_minor,
  address_canonical,
  address_scheme,
  derivation_index,
  chain_id,
  token_standard,
  token_contract,
  token_decimals,
  metadata,
  expires_at,
  created_at,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7,
  $8, $9, $10, $11, $12, $13, $14,
  $15, $16, $17, $18
	)
`

	metadata := command.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataBytes, marshalErr := json.Marshal(metadata)
	if marshalErr != nil {
		return apperrors.NewValidation(
			"invalid_request",
			"metadata must be valid JSON object",
			map[string]any{"field": "metadata"},
		)
	}

	var expectedAmount any
	if command.ExpectedAmountMinor != nil {
		expectedAmount = *command.ExpectedAmountMinor
	}

	_, err := tx.ExecContext(
		ctx,
		insertSQL,
		command.ResourceID,
		walletAccountID,
		command.Chain,
		command.Network,
		command.Asset,
		command.Status,
		expectedAmount,
		addressCanonical,
		command.AssetCatalogSnapshot.AddressScheme,
		derivationIndex,
		command.AssetCatalogSnapshot.ChainID,
		command.AssetCatalogSnapshot.TokenStandard,
		command.AssetCatalogSnapshot.TokenContract,
		command.AssetCatalogSnapshot.TokenDecimals,
		metadataBytes,
		command.ExpiresAt,
		command.CreatedAt,
		command.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.NewInternal(
				"address_allocation_conflict",
				"payment request uniqueness constraint failed",
				map[string]any{"error": err.Error()},
			)
		}

		return apperrors.NewInternal(
			"payment_request_insert_failed",
			"failed to insert payment request",
			map[string]any{"error": err.Error()},
		)
	}

	return nil
}

func (r *Repository) insertIdempotencyRecord(
	ctx context.Context,
	tx *sql.Tx,
	command dto.CreatePaymentRequestPersistenceCommand,
	responsePayload []byte,
	expiresAt time.Time,
) *apperrors.AppError {
	const insertSQL = `
INSERT INTO app.idempotency_records (
  scope_principal,
  scope_method,
  scope_path,
  idempotency_key,
  request_hash,
  hash_algorithm,
  resource_id,
  response_payload,
  created_at,
  expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`

	_, err := tx.ExecContext(
		ctx,
		insertSQL,
		command.IdempotencyScope.PrincipalID,
		command.IdempotencyScope.HTTPMethod,
		command.IdempotencyScope.HTTPPath,
		command.IdempotencyKey,
		command.RequestHash,
		command.HashAlgorithm,
		command.ResourceID,
		responsePayload,
		command.CreatedAt,
		expiresAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.NewConflict(
				"idempotency_key_conflict",
				"Idempotency key reused with different request payload",
				map[string]any{"idempotency_key": command.IdempotencyKey},
			)
		}

		return apperrors.NewInternal(
			"idempotency_record_insert_failed",
			"failed to insert idempotency record",
			map[string]any{"error": err.Error()},
		)
	}

	return nil
}

func (r *Repository) bumpWalletNextIndex(ctx context.Context, tx *sql.Tx, walletAccountID string, previousIndex int64, updatedAt time.Time) *apperrors.AppError {
	const updateSQL = `
UPDATE app.wallet_accounts
SET next_index = next_index + 1,
    updated_at = $3
WHERE id = $1 AND next_index = $2
`

	result, err := tx.ExecContext(ctx, updateSQL, walletAccountID, previousIndex, updatedAt)
	if err != nil {
		return apperrors.NewInternal(
			"wallet_account_update_failed",
			"failed to advance wallet account index",
			map[string]any{"error": err.Error(), "wallet_account_id": walletAccountID},
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternal(
			"wallet_account_update_result_failed",
			"failed to verify wallet account index update",
			map[string]any{"error": err.Error(), "wallet_account_id": walletAccountID},
		)
	}
	if rows != 1 {
		return apperrors.NewInternal(
			"wallet_account_index_conflict",
			"wallet account index update conflict",
			map[string]any{"wallet_account_id": walletAccountID},
		)
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !stderrors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == "23505"
}
