package database

import (
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func (db *DB) Migrate() error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("setting dialect: %w", err)
	}

	if err := goose.Up(db.DB, "migrations"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}

func (db *DB) MigrateDown() error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("setting dialect: %w", err)
	}

	if err := goose.Down(db.DB, "migrations"); err != nil {
		return fmt.Errorf("running down migration: %w", err)
	}

	return nil
}
