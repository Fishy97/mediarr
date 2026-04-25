package database

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

func Open(configDir string) (*Store, error) {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(configDir, "media-steward.db"))
	if err != nil {
		return nil, err
	}
	store := &Store{DB: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (store *Store) Close() error {
	if store == nil || store.DB == nil {
		return nil
	}
	return store.DB.Close()
}

func (store *Store) migrate() error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	statements := []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS libraries (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			root TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS media_files (
			id TEXT PRIMARY KEY,
			library_id TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			canonical_key TEXT NOT NULL,
			title TEXT NOT NULL,
			kind TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			quality TEXT,
			fingerprint TEXT NOT NULL,
			modified_at TEXT NOT NULL,
			scanned_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS recommendations (
			id TEXT PRIMARY KEY,
			action TEXT NOT NULL,
			title TEXT NOT NULL,
			explanation TEXT NOT NULL,
			space_saved_bytes INTEGER NOT NULL,
			confidence REAL NOT NULL,
			source TEXT NOT NULL,
			affected_paths TEXT NOT NULL,
			destructive INTEGER NOT NULL DEFAULT 0,
			ignored_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS provider_cache (
			provider TEXT NOT NULL,
			cache_key TEXT NOT NULL,
			body TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			PRIMARY KEY (provider, cache_key)
		)`,
		`CREATE TABLE IF NOT EXISTS api_tokens (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			token_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			last_used_at TEXT
		)`,
	}
	for _, statement := range statements {
		if _, err := store.DB.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}
