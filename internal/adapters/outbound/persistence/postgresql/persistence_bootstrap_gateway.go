package postgresql

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

type PersistenceBootstrapGateway struct {
	databaseURL    string
	databaseTarget string
	migrationsPath string
	logger         *log.Logger
}

var _ portsout.PersistenceBootstrapGateway = (*PersistenceBootstrapGateway)(nil)

func NewPersistenceBootstrapGateway(
	databaseURL string,
	databaseTarget string,
	migrationsPath string,
	logger *log.Logger,
) *PersistenceBootstrapGateway {
	return &PersistenceBootstrapGateway{
		databaseURL:    databaseURL,
		databaseTarget: databaseTarget,
		migrationsPath: migrationsPath,
		logger:         logger,
	}
}

func (g *PersistenceBootstrapGateway) CheckReadiness(ctx context.Context) *apperrors.AppError {
	db, err := sql.Open("pgx", g.databaseURL)
	if err != nil {
		g.logger.Printf("database connection initialization failed target=%s error=%v", g.databaseTarget, err)
		return apperrors.NewInternal(
			"DB_CONNECT_INIT_FAILED",
			"failed to initialize database connection",
			map[string]string{"database_target": g.databaseTarget},
		)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		g.logger.Printf("database readiness check failed target=%s error=%v", g.databaseTarget, err)
		return apperrors.NewInternal(
			"DB_CONNECT_FAILED",
			"failed to connect to database",
			map[string]string{"database_target": g.databaseTarget},
		)
	}

	g.logger.Printf("database readiness check succeeded target=%s", g.databaseTarget)
	return nil
}

func (g *PersistenceBootstrapGateway) RunMigrations(ctx context.Context) *apperrors.AppError {
	if err := ctx.Err(); err != nil {
		return apperrors.NewInternal(
			"DB_MIGRATION_CONTEXT_CANCELED",
			"migration context canceled",
			map[string]string{"database_target": g.databaseTarget},
		)
	}

	migrationsAbsPath, err := filepath.Abs(g.migrationsPath)
	if err != nil {
		return apperrors.NewInternal(
			"DB_MIGRATION_PATH_RESOLVE_FAILED",
			"failed to resolve migration path",
			map[string]string{"migrations_path": g.migrationsPath},
		)
	}

	sourceURL := "file://" + filepath.ToSlash(migrationsAbsPath)
	migrationRunner, err := migrate.New(sourceURL, g.databaseURL)
	if err != nil {
		return apperrors.NewInternal(
			"DB_MIGRATION_SETUP_FAILED",
			"failed to initialize migration runner",
			map[string]string{
				"database_target": g.databaseTarget,
				"migrations_path": g.migrationsPath,
			},
		)
	}

	defer func() {
		sourceErr, dbErr := migrationRunner.Close()
		if sourceErr != nil {
			g.logger.Printf("migration source close warning path=%s error=%v", g.migrationsPath, sourceErr)
		}
		if dbErr != nil {
			g.logger.Printf("migration db close warning target=%s error=%v", g.databaseTarget, dbErr)
		}
	}()

	err = migrationRunner.Up()
	if err != nil && !stderrors.Is(err, migrate.ErrNoChange) {
		g.logger.Printf("database migrations failed target=%s error=%v", g.databaseTarget, err)
		return apperrors.NewInternal(
			"DB_MIGRATION_APPLY_FAILED",
			"failed to apply migrations",
			map[string]string{
				"database_target": g.databaseTarget,
				"migrations_path": g.migrationsPath,
			},
		)
	}

	if stderrors.Is(err, migrate.ErrNoChange) {
		g.logger.Printf("database migrations up to date target=%s", g.databaseTarget)
	} else {
		g.logger.Printf("database migrations applied target=%s", g.databaseTarget)
	}

	return nil
}
