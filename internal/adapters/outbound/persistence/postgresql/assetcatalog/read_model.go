package assetcatalog

import (
	"context"
	"database/sql"
	"strings"

	"chaintx/internal/application/dto"
	portsout "chaintx/internal/application/ports/out"
	valueobjects "chaintx/internal/domain/value_objects"
	apperrors "chaintx/internal/shared_kernel/errors"
)

type ReadModel struct {
	db *sql.DB
}

var _ portsout.AssetCatalogReadModel = (*ReadModel)(nil)

func NewReadModel(db *sql.DB) *ReadModel {
	return &ReadModel{db: db}
}

func (r *ReadModel) ListEnabled(ctx context.Context) ([]dto.AssetCatalogEntry, *apperrors.AppError) {
	const query = `
SELECT
  chain,
  network,
  asset,
  minor_unit,
  decimals,
  address_scheme,
  default_expires_in_seconds,
  chain_id,
  token_standard,
  token_contract,
  token_decimals,
  wallet_account_id
FROM app.asset_catalog
WHERE enabled = TRUE
ORDER BY chain ASC, network ASC, asset ASC
`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, apperrors.NewInternal(
			"asset_catalog_query_failed",
			"failed to query asset catalog",
			map[string]any{"error": err.Error()},
		)
	}
	defer rows.Close()

	assets := make([]dto.AssetCatalogEntry, 0)
	for rows.Next() {
		var (
			entry           dto.AssetCatalogEntry
			chainID         sql.NullInt64
			tokenStandard   sql.NullString
			tokenContract   sql.NullString
			tokenDecimals   sql.NullInt64
			walletAccountID string
		)

		if scanErr := rows.Scan(
			&entry.Chain,
			&entry.Network,
			&entry.Asset,
			&entry.MinorUnit,
			&entry.Decimals,
			&entry.AddressScheme,
			&entry.DefaultExpiresInSeconds,
			&chainID,
			&tokenStandard,
			&tokenContract,
			&tokenDecimals,
			&walletAccountID,
		); scanErr != nil {
			return nil, apperrors.NewInternal(
				"asset_catalog_scan_failed",
				"failed to parse asset catalog row",
				map[string]any{"error": scanErr.Error()},
			)
		}

		entry.Chain = strings.ToLower(entry.Chain)
		entry.Network = strings.ToLower(entry.Network)
		entry.Asset = strings.ToUpper(entry.Asset)
		entry.WalletAccountID = walletAccountID

		if chainID.Valid {
			value := chainID.Int64
			entry.ChainID = &value
		}
		if tokenStandard.Valid {
			value := tokenStandard.String
			entry.TokenStandard = &value
		}
		if tokenContract.Valid {
			normalizedContract, appErr := valueobjects.NormalizeTokenContract(tokenContract.String)
			if appErr != nil {
				return nil, apperrors.NewInternal(
					"asset_catalog_token_contract_invalid",
					"asset catalog token_contract is invalid",
					map[string]any{"asset": entry.Asset, "chain": entry.Chain, "network": entry.Network},
				)
			}
			entry.TokenContract = &normalizedContract
		}
		if tokenDecimals.Valid {
			value := int(tokenDecimals.Int64)
			entry.TokenDecimals = &value
		}

		assets = append(assets, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternal(
			"asset_catalog_rows_failed",
			"failed to iterate asset catalog rows",
			map[string]any{"error": err.Error()},
		)
	}

	return assets, nil
}
