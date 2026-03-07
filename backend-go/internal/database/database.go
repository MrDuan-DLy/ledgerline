package database

import (
	"embed"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

var db *sqlx.DB

// Init opens the SQLite database and runs goose migrations.
func Init(dsn string) error {
	var err error
	db, err = sqlx.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	// Enable WAL mode for better concurrent reads
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("WARN: could not set WAL mode: %v", err)
	}

	// SQLite only supports one writer at a time; limit connections to avoid "database is locked" errors.
	db.SetMaxOpenConns(1)

	// Run goose migrations
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.Up(db.DB, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}

// DB returns the shared database handle.
func DB() *sqlx.DB {
	return db
}

// Close closes the database connection.
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
