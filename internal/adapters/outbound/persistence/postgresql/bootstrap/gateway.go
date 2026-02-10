package bootstrap

import (
	"context"
	"database/sql"
	stderrors "errors"
	"log"
	"path/filepath"

	portsout "chaintx/internal/application/ports/out"
	apperrors "chaintx/internal/shared_kernel/errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Gateway struct {
	databaseURL    string
	databaseTarget string
	migrationsPath string
	logger         *log.Logger
}

var _ portsout.PersistenceBootstrapGateway = (*Gateway)(nil)

func NewGateway(
	databaseURL string,
	databaseTarget string,
	migrationsPath string,
	logger *log.Logger,
) *Gateway {
	return &Gateway{
		databaseURL:    databaseURL,
		databaseTarget: databaseTarget,
		migrationsPath: migrationsPath,
		logger:         logger,
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
SELECT COUNT(*)
FROM app.asset_catalog ac
LEFT JOIN app.wallet_accounts wa ON wa.id = ac.wallet_account_id
WHERE ac.enabled = TRUE
  AND (
    wa.id IS NULL
    OR wa.is_active = FALSE
    OR wa.chain <> ac.chain
    OR wa.network <> ac.network
    OR wa.address_scheme <> ac.address_scheme
    OR ac.default_expires_in_seconds < 60
    OR ac.default_expires_in_seconds > 2592000
    OR (ac.chain = 'ethereum' AND ac.address_scheme LIKE 'evm%' AND ac.chain_id IS NULL)
    OR (ac.chain = 'ethereum' AND ac.address_scheme LIKE 'evm%' AND wa.chain_id IS DISTINCT FROM ac.chain_id)
  )
`

	var invalidCount int
	if err := db.QueryRowContext(ctx, query).Scan(&invalidCount); err != nil {
		return apperrors.NewInternal(
			"asset_catalog_validation_query_failed",
			"failed to validate asset catalog integrity",
			map[string]any{"error": err.Error()},
		)
	}
	if invalidCount > 0 {
		return apperrors.NewInternal(
			"asset_catalog_integrity_invalid",
			"asset catalog integrity validation failed",
			map[string]any{"invalid_rows": invalidCount},
		)
	}

	g.logf("asset catalog integrity validation passed")
	return nil
}

func (g *Gateway) logf(format string, args ...any) {
	if g.logger == nil {
		return
	}
	g.logger.Printf(format, args...)
}
