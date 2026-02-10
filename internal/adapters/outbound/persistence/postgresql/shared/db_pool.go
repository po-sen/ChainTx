package shared

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewDatabasePool(databaseURL string, logger *log.Logger) *sql.DB {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(30 * time.Minute)

	if logger != nil {
		logger.Printf("database pool initialized")
	}

	return db
}
