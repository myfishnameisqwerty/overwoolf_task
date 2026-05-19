package db

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"tracker/internal/storage"

	_ "modernc.org/sqlite"
)

func New(path string) (storage.Storage, error) {
	sqlDB, err := Open(path, SchemaSQL)
	if err != nil {
		return nil, err
	}
	return NewStore(sqlDB), nil
}

func Open(path, schemaSQL string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// SQLite allows only one writer at a time. A single connection lets the
	// database/sql pool serialize writes without "database is locked" errors.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, err
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, err
	}

	slog.Info("database opened", "path", path)
	return db, nil
}
