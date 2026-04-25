package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/catalog"
	"github.com/Fishy97/mediarr/backend/internal/filescan"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

type CatalogItem struct {
	ID           string       `json:"id"`
	LibraryID    string       `json:"libraryId"`
	Path         string       `json:"path"`
	CanonicalKey string       `json:"canonicalKey"`
	Title        string       `json:"title"`
	Kind         catalog.Kind `json:"kind"`
	SizeBytes    int64        `json:"sizeBytes"`
	Quality      string       `json:"quality,omitempty"`
	Fingerprint  string       `json:"fingerprint"`
	Subtitles    []string     `json:"subtitles"`
	ModifiedAt   time.Time    `json:"modifiedAt"`
	ScannedAt    time.Time    `json:"scannedAt"`
}

func Open(configDir string) (*Store, error) {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, err
	}
	path, err := databasePath(configDir)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
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

func databasePath(configDir string) (string, error) {
	current := filepath.Join(configDir, "mediarr.db")
	if _, err := os.Stat(current); err == nil {
		return current, nil
	}
	for _, legacy := range []string{"mediaar.db", "media-steward.db"} {
		legacyPath := filepath.Join(configDir, legacy)
		if _, err := os.Stat(legacyPath); err == nil {
			if err := os.Rename(legacyPath, current); err != nil {
				return "", err
			}
			return current, nil
		}
	}
	return current, nil
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
			subtitles TEXT NOT NULL DEFAULT '[]',
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
	if err := store.ensureColumn("media_files", "subtitles", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	return nil
}

func (store *Store) ensureColumn(table string, column string, definition string) error {
	_, err := store.DB.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return err
	}
	return nil
}

func (store *Store) SaveScan(scan filescan.Result) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	tx, err := store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	scannedAt := scan.CompletedAt
	if scannedAt.IsZero() {
		scannedAt = time.Now().UTC()
	}

	stmt, err := tx.Prepare(`INSERT INTO media_files (
		id, library_id, path, canonical_key, title, kind, size_bytes, quality, fingerprint, subtitles, modified_at, scanned_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(path) DO UPDATE SET
		id = excluded.id,
		library_id = excluded.library_id,
		canonical_key = excluded.canonical_key,
		title = excluded.title,
		kind = excluded.kind,
		size_bytes = excluded.size_bytes,
		quality = excluded.quality,
		fingerprint = excluded.fingerprint,
		subtitles = excluded.subtitles,
		modified_at = excluded.modified_at,
		scanned_at = excluded.scanned_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range scan.Items {
		itemSubtitles := item.Subtitles
		if itemSubtitles == nil {
			itemSubtitles = []string{}
		}
		subtitles, err := json.Marshal(itemSubtitles)
		if err != nil {
			return err
		}
		_, err = stmt.Exec(
			item.ID,
			item.LibraryID,
			item.Path,
			item.Parsed.CanonicalKey,
			item.Parsed.Title,
			string(item.Parsed.Kind),
			item.SizeBytes,
			item.Parsed.Quality,
			item.Fingerprint,
			string(subtitles),
			item.ModifiedAt.UTC().Format(time.RFC3339Nano),
			scannedAt.UTC().Format(time.RFC3339Nano),
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (store *Store) ListCatalog() ([]CatalogItem, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	rows, err := store.DB.Query(`SELECT id, library_id, path, canonical_key, title, kind, size_bytes, quality, fingerprint, subtitles, modified_at, scanned_at
		FROM media_files
		ORDER BY title, path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []CatalogItem{}
	for rows.Next() {
		var item CatalogItem
		var kind string
		var subtitles string
		var modifiedAt string
		var scannedAt string
		if err := rows.Scan(&item.ID, &item.LibraryID, &item.Path, &item.CanonicalKey, &item.Title, &kind, &item.SizeBytes, &item.Quality, &item.Fingerprint, &subtitles, &modifiedAt, &scannedAt); err != nil {
			return nil, err
		}
		item.Kind = catalog.Kind(kind)
		if err := json.Unmarshal([]byte(subtitles), &item.Subtitles); err != nil {
			return nil, err
		}
		if item.Subtitles == nil {
			item.Subtitles = []string{}
		}
		item.ModifiedAt, _ = time.Parse(time.RFC3339Nano, modifiedAt)
		item.ScannedAt, _ = time.Parse(time.RFC3339Nano, scannedAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (store *Store) ReplaceRecommendations(recs []recommendations.Recommendation) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	tx, err := store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM recommendations WHERE ignored_at IS NULL`); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO recommendations (
		id, action, title, explanation, space_saved_bytes, confidence, source, affected_paths, destructive
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, rec := range recs {
		paths, err := json.Marshal(rec.AffectedPaths)
		if err != nil {
			return err
		}
		destructive := 0
		if rec.Destructive {
			destructive = 1
		}
		if _, err := stmt.Exec(rec.ID, string(rec.Action), rec.Title, rec.Explanation, rec.SpaceSavedBytes, rec.Confidence, rec.Source, string(paths), destructive); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (store *Store) ListRecommendations() ([]recommendations.Recommendation, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	rows, err := store.DB.Query(`SELECT id, action, title, explanation, space_saved_bytes, confidence, source, affected_paths, destructive
		FROM recommendations
		WHERE ignored_at IS NULL
		ORDER BY space_saved_bytes DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recs := []recommendations.Recommendation{}
	for rows.Next() {
		var rec recommendations.Recommendation
		var action string
		var paths string
		var destructive int
		if err := rows.Scan(&rec.ID, &action, &rec.Title, &rec.Explanation, &rec.SpaceSavedBytes, &rec.Confidence, &rec.Source, &paths, &destructive); err != nil {
			return nil, err
		}
		rec.Action = recommendations.Action(action)
		rec.Destructive = destructive == 1
		if err := json.Unmarshal([]byte(paths), &rec.AffectedPaths); err != nil {
			return nil, err
		}
		recs = append(recs, rec)
	}
	return recs, rows.Err()
}
