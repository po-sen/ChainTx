package bootstrap

import (
	"context"
	"database/sql"
	stderrors "errors"
	"log"
	"path/filepath"
	"strings"

	portsout "chaintx/internal/application/ports/out"
	"chaintx/internal/infrastructure/walletkeys"
	apperrors "chaintx/internal/shared_kernel/errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	minExpiresInSeconds int64 = 60
	maxExpiresInSeconds int64 = 2_592_000
)

type ValidationRules struct {
	AllocationMode         string
	DevtestAllowMainnet    bool
	DevtestKeysets         map[string]string
	DevtestKeyNormalizers  map[string]DevtestKeyNormalizer
	AddressSchemeAllowList map[string]map[string]struct{}
	ModeStartupValidators  map[string]AllocationModeStartupValidator
	ModeCatalogValidators  map[string]AllocationModeCatalogRowValidator
}

type DevtestKeyNormalizer func(raw string) (walletkeys.ExtendedPublicKey, string, *walletkeys.KeyError)
type AllocationModeStartupValidator func(g *Gateway) *apperrors.AppError
type AllocationModeCatalogRowValidator func(g *Gateway, row catalogValidationRow, details map[string]any) *apperrors.AppError

type Gateway struct {
	databaseURL     string
	databaseTarget  string
	migrationsPath  string
	validationRules ValidationRules
	logger          *log.Logger
}

var _ portsout.PersistenceBootstrapGateway = (*Gateway)(nil)

func NewGateway(
	databaseURL string,
	databaseTarget string,
	migrationsPath string,
	validationRules ValidationRules,
	logger *log.Logger,
) *Gateway {
	return &Gateway{
		databaseURL:    databaseURL,
		databaseTarget: databaseTarget,
		migrationsPath: migrationsPath,
		validationRules: ValidationRules{
			AllocationMode:         strings.ToLower(strings.TrimSpace(validationRules.AllocationMode)),
			DevtestAllowMainnet:    validationRules.DevtestAllowMainnet,
			DevtestKeysets:         copyStringMap(validationRules.DevtestKeysets),
			DevtestKeyNormalizers:  mergeDevtestKeyNormalizers(validationRules.DevtestKeyNormalizers),
			AddressSchemeAllowList: copyAllowList(validationRules.AddressSchemeAllowList),
			ModeStartupValidators:  mergeModeStartupValidators(validationRules.ModeStartupValidators),
			ModeCatalogValidators:  mergeModeCatalogValidators(validationRules.ModeCatalogValidators),
		},
		logger: logger,
	}
}

func (g *Gateway) CheckReadiness(ctx context.Context) *apperrors.AppError {
	db, err := sql.Open("pgx", g.databaseURL)
	if err != nil {
		g.logf("database connection initialization failed target=%s error=%v", g.databaseTarget, err)
		return apperrors.NewInternal(
			"DB_CONNECT_INIT_FAILED",
			"failed to initialize database connection",
			map[string]any{"database_target": g.databaseTarget},
		)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		g.logf("database readiness check failed target=%s error=%v", g.databaseTarget, err)
		return apperrors.NewInternal(
			"DB_CONNECT_FAILED",
			"failed to connect to database",
			map[string]any{"database_target": g.databaseTarget},
		)
	}

	g.logf("database readiness check succeeded target=%s", g.databaseTarget)
	return nil
}

func (g *Gateway) RunMigrations(ctx context.Context) *apperrors.AppError {
	if err := ctx.Err(); err != nil {
		return apperrors.NewInternal(
			"DB_MIGRATION_CONTEXT_CANCELED",
			"migration context canceled",
			map[string]any{"database_target": g.databaseTarget},
		)
	}

	migrationsAbsPath, err := filepath.Abs(g.migrationsPath)
	if err != nil {
		return apperrors.NewInternal(
			"DB_MIGRATION_PATH_RESOLVE_FAILED",
			"failed to resolve migration path",
			map[string]any{"migrations_path": g.migrationsPath},
		)
	}

	sourceURL := "file://" + filepath.ToSlash(migrationsAbsPath)
	migrationRunner, err := migrate.New(sourceURL, g.databaseURL)
	if err != nil {
		return apperrors.NewInternal(
			"DB_MIGRATION_SETUP_FAILED",
			"failed to initialize migration runner",
			map[string]any{
				"database_target": g.databaseTarget,
				"migrations_path": g.migrationsPath,
			},
		)
	}

	defer func() {
		sourceErr, dbErr := migrationRunner.Close()
		if sourceErr != nil {
			g.logf("migration source close warning path=%s error=%v", g.migrationsPath, sourceErr)
		}
		if dbErr != nil {
			g.logf("migration db close warning target=%s error=%v", g.databaseTarget, dbErr)
		}
	}()

	err = migrationRunner.Up()
	if err != nil && !stderrors.Is(err, migrate.ErrNoChange) {
		g.logf("database migrations failed target=%s error=%v", g.databaseTarget, err)
		return apperrors.NewInternal(
			"DB_MIGRATION_APPLY_FAILED",
			"failed to apply migrations",
			map[string]any{
				"database_target": g.databaseTarget,
				"migrations_path": g.migrationsPath,
			},
		)
	}

	if stderrors.Is(err, migrate.ErrNoChange) {
		g.logf("database migrations up to date target=%s", g.databaseTarget)
	} else {
		g.logf("database migrations applied target=%s", g.databaseTarget)
	}

	return nil
}

func (g *Gateway) ValidateAssetCatalogIntegrity(ctx context.Context) *apperrors.AppError {
	mode := strings.ToLower(strings.TrimSpace(g.validationRules.AllocationMode))
	startupValidator, exists := g.validationRules.ModeStartupValidators[mode]
	if !exists {
		return apperrors.NewInternal(
			"invalid_configuration",
			"allocation mode is invalid",
			map[string]any{"allocation_mode": g.validationRules.AllocationMode},
		)
	}

	if appErr := startupValidator(g); appErr != nil {
		return appErr
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

	const query = `
SELECT
  ac.chain,
  ac.network,
  ac.asset,
  ac.wallet_account_id,
  ac.address_scheme,
  ac.default_expires_in_seconds,
  ac.chain_id,
  ac.token_standard,
  ac.token_contract,
  ac.token_decimals,
  wa.id,
  wa.chain,
  wa.network,
  wa.keyset_id,
  wa.derivation_path_template,
  wa.next_index,
  wa.is_active
FROM app.asset_catalog ac
LEFT JOIN app.wallet_accounts wa ON wa.id = ac.wallet_account_id
WHERE ac.enabled = TRUE
ORDER BY ac.chain, ac.network, ac.asset
`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return apperrors.NewInternal(
			"invalid_configuration",
			"failed to query enabled asset catalog rows during startup validation",
			map[string]any{"error": err.Error()},
		)
	}
	defer rows.Close()

	for rows.Next() {
		row, appErr := scanCatalogValidationRow(rows)
		if appErr != nil {
			return appErr
		}
		if appErr := g.validateCatalogRow(row); appErr != nil {
			return appErr
		}
	}

	if err := rows.Err(); err != nil {
		return apperrors.NewInternal(
			"invalid_configuration",
			"failed while iterating startup validation rows",
			map[string]any{"error": err.Error()},
		)
	}

	g.logf("asset catalog startup validation passed")
	return nil
}

type catalogValidationRow struct {
	Chain                   string
	Network                 string
	Asset                   string
	WalletAccountID         string
	AddressScheme           string
	DefaultExpiresInSeconds int64
	ChainID                 sql.NullInt64
	TokenStandard           sql.NullString
	TokenContract           sql.NullString
	TokenDecimals           sql.NullInt64
	WalletID                sql.NullString
	WalletChain             sql.NullString
	WalletNetwork           sql.NullString
	WalletKeysetID          sql.NullString
	WalletPathTemplate      sql.NullString
	WalletNextIndex         sql.NullInt64
	WalletIsActive          sql.NullBool
}

func scanCatalogValidationRow(rows *sql.Rows) (catalogValidationRow, *apperrors.AppError) {
	row := catalogValidationRow{}
	if err := rows.Scan(
		&row.Chain,
		&row.Network,
		&row.Asset,
		&row.WalletAccountID,
		&row.AddressScheme,
		&row.DefaultExpiresInSeconds,
		&row.ChainID,
		&row.TokenStandard,
		&row.TokenContract,
		&row.TokenDecimals,
		&row.WalletID,
		&row.WalletChain,
		&row.WalletNetwork,
		&row.WalletKeysetID,
		&row.WalletPathTemplate,
		&row.WalletNextIndex,
		&row.WalletIsActive,
	); err != nil {
		return catalogValidationRow{}, apperrors.NewInternal(
			"invalid_configuration",
			"failed to parse startup validation row",
			map[string]any{"error": err.Error()},
		)
	}

	row.Chain = strings.ToLower(strings.TrimSpace(row.Chain))
	row.Network = strings.ToLower(strings.TrimSpace(row.Network))
	row.Asset = strings.ToUpper(strings.TrimSpace(row.Asset))
	row.AddressScheme = strings.ToLower(strings.TrimSpace(row.AddressScheme))
	row.WalletAccountID = strings.TrimSpace(row.WalletAccountID)
	return row, nil
}

func (g *Gateway) validateCatalogRow(row catalogValidationRow) *apperrors.AppError {
	baseDetails := map[string]any{
		"chain":             row.Chain,
		"network":           row.Network,
		"asset":             row.Asset,
		"wallet_account_id": row.WalletAccountID,
	}

	if !row.WalletID.Valid || strings.TrimSpace(row.WalletID.String) == "" {
		return apperrors.NewInternal(
			"invalid_configuration",
			"enabled asset catalog row references missing wallet account",
			baseDetails,
		)
	}
	if !row.WalletIsActive.Valid || !row.WalletIsActive.Bool {
		return apperrors.NewInternal(
			"invalid_configuration",
			"enabled asset catalog row references inactive wallet account",
			baseDetails,
		)
	}
	if !row.WalletChain.Valid || !row.WalletNetwork.Valid {
		return apperrors.NewInternal(
			"invalid_configuration",
			"wallet account is missing chain/network",
			baseDetails,
		)
	}

	walletChain := strings.ToLower(strings.TrimSpace(row.WalletChain.String))
	walletNetwork := strings.ToLower(strings.TrimSpace(row.WalletNetwork.String))
	if walletChain != row.Chain || walletNetwork != row.Network {
		return apperrors.NewInternal(
			"invalid_configuration",
			"wallet account chain/network mismatch with asset catalog row",
			mergeDetails(baseDetails, map[string]any{
				"wallet_chain":   walletChain,
				"wallet_network": walletNetwork,
			}),
		)
	}

	if row.DefaultExpiresInSeconds < minExpiresInSeconds || row.DefaultExpiresInSeconds > maxExpiresInSeconds {
		return apperrors.NewInternal(
			"invalid_configuration",
			"default_expires_in_seconds is out of allowed range",
			mergeDetails(baseDetails, map[string]any{
				"default_expires_in_seconds": row.DefaultExpiresInSeconds,
			}),
		)
	}

	if !g.isAddressSchemeAllowed(row.Chain, row.AddressScheme) {
		return apperrors.NewInternal(
			"invalid_configuration",
			"address_scheme is not allowed for chain",
			mergeDetails(baseDetails, map[string]any{
				"address_scheme": row.AddressScheme,
			}),
		)
	}

	if row.Chain == "ethereum" && !row.ChainID.Valid {
		return apperrors.NewInternal(
			"invalid_configuration",
			"ethereum catalog row must define chain_id",
			baseDetails,
		)
	}

	isToken := row.TokenStandard.Valid
	if isToken {
		if !row.TokenContract.Valid || !row.TokenDecimals.Valid {
			return apperrors.NewInternal(
				"invalid_configuration",
				"token asset row is missing token metadata",
				baseDetails,
			)
		}
	} else if row.TokenContract.Valid || row.TokenDecimals.Valid {
		return apperrors.NewInternal(
			"invalid_configuration",
			"native asset row must not include token metadata",
			baseDetails,
		)
	}

	if !row.WalletNextIndex.Valid || row.WalletNextIndex.Int64 < 0 {
		return apperrors.NewInternal(
			"invalid_configuration",
			"wallet account next_index must be non-negative",
			baseDetails,
		)
	}

	if !row.WalletPathTemplate.Valid {
		return apperrors.NewInternal(
			"invalid_configuration",
			"wallet account derivation_path_template is required",
			baseDetails,
		)
	}
	pathTemplate := strings.TrimSpace(row.WalletPathTemplate.String)
	if keyErr := walletkeys.ValidateDerivationPathTemplate(pathTemplate); keyErr != nil {
		return mapWalletKeyError(keyErr, baseDetails)
	}

	if !row.WalletKeysetID.Valid || strings.TrimSpace(row.WalletKeysetID.String) == "" {
		return apperrors.NewInternal(
			"invalid_configuration",
			"wallet account keyset_id is required",
			baseDetails,
		)
	}
	keysetID := strings.TrimSpace(row.WalletKeysetID.String)

	mode := strings.ToLower(strings.TrimSpace(g.validationRules.AllocationMode))
	rowValidator, exists := g.validationRules.ModeCatalogValidators[mode]
	if !exists {
		return apperrors.NewInternal(
			"invalid_configuration",
			"unsupported allocation mode for startup validation",
			mergeDetails(baseDetails, map[string]any{
				"allocation_mode": mode,
			}),
		)
	}

	return rowValidator(g, row, mergeDetails(baseDetails, map[string]any{
		"keyset_id": keysetID,
	}))
}

func (g *Gateway) validateDevtestKeyset(chain, keysetID string, details map[string]any) *apperrors.AppError {
	rawKey, exists := g.validationRules.DevtestKeysets[keysetID]
	if !exists || strings.TrimSpace(rawKey) == "" {
		return apperrors.NewInternal(
			"invalid_configuration",
			"devtest keyset is missing for wallet account",
			mergeDetails(details, map[string]any{
				"keyset_id": keysetID,
			}),
		)
	}

	normalizer, exists := g.validationRules.DevtestKeyNormalizers[strings.ToLower(strings.TrimSpace(chain))]
	if !exists {
		return apperrors.NewInternal(
			"invalid_configuration",
			"unsupported chain for devtest keyset validation",
			mergeDetails(details, map[string]any{
				"keyset_id": keysetID,
				"chain":     chain,
			}),
		)
	}

	key, _, keyErr := normalizer(rawKey)
	if keyErr != nil {
		return mapWalletKeyError(keyErr, mergeDetails(details, map[string]any{
			"keyset_id": keysetID,
		}))
	}

	if keyErr := walletkeys.ValidateAccountLevelPolicy(key); keyErr != nil {
		return mapWalletKeyError(keyErr, mergeDetails(details, map[string]any{
			"keyset_id": keysetID,
		}))
	}

	return nil
}

func (g *Gateway) isAddressSchemeAllowed(chain, addressScheme string) bool {
	chainRules, ok := g.validationRules.AddressSchemeAllowList[strings.ToLower(strings.TrimSpace(chain))]
	if !ok {
		return false
	}
	_, exists := chainRules[strings.ToLower(strings.TrimSpace(addressScheme))]
	return exists
}

func mapWalletKeyError(keyErr *walletkeys.KeyError, details map[string]any) *apperrors.AppError {
	if keyErr == nil {
		return nil
	}

	code := string(keyErr.Code)
	if code == "" {
		code = "invalid_configuration"
	}

	mergedDetails := mergeDetails(details, map[string]any{
		"reason": keyErr.Message,
	})
	if keyErr.Cause != nil {
		mergedDetails["cause"] = keyErr.Cause.Error()
	}

	switch keyErr.Code {
	case walletkeys.CodeInvalidKeyMaterialFormat:
		return apperrors.NewInternal("invalid_key_material_format", keyErr.Message, mergedDetails)
	case walletkeys.CodeInvalidConfiguration:
		return apperrors.NewInternal("invalid_configuration", keyErr.Message, mergedDetails)
	case walletkeys.CodeUnsupportedTarget:
		return apperrors.NewInternal("invalid_configuration", keyErr.Message, mergedDetails)
	default:
		return apperrors.NewInternal(code, keyErr.Message, mergedDetails)
	}
}

func mergeDetails(base map[string]any, extras map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extras))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extras {
		out[key] = value
	}
	return out
}

func copyStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}

func copyAllowList(input map[string]map[string]struct{}) map[string]map[string]struct{} {
	if len(input) == 0 {
		return map[string]map[string]struct{}{}
	}
	out := make(map[string]map[string]struct{}, len(input))
	for chain, schemes := range input {
		normalizedChain := strings.ToLower(strings.TrimSpace(chain))
		if normalizedChain == "" {
			continue
		}
		out[normalizedChain] = map[string]struct{}{}
		for scheme := range schemes {
			normalizedScheme := strings.ToLower(strings.TrimSpace(scheme))
			if normalizedScheme == "" {
				continue
			}
			out[normalizedChain][normalizedScheme] = struct{}{}
		}
	}
	return out
}

func mergeDevtestKeyNormalizers(overrides map[string]DevtestKeyNormalizer) map[string]DevtestKeyNormalizer {
	normalizers := defaultDevtestKeyNormalizers()
	for chain, normalizer := range overrides {
		normalizedChain := strings.ToLower(strings.TrimSpace(chain))
		if normalizedChain == "" || normalizer == nil {
			continue
		}
		normalizers[normalizedChain] = normalizer
	}
	return normalizers
}

func defaultDevtestKeyNormalizers() map[string]DevtestKeyNormalizer {
	return map[string]DevtestKeyNormalizer{
		"bitcoin":  walletkeys.NormalizeBitcoinKeyset,
		"ethereum": walletkeys.NormalizeEVMKeyset,
	}
}

func mergeModeStartupValidators(overrides map[string]AllocationModeStartupValidator) map[string]AllocationModeStartupValidator {
	validators := defaultModeStartupValidators()
	for mode, validator := range overrides {
		normalizedMode := strings.ToLower(strings.TrimSpace(mode))
		if normalizedMode == "" || validator == nil {
			continue
		}
		validators[normalizedMode] = validator
	}
	return validators
}

func defaultModeStartupValidators() map[string]AllocationModeStartupValidator {
	return map[string]AllocationModeStartupValidator{
		"devtest": func(g *Gateway) *apperrors.AppError {
			if g.validationRules.DevtestAllowMainnet {
				g.logf("WARNING: devtest mainnet allocation override is enabled")
			}
			return nil
		},
		"prod": func(_ *Gateway) *apperrors.AppError {
			return apperrors.NewInternal(
				"wallet_allocation_not_implemented",
				"production wallet allocation mode is not implemented",
				nil,
			)
		},
	}
}

func mergeModeCatalogValidators(overrides map[string]AllocationModeCatalogRowValidator) map[string]AllocationModeCatalogRowValidator {
	validators := defaultModeCatalogValidators()
	for mode, validator := range overrides {
		normalizedMode := strings.ToLower(strings.TrimSpace(mode))
		if normalizedMode == "" || validator == nil {
			continue
		}
		validators[normalizedMode] = validator
	}
	return validators
}

func defaultModeCatalogValidators() map[string]AllocationModeCatalogRowValidator {
	return map[string]AllocationModeCatalogRowValidator{
		"devtest": func(g *Gateway, row catalogValidationRow, details map[string]any) *apperrors.AppError {
			if row.Network == "mainnet" && !g.validationRules.DevtestAllowMainnet {
				return apperrors.NewInternal(
					"invalid_configuration",
					"devtest mode blocks enabled mainnet allocator rows unless override is enabled",
					details,
				)
			}
			return g.validateDevtestKeyset(row.Chain, strings.TrimSpace(row.WalletKeysetID.String), details)
		},
		"prod": func(_ *Gateway, _ catalogValidationRow, _ map[string]any) *apperrors.AppError {
			return apperrors.NewInternal(
				"wallet_allocation_not_implemented",
				"production wallet allocation mode is not implemented",
				nil,
			)
		},
	}
}

func (g *Gateway) logf(format string, args ...any) {
	if g.logger == nil {
		return
	}
	g.logger.Printf(format, args...)
}
