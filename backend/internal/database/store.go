package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
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

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

type CatalogItem struct {
	ID                 string       `json:"id"`
	LibraryID          string       `json:"libraryId"`
	Path               string       `json:"path"`
	CanonicalKey       string       `json:"canonicalKey"`
	Title              string       `json:"title"`
	Kind               catalog.Kind `json:"kind"`
	Year               int          `json:"year,omitempty"`
	SizeBytes          int64        `json:"sizeBytes"`
	Quality            string       `json:"quality,omitempty"`
	Fingerprint        string       `json:"fingerprint"`
	Subtitles          []string     `json:"subtitles"`
	MetadataProvider   string       `json:"metadataProvider,omitempty"`
	MetadataProviderID string       `json:"metadataProviderId,omitempty"`
	MetadataConfidence float64      `json:"metadataConfidence,omitempty"`
	MetadataCorrected  bool         `json:"metadataCorrected"`
	ModifiedAt         time.Time    `json:"modifiedAt"`
	ScannedAt          time.Time    `json:"scannedAt"`
}

type ProviderSetting struct {
	Provider         string    `json:"provider"`
	BaseURL          string    `json:"baseUrl,omitempty"`
	APIKey           string    `json:"-"`
	APIKeyConfigured bool      `json:"apiKeyConfigured"`
	APIKeyLast4      string    `json:"apiKeyLast4,omitempty"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type ProviderSettingInput struct {
	Provider     string `json:"provider"`
	BaseURL      string `json:"baseUrl"`
	APIKey       string `json:"apiKey"`
	ClearAPIKey  bool   `json:"clearApiKey"`
	ClearBaseURL bool   `json:"clearBaseUrl"`
}

type IntegrationSetting struct {
	Integration             string    `json:"integration"`
	BaseURL                 string    `json:"baseUrl,omitempty"`
	APIKey                  string    `json:"-"`
	APIKeyConfigured        bool      `json:"apiKeyConfigured"`
	APIKeyLast4             string    `json:"apiKeyLast4,omitempty"`
	AutoSyncEnabled         bool      `json:"autoSyncEnabled"`
	AutoSyncIntervalMinutes int       `json:"autoSyncIntervalMinutes"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

type IntegrationSettingInput struct {
	Integration             string `json:"integration"`
	BaseURL                 string `json:"baseUrl"`
	APIKey                  string `json:"apiKey"`
	ClearAPIKey             bool   `json:"clearApiKey"`
	ClearBaseURL            bool   `json:"clearBaseUrl"`
	AutoSyncEnabled         *bool  `json:"autoSyncEnabled,omitempty"`
	AutoSyncIntervalMinutes int    `json:"autoSyncIntervalMinutes,omitempty"`
}

type CatalogCorrection struct {
	MediaFileID  string       `json:"mediaFileId"`
	Title        string       `json:"title"`
	Kind         catalog.Kind `json:"kind"`
	Year         int          `json:"year,omitempty"`
	CanonicalKey string       `json:"canonicalKey"`
	Provider     string       `json:"provider,omitempty"`
	ProviderID   string       `json:"providerId,omitempty"`
	Confidence   float64      `json:"confidence"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type CatalogCorrectionInput struct {
	Title        string       `json:"title"`
	Kind         catalog.Kind `json:"kind"`
	Year         int          `json:"year,omitempty"`
	CanonicalKey string       `json:"canonicalKey"`
	Provider     string       `json:"provider,omitempty"`
	ProviderID   string       `json:"providerId,omitempty"`
	Confidence   float64      `json:"confidence"`
}

type MediaServer struct {
	ID           string    `json:"id"`
	Kind         string    `json:"kind"`
	Name         string    `json:"name"`
	BaseURL      string    `json:"baseUrl"`
	Status       string    `json:"status"`
	LastSyncedAt time.Time `json:"lastSyncedAt,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type MediaServerUser struct {
	ServerID    string    `json:"serverId"`
	ExternalID  string    `json:"externalId"`
	DisplayName string    `json:"displayName"`
	LastSeenAt  time.Time `json:"lastSeenAt,omitempty"`
}

type MediaServerLibrary struct {
	ServerID   string `json:"serverId"`
	ExternalID string `json:"externalId"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	ItemCount  int    `json:"itemCount"`
}

type MediaServerItem struct {
	ServerID          string            `json:"serverId"`
	ExternalID        string            `json:"externalId"`
	LibraryExternalID string            `json:"libraryExternalId"`
	ParentExternalID  string            `json:"parentExternalId,omitempty"`
	Kind              string            `json:"kind"`
	Title             string            `json:"title"`
	Year              int               `json:"year,omitempty"`
	Path              string            `json:"path,omitempty"`
	ProviderIDs       map[string]string `json:"providerIds"`
	RuntimeSeconds    int               `json:"runtimeSeconds,omitempty"`
	DateCreated       time.Time         `json:"dateCreated,omitempty"`
	MatchConfidence   float64           `json:"matchConfidence"`
	UpdatedAt         time.Time         `json:"updatedAt"`
}

type MediaServerFile struct {
	ServerID         string  `json:"serverId"`
	ItemExternalID   string  `json:"itemExternalId"`
	Path             string  `json:"path"`
	SizeBytes        int64   `json:"sizeBytes"`
	Container        string  `json:"container,omitempty"`
	LocalPath        string  `json:"localPath,omitempty"`
	LocalMediaFileID string  `json:"localMediaFileId,omitempty"`
	Verification     string  `json:"verification"`
	MatchConfidence  float64 `json:"matchConfidence"`
}

type MediaActivityRollup struct {
	ServerID       string    `json:"serverId"`
	ItemExternalID string    `json:"itemExternalId"`
	PlayCount      int       `json:"playCount"`
	UniqueUsers    int       `json:"uniqueUsers"`
	WatchedUsers   int       `json:"watchedUsers"`
	FavoriteCount  int       `json:"favoriteCount"`
	LastPlayedAt   time.Time `json:"lastPlayedAt,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type MediaSyncJob struct {
	ID              string    `json:"id"`
	ServerID        string    `json:"serverId"`
	Status          string    `json:"status"`
	Phase           string    `json:"phase,omitempty"`
	Message         string    `json:"message,omitempty"`
	CurrentLabel    string    `json:"currentLabel,omitempty"`
	Processed       int       `json:"processed,omitempty"`
	Total           int       `json:"total,omitempty"`
	ItemsImported   int       `json:"itemsImported"`
	RollupsImported int       `json:"rollupsImported"`
	UnmappedItems   int       `json:"unmappedItems"`
	Cursor          string    `json:"cursor,omitempty"`
	Error           string    `json:"error,omitempty"`
	StartedAt       time.Time `json:"startedAt"`
	CompletedAt     time.Time `json:"completedAt,omitempty"`
}

type Job struct {
	ID              string    `json:"id"`
	Kind            string    `json:"kind"`
	TargetID        string    `json:"targetId,omitempty"`
	Status          string    `json:"status"`
	Phase           string    `json:"phase"`
	Message         string    `json:"message"`
	CurrentLabel    string    `json:"currentLabel,omitempty"`
	Processed       int       `json:"processed"`
	Total           int       `json:"total"`
	ItemsImported   int       `json:"itemsImported"`
	RollupsImported int       `json:"rollupsImported"`
	UnmappedItems   int       `json:"unmappedItems"`
	Error           string    `json:"error,omitempty"`
	StartedAt       time.Time `json:"startedAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	CompletedAt     time.Time `json:"completedAt,omitempty"`
}

type JobInput struct {
	Kind         string
	TargetID     string
	Status       string
	Phase        string
	Message      string
	CurrentLabel string
	Total        int
}

type JobUpdate struct {
	Status          string
	Phase           string
	Message         string
	CurrentLabel    string
	Processed       *int
	Total           *int
	ItemsImported   *int
	RollupsImported *int
	UnmappedItems   *int
	Error           string
	Completed       bool
}

type JobFilter struct {
	Kind     string
	TargetID string
	Status   string
	Active   bool
	Limit    int
}

type JobEvent struct {
	ID           string    `json:"id"`
	JobID        string    `json:"jobId"`
	Level        string    `json:"level"`
	Phase        string    `json:"phase"`
	Message      string    `json:"message"`
	CurrentLabel string    `json:"currentLabel,omitempty"`
	Processed    int       `json:"processed"`
	Total        int       `json:"total"`
	CreatedAt    time.Time `json:"createdAt"`
}

type JobEventInput struct {
	JobID        string
	Level        string
	Phase        string
	Message      string
	CurrentLabel string
	Processed    int
	Total        int
}

type PathMapping struct {
	ID               string    `json:"id"`
	ServerID         string    `json:"serverId,omitempty"`
	ServerPathPrefix string    `json:"serverPathPrefix"`
	LocalPathPrefix  string    `json:"localPathPrefix"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type PathMappingVerification struct {
	Mapping       PathMapping `json:"mapping"`
	MatchedFiles  int         `json:"matchedFiles"`
	MappedFiles   int         `json:"mappedFiles"`
	VerifiedFiles int         `json:"verifiedFiles"`
	MissingFiles  int         `json:"missingFiles"`
	UpdatedAt     time.Time   `json:"updatedAt"`
}

type MediaServerSnapshot struct {
	Server    MediaServer           `json:"server"`
	Users     []MediaServerUser     `json:"users"`
	Libraries []MediaServerLibrary  `json:"libraries"`
	Items     []MediaServerItem     `json:"items"`
	Files     []MediaServerFile     `json:"files"`
	Rollups   []MediaActivityRollup `json:"rollups"`
	Job       MediaSyncJob          `json:"job"`
}

type MediaServerItemFilter struct {
	ServerID     string
	UnmappedOnly bool
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
			year INTEGER NOT NULL DEFAULT 0,
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
			state TEXT NOT NULL DEFAULT 'new',
			title TEXT NOT NULL,
			explanation TEXT NOT NULL,
			space_saved_bytes INTEGER NOT NULL,
			confidence REAL NOT NULL,
			source TEXT NOT NULL,
			affected_paths TEXT NOT NULL,
			destructive INTEGER NOT NULL DEFAULT 0,
			ai_rationale TEXT NOT NULL DEFAULT '',
			ai_tags TEXT NOT NULL DEFAULT '[]',
			ai_confidence REAL NOT NULL DEFAULT 0,
			ai_source TEXT NOT NULL DEFAULT '',
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
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token_hash TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			last_used_at TEXT,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS provider_settings (
			provider TEXT PRIMARY KEY,
			base_url TEXT NOT NULL DEFAULT '',
			api_key TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS integration_settings (
			integration TEXT PRIMARY KEY,
			base_url TEXT NOT NULL DEFAULT '',
			api_key TEXT NOT NULL DEFAULT '',
			auto_sync_enabled INTEGER NOT NULL DEFAULT 1,
			auto_sync_interval_minutes INTEGER NOT NULL DEFAULT 360,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			target_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			phase TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			current_label TEXT NOT NULL DEFAULT '',
			processed INTEGER NOT NULL DEFAULT 0,
			total INTEGER NOT NULL DEFAULT 0,
			items_imported INTEGER NOT NULL DEFAULT 0,
			rollups_imported INTEGER NOT NULL DEFAULT 0,
			unmapped_items INTEGER NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '',
			started_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			completed_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS job_events (
			id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			level TEXT NOT NULL,
			phase TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL,
			current_label TEXT NOT NULL DEFAULT '',
			processed INTEGER NOT NULL DEFAULT 0,
			total INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			FOREIGN KEY(job_id) REFERENCES jobs(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS catalog_corrections (
			media_file_id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			kind TEXT NOT NULL,
			year INTEGER NOT NULL DEFAULT 0,
			canonical_key TEXT NOT NULL,
			provider TEXT NOT NULL DEFAULT '',
			provider_id TEXT NOT NULL DEFAULT '',
			confidence REAL NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(media_file_id) REFERENCES media_files(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS media_servers (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			base_url TEXT NOT NULL,
			status TEXT NOT NULL,
			last_synced_at TEXT,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS media_server_users (
			server_id TEXT NOT NULL,
			external_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			last_seen_at TEXT,
			PRIMARY KEY (server_id, external_id)
		)`,
		`CREATE TABLE IF NOT EXISTS media_server_libraries (
			server_id TEXT NOT NULL,
			external_id TEXT NOT NULL,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			item_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (server_id, external_id)
		)`,
		`CREATE TABLE IF NOT EXISTS media_server_items (
			server_id TEXT NOT NULL,
			external_id TEXT NOT NULL,
			library_external_id TEXT NOT NULL DEFAULT '',
			parent_external_id TEXT NOT NULL DEFAULT '',
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			year INTEGER NOT NULL DEFAULT 0,
			path TEXT NOT NULL DEFAULT '',
			provider_ids TEXT NOT NULL DEFAULT '{}',
			runtime_seconds INTEGER NOT NULL DEFAULT 0,
			date_created TEXT,
			match_confidence REAL NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (server_id, external_id)
		)`,
		`CREATE TABLE IF NOT EXISTS media_server_files (
			server_id TEXT NOT NULL,
			item_external_id TEXT NOT NULL,
			path TEXT NOT NULL,
			size_bytes INTEGER NOT NULL DEFAULT 0,
			container TEXT NOT NULL DEFAULT '',
			local_path TEXT NOT NULL DEFAULT '',
			local_media_file_id TEXT NOT NULL DEFAULT '',
			verification TEXT NOT NULL DEFAULT 'server_reported',
			match_confidence REAL NOT NULL DEFAULT 0,
			PRIMARY KEY (server_id, item_external_id, path)
		)`,
		`CREATE TABLE IF NOT EXISTS media_activity_rollups (
			server_id TEXT NOT NULL,
			item_external_id TEXT NOT NULL,
			play_count INTEGER NOT NULL DEFAULT 0,
			unique_users INTEGER NOT NULL DEFAULT 0,
			watched_users INTEGER NOT NULL DEFAULT 0,
			favorite_count INTEGER NOT NULL DEFAULT 0,
			last_played_at TEXT,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (server_id, item_external_id)
		)`,
		`CREATE TABLE IF NOT EXISTS media_sync_jobs (
			id TEXT PRIMARY KEY,
			server_id TEXT NOT NULL,
			status TEXT NOT NULL,
			items_imported INTEGER NOT NULL DEFAULT 0,
			rollups_imported INTEGER NOT NULL DEFAULT 0,
			unmapped_items INTEGER NOT NULL DEFAULT 0,
			cursor TEXT NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '',
			started_at TEXT NOT NULL,
			completed_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS integration_path_mappings (
			id TEXT PRIMARY KEY,
			server_id TEXT NOT NULL DEFAULT '',
			server_path_prefix TEXT NOT NULL,
			local_path_prefix TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
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
	if err := store.ensureColumn("media_files", "year", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	for _, column := range []struct {
		name       string
		definition string
	}{
		{name: "auto_sync_enabled", definition: "INTEGER NOT NULL DEFAULT 1"},
		{name: "auto_sync_interval_minutes", definition: "INTEGER NOT NULL DEFAULT 360"},
	} {
		if err := store.ensureColumn("integration_settings", column.name, column.definition); err != nil {
			return err
		}
	}
	for _, column := range []struct {
		name       string
		definition string
	}{
		{name: "state", definition: "TEXT NOT NULL DEFAULT 'new'"},
		{name: "ai_rationale", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "ai_tags", definition: "TEXT NOT NULL DEFAULT '[]'"},
		{name: "ai_confidence", definition: "REAL NOT NULL DEFAULT 0"},
		{name: "ai_source", definition: "TEXT NOT NULL DEFAULT ''"},
	} {
		if err := store.ensureColumn("recommendations", column.name, column.definition); err != nil {
			return err
		}
	}
	for _, column := range []struct {
		name       string
		definition string
	}{
		{name: "server_id", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "external_item_id", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "last_played_at", definition: "TEXT"},
		{name: "play_count", definition: "INTEGER NOT NULL DEFAULT 0"},
		{name: "unique_users", definition: "INTEGER NOT NULL DEFAULT 0"},
		{name: "favorite_count", definition: "INTEGER NOT NULL DEFAULT 0"},
		{name: "verification", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "evidence", definition: "TEXT NOT NULL DEFAULT '{}'"},
	} {
		if err := store.ensureColumn("recommendations", column.name, column.definition); err != nil {
			return err
		}
	}
	return nil
}

func (store *Store) AdminUserExists() (bool, error) {
	if store == nil || store.DB == nil {
		return false, errors.New("nil database store")
	}
	var count int
	if err := store.DB.QueryRow(`SELECT COUNT(1) FROM users WHERE role = 'admin'`).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (store *Store) CreateAdminUser(email string, passwordHash string) (User, error) {
	if store == nil || store.DB == nil {
		return User{}, errors.New("nil database store")
	}
	email = normalizeEmail(email)
	if email == "" {
		return User{}, errors.New("email is required")
	}
	if strings.TrimSpace(passwordHash) == "" {
		return User{}, errors.New("password hash is required")
	}
	exists, err := store.AdminUserExists()
	if err != nil {
		return User{}, err
	}
	if exists {
		return User{}, errors.New("admin user already exists")
	}
	user := User{
		ID:           randomID("usr"),
		Email:        email,
		Role:         "admin",
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}
	_, err = store.DB.Exec(
		`INSERT INTO users (id, email, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)`,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (store *Store) UserByEmail(email string) (User, error) {
	return store.userFromQuery(`SELECT id, email, password_hash, role, created_at FROM users WHERE email = ?`, normalizeEmail(email))
}

func (store *Store) UserByID(id string) (User, error) {
	return store.userFromQuery(`SELECT id, email, password_hash, role, created_at FROM users WHERE id = ?`, id)
}

func (store *Store) CreateSession(tokenHash string, userID string, expiresAt time.Time) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	if tokenHash == "" || userID == "" {
		return errors.New("token hash and user id are required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := store.DB.Exec(
		`INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		tokenHash,
		userID,
		now,
		expiresAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (store *Store) UserBySessionHash(tokenHash string, now time.Time) (User, error) {
	if store == nil || store.DB == nil {
		return User{}, errors.New("nil database store")
	}
	user, err := store.userFromQuery(
		`SELECT users.id, users.email, users.password_hash, users.role, users.created_at
		FROM sessions
		JOIN users ON users.id = sessions.user_id
		WHERE sessions.token_hash = ? AND sessions.expires_at > ?`,
		tokenHash,
		now.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return User{}, err
	}
	_, _ = store.DB.Exec(`UPDATE sessions SET last_used_at = ? WHERE token_hash = ?`, now.UTC().Format(time.RFC3339Nano), tokenHash)
	return user, nil
}

func (store *Store) DeleteSession(tokenHash string) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	_, err := store.DB.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (store *Store) DeleteExpiredSessions(now time.Time) (int, error) {
	if store == nil || store.DB == nil {
		return 0, errors.New("nil database store")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result, err := store.DB.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(affected), nil
}

func (store *Store) userFromQuery(query string, args ...any) (User, error) {
	if store == nil || store.DB == nil {
		return User{}, errors.New("nil database store")
	}
	var user User
	var createdAt string
	err := store.DB.QueryRow(query, args...).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &createdAt)
	if err != nil {
		return User{}, err
	}
	user.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return user, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func randomID(prefix string) string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
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
		id, library_id, path, canonical_key, title, kind, year, size_bytes, quality, fingerprint, subtitles, modified_at, scanned_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(path) DO UPDATE SET
		library_id = excluded.library_id,
		canonical_key = excluded.canonical_key,
		title = excluded.title,
		kind = excluded.kind,
		year = excluded.year,
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
			item.Parsed.Year,
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
	if err := pruneStaleMedia(tx, scan.LibraryID, scan.Items); err != nil {
		return err
	}
	return tx.Commit()
}

func pruneStaleMedia(tx *sql.Tx, libraryID string, items []filescan.Item) error {
	if libraryID == "" {
		return nil
	}
	if len(items) == 0 {
		_, err := tx.Exec(`DELETE FROM media_files WHERE library_id = ?`, libraryID)
		return err
	}
	placeholders := make([]string, 0, len(items))
	args := make([]any, 0, len(items)+1)
	args = append(args, libraryID)
	for _, item := range items {
		placeholders = append(placeholders, "?")
		args = append(args, item.Path)
	}
	_, err := tx.Exec(`DELETE FROM media_files WHERE library_id = ? AND path NOT IN (`+strings.Join(placeholders, ",")+`)`, args...)
	return err
}

func (store *Store) ListCatalog() ([]CatalogItem, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	rows, err := store.DB.Query(`SELECT
			media_files.id,
			media_files.library_id,
			media_files.path,
			COALESCE(catalog_corrections.canonical_key, media_files.canonical_key),
			COALESCE(catalog_corrections.title, media_files.title),
			COALESCE(catalog_corrections.kind, media_files.kind),
			COALESCE(catalog_corrections.year, media_files.year),
			media_files.size_bytes,
			media_files.quality,
			media_files.fingerprint,
			media_files.subtitles,
			COALESCE(catalog_corrections.provider, ''),
			COALESCE(catalog_corrections.provider_id, ''),
			COALESCE(catalog_corrections.confidence, 0),
			catalog_corrections.updated_at,
			media_files.modified_at,
			media_files.scanned_at
		FROM media_files
		LEFT JOIN catalog_corrections ON catalog_corrections.media_file_id = media_files.id
		ORDER BY COALESCE(catalog_corrections.title, media_files.title), media_files.path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []CatalogItem{}
	for rows.Next() {
		var item CatalogItem
		var kind string
		var subtitles string
		var correctedAt sql.NullString
		var modifiedAt string
		var scannedAt string
		if err := rows.Scan(
			&item.ID,
			&item.LibraryID,
			&item.Path,
			&item.CanonicalKey,
			&item.Title,
			&kind,
			&item.Year,
			&item.SizeBytes,
			&item.Quality,
			&item.Fingerprint,
			&subtitles,
			&item.MetadataProvider,
			&item.MetadataProviderID,
			&item.MetadataConfidence,
			&correctedAt,
			&modifiedAt,
			&scannedAt,
		); err != nil {
			return nil, err
		}
		item.Kind = catalog.Kind(kind)
		item.MetadataCorrected = correctedAt.Valid
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

func (store *Store) ReplaceMediaServerSnapshot(snapshot MediaServerSnapshot) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	serverID := strings.TrimSpace(snapshot.Server.ID)
	if serverID == "" {
		return errors.New("media server id is required")
	}
	now := time.Now().UTC()
	snapshot.Server.ID = serverID
	snapshot.Server.Kind = strings.ToLower(strings.TrimSpace(snapshot.Server.Kind))
	snapshot.Server.Name = strings.TrimSpace(snapshot.Server.Name)
	snapshot.Server.BaseURL = strings.TrimRight(strings.TrimSpace(snapshot.Server.BaseURL), "/")
	snapshot.Server.Status = strings.TrimSpace(snapshot.Server.Status)
	if snapshot.Server.Status == "" {
		snapshot.Server.Status = "configured"
	}
	if snapshot.Server.UpdatedAt.IsZero() {
		snapshot.Server.UpdatedAt = now
	}
	if snapshot.Job.ID == "" {
		snapshot.Job.ID = randomID("sync")
	}
	if snapshot.Job.ServerID == "" {
		snapshot.Job.ServerID = serverID
	}
	if snapshot.Job.Status == "" {
		snapshot.Job.Status = "completed"
	}
	if snapshot.Job.StartedAt.IsZero() {
		snapshot.Job.StartedAt = now
	}
	if snapshot.Job.CompletedAt.IsZero() && snapshot.Job.Status == "completed" {
		snapshot.Job.CompletedAt = now
	}
	if snapshot.Server.LastSyncedAt.IsZero() {
		snapshot.Server.LastSyncedAt = snapshot.Job.CompletedAt
	}
	tx, err := store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`INSERT INTO media_servers (
		id, kind, name, base_url, status, last_synced_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		kind = excluded.kind,
		name = excluded.name,
		base_url = excluded.base_url,
		status = excluded.status,
		last_synced_at = excluded.last_synced_at,
		updated_at = excluded.updated_at`,
		snapshot.Server.ID,
		snapshot.Server.Kind,
		snapshot.Server.Name,
		snapshot.Server.BaseURL,
		snapshot.Server.Status,
		formatOptionalTime(snapshot.Server.LastSyncedAt),
		snapshot.Server.UpdatedAt.Format(time.RFC3339Nano),
	); err != nil {
		return err
	}

	for _, table := range []string{"media_server_users", "media_server_libraries", "media_server_items", "media_server_files", "media_activity_rollups"} {
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE server_id = ?`, serverID); err != nil {
			return err
		}
	}

	userStmt, err := tx.Prepare(`INSERT INTO media_server_users (server_id, external_id, display_name, last_seen_at) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer userStmt.Close()
	for _, user := range snapshot.Users {
		user.ServerID = defaultString(user.ServerID, serverID)
		if strings.TrimSpace(user.ExternalID) == "" {
			continue
		}
		if _, err := userStmt.Exec(user.ServerID, strings.TrimSpace(user.ExternalID), strings.TrimSpace(user.DisplayName), formatOptionalTime(user.LastSeenAt)); err != nil {
			return err
		}
	}

	libraryStmt, err := tx.Prepare(`INSERT INTO media_server_libraries (server_id, external_id, name, kind, item_count) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer libraryStmt.Close()
	for _, library := range snapshot.Libraries {
		library.ServerID = defaultString(library.ServerID, serverID)
		if strings.TrimSpace(library.ExternalID) == "" {
			continue
		}
		if _, err := libraryStmt.Exec(library.ServerID, strings.TrimSpace(library.ExternalID), strings.TrimSpace(library.Name), strings.TrimSpace(library.Kind), library.ItemCount); err != nil {
			return err
		}
	}

	itemStmt, err := tx.Prepare(`INSERT INTO media_server_items (
		server_id, external_id, library_external_id, parent_external_id, kind, title, year, path, provider_ids, runtime_seconds, date_created, match_confidence, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer itemStmt.Close()
	for _, item := range snapshot.Items {
		item.ServerID = defaultString(item.ServerID, serverID)
		if strings.TrimSpace(item.ExternalID) == "" {
			continue
		}
		if item.ProviderIDs == nil {
			item.ProviderIDs = map[string]string{}
		}
		providerIDs, err := json.Marshal(item.ProviderIDs)
		if err != nil {
			return err
		}
		if item.UpdatedAt.IsZero() {
			item.UpdatedAt = now
		}
		if _, err := itemStmt.Exec(
			item.ServerID,
			strings.TrimSpace(item.ExternalID),
			strings.TrimSpace(item.LibraryExternalID),
			strings.TrimSpace(item.ParentExternalID),
			strings.TrimSpace(item.Kind),
			strings.TrimSpace(item.Title),
			item.Year,
			strings.TrimSpace(item.Path),
			string(providerIDs),
			item.RuntimeSeconds,
			formatOptionalTime(item.DateCreated),
			clampConfidence(item.MatchConfidence),
			item.UpdatedAt.Format(time.RFC3339Nano),
		); err != nil {
			return err
		}
	}

	fileStmt, err := tx.Prepare(`INSERT INTO media_server_files (
		server_id, item_external_id, path, size_bytes, container, local_path, local_media_file_id, verification, match_confidence
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer fileStmt.Close()
	for _, file := range snapshot.Files {
		file.ServerID = defaultString(file.ServerID, serverID)
		if strings.TrimSpace(file.ItemExternalID) == "" || strings.TrimSpace(file.Path) == "" {
			continue
		}
		if file.Verification == "" {
			file.Verification = "server_reported"
		}
		if _, err := fileStmt.Exec(
			file.ServerID,
			strings.TrimSpace(file.ItemExternalID),
			strings.TrimSpace(file.Path),
			file.SizeBytes,
			strings.TrimSpace(file.Container),
			strings.TrimSpace(file.LocalPath),
			strings.TrimSpace(file.LocalMediaFileID),
			strings.TrimSpace(file.Verification),
			clampConfidence(file.MatchConfidence),
		); err != nil {
			return err
		}
	}

	rollupStmt, err := tx.Prepare(`INSERT INTO media_activity_rollups (
		server_id, item_external_id, play_count, unique_users, watched_users, favorite_count, last_played_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer rollupStmt.Close()
	for _, rollup := range snapshot.Rollups {
		rollup.ServerID = defaultString(rollup.ServerID, serverID)
		if strings.TrimSpace(rollup.ItemExternalID) == "" {
			continue
		}
		if rollup.UpdatedAt.IsZero() {
			rollup.UpdatedAt = now
		}
		if _, err := rollupStmt.Exec(
			rollup.ServerID,
			strings.TrimSpace(rollup.ItemExternalID),
			rollup.PlayCount,
			rollup.UniqueUsers,
			rollup.WatchedUsers,
			rollup.FavoriteCount,
			formatOptionalTime(rollup.LastPlayedAt),
			rollup.UpdatedAt.Format(time.RFC3339Nano),
		); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`INSERT INTO media_sync_jobs (
		id, server_id, status, items_imported, rollups_imported, unmapped_items, cursor, error, started_at, completed_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.Job.ID,
		serverID,
		snapshot.Job.Status,
		snapshot.Job.ItemsImported,
		snapshot.Job.RollupsImported,
		snapshot.Job.UnmappedItems,
		snapshot.Job.Cursor,
		snapshot.Job.Error,
		snapshot.Job.StartedAt.UTC().Format(time.RFC3339Nano),
		formatOptionalTime(snapshot.Job.CompletedAt),
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (store *Store) ListMediaServerItems(filter MediaServerItemFilter) ([]MediaServerItem, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	query := `SELECT server_id, external_id, library_external_id, parent_external_id, kind, title, year, path, provider_ids, runtime_seconds, date_created, match_confidence, updated_at
		FROM media_server_items`
	args := []any{}
	conditions := []string{}
	if strings.TrimSpace(filter.ServerID) != "" {
		conditions = append(conditions, "server_id = ?")
		args = append(args, strings.TrimSpace(filter.ServerID))
	}
	if filter.UnmappedOnly {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM media_server_files
			WHERE media_server_files.server_id = media_server_items.server_id
			AND media_server_files.item_external_id = media_server_items.external_id
			AND (media_server_files.local_path = '' OR media_server_files.verification = 'unmapped')
		)`)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY title, path"
	rows, err := store.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []MediaServerItem{}
	for rows.Next() {
		var item MediaServerItem
		var providerIDs string
		var dateCreated sql.NullString
		var updatedAt string
		if err := rows.Scan(
			&item.ServerID,
			&item.ExternalID,
			&item.LibraryExternalID,
			&item.ParentExternalID,
			&item.Kind,
			&item.Title,
			&item.Year,
			&item.Path,
			&providerIDs,
			&item.RuntimeSeconds,
			&dateCreated,
			&item.MatchConfidence,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(providerIDs), &item.ProviderIDs); err != nil {
			return nil, err
		}
		if item.ProviderIDs == nil {
			item.ProviderIDs = map[string]string{}
		}
		item.DateCreated = parseSQLTime(dateCreated)
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (store *Store) ListMediaActivityRollups(serverID string) ([]MediaActivityRollup, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	query := `SELECT server_id, item_external_id, play_count, unique_users, watched_users, favorite_count, last_played_at, updated_at
		FROM media_activity_rollups`
	args := []any{}
	if strings.TrimSpace(serverID) != "" {
		query += " WHERE server_id = ?"
		args = append(args, strings.TrimSpace(serverID))
	}
	query += " ORDER BY server_id, item_external_id"
	rows, err := store.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rollups := []MediaActivityRollup{}
	for rows.Next() {
		var rollup MediaActivityRollup
		var lastPlayed sql.NullString
		var updatedAt string
		if err := rows.Scan(&rollup.ServerID, &rollup.ItemExternalID, &rollup.PlayCount, &rollup.UniqueUsers, &rollup.WatchedUsers, &rollup.FavoriteCount, &lastPlayed, &updatedAt); err != nil {
			return nil, err
		}
		rollup.LastPlayedAt = parseSQLTime(lastPlayed)
		rollup.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		rollups = append(rollups, rollup)
	}
	return rollups, rows.Err()
}

func (store *Store) ListActivityRecommendationMedia() ([]recommendations.ActivityMedia, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	rows, err := store.DB.Query(`SELECT
			items.server_id,
			items.external_id,
			items.parent_external_id,
			COALESCE(parent.title, ''),
			COALESCE(libraries.name, ''),
			items.kind,
			items.title,
			COALESCE(NULLIF(files.local_path, ''), files.path),
			files.size_bytes,
			items.date_created,
			rollups.last_played_at,
			COALESCE(rollups.play_count, 0),
			COALESCE(rollups.unique_users, 0),
			COALESCE(rollups.favorite_count, 0),
			files.verification,
			CASE
				WHEN files.match_confidence > 0 AND items.match_confidence > 0 AND files.match_confidence < items.match_confidence THEN files.match_confidence
				WHEN items.match_confidence > 0 THEN items.match_confidence
				ELSE files.match_confidence
			END
		FROM media_server_items AS items
		JOIN media_server_files AS files
			ON files.server_id = items.server_id
			AND files.item_external_id = items.external_id
		LEFT JOIN media_activity_rollups AS rollups
			ON rollups.server_id = items.server_id
			AND rollups.item_external_id = items.external_id
		LEFT JOIN media_server_items AS parent
			ON parent.server_id = items.server_id
			AND parent.external_id = items.parent_external_id
		LEFT JOIN media_server_libraries AS libraries
			ON libraries.server_id = items.server_id
			AND libraries.external_id = items.library_external_id
		ORDER BY items.title, files.path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	media := []recommendations.ActivityMedia{}
	for rows.Next() {
		var item recommendations.ActivityMedia
		var addedAt sql.NullString
		var lastPlayedAt sql.NullString
		if err := rows.Scan(
			&item.ServerID,
			&item.ExternalItemID,
			&item.ParentExternalItemID,
			&item.ParentTitle,
			&item.LibraryName,
			&item.Kind,
			&item.Title,
			&item.Path,
			&item.SizeBytes,
			&addedAt,
			&lastPlayedAt,
			&item.PlayCount,
			&item.UniqueUsers,
			&item.FavoriteCount,
			&item.Verification,
			&item.MatchConfidence,
		); err != nil {
			return nil, err
		}
		item.AddedAt = parseSQLTime(addedAt)
		item.LastPlayedAt = parseSQLTime(lastPlayedAt)
		media = append(media, item)
	}
	return media, rows.Err()
}

func (store *Store) LatestMediaSyncJob(serverID string) (MediaSyncJob, error) {
	if store == nil || store.DB == nil {
		return MediaSyncJob{}, errors.New("nil database store")
	}
	var job MediaSyncJob
	var startedAt string
	var completedAt sql.NullString
	err := store.DB.QueryRow(`SELECT id, server_id, status, items_imported, rollups_imported, unmapped_items, cursor, error, started_at, completed_at
		FROM media_sync_jobs
		WHERE server_id = ?
		ORDER BY started_at DESC
		LIMIT 1`, strings.TrimSpace(serverID)).Scan(
		&job.ID,
		&job.ServerID,
		&job.Status,
		&job.ItemsImported,
		&job.RollupsImported,
		&job.UnmappedItems,
		&job.Cursor,
		&job.Error,
		&startedAt,
		&completedAt,
	)
	if err != nil {
		return MediaSyncJob{}, err
	}
	job.StartedAt, _ = time.Parse(time.RFC3339Nano, startedAt)
	job.CompletedAt = parseSQLTime(completedAt)
	return job, nil
}

func (store *Store) CreateJob(input JobInput) (Job, error) {
	if store == nil || store.DB == nil {
		return Job{}, errors.New("nil database store")
	}
	kind := normalizeJobKind(input.Kind)
	if !knownJobKind(kind) {
		return Job{}, errors.New("unknown job kind")
	}
	status := normalizeJobStatus(input.Status)
	if status == "" {
		status = "queued"
	}
	if !knownJobStatus(status) {
		return Job{}, errors.New("unknown job status")
	}
	now := time.Now().UTC()
	job := Job{
		ID:           randomID("job"),
		Kind:         kind,
		TargetID:     strings.TrimSpace(input.TargetID),
		Status:       status,
		Phase:        strings.TrimSpace(input.Phase),
		Message:      strings.TrimSpace(input.Message),
		CurrentLabel: strings.TrimSpace(input.CurrentLabel),
		Total:        maxInt(input.Total, 0),
		StartedAt:    now,
		UpdatedAt:    now,
	}
	if job.Phase == "" {
		job.Phase = status
	}
	if job.Message == "" {
		job.Message = "Job queued"
	}
	_, err := store.DB.Exec(`INSERT INTO jobs (
		id, kind, target_id, status, phase, message, current_label, processed, total,
		items_imported, rollups_imported, unmapped_items, error, started_at, updated_at, completed_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID,
		job.Kind,
		job.TargetID,
		job.Status,
		job.Phase,
		job.Message,
		job.CurrentLabel,
		job.Processed,
		job.Total,
		job.ItemsImported,
		job.RollupsImported,
		job.UnmappedItems,
		job.Error,
		job.StartedAt.Format(time.RFC3339Nano),
		job.UpdatedAt.Format(time.RFC3339Nano),
		formatOptionalTime(job.CompletedAt),
	)
	if err != nil {
		return Job{}, err
	}
	return job, nil
}

func (store *Store) UpdateJob(id string, update JobUpdate) (Job, error) {
	if store == nil || store.DB == nil {
		return Job{}, errors.New("nil database store")
	}
	job, err := store.GetJob(id)
	if err != nil {
		return Job{}, err
	}
	if strings.TrimSpace(update.Status) != "" {
		status := normalizeJobStatus(update.Status)
		if !knownJobStatus(status) {
			return Job{}, errors.New("unknown job status")
		}
		job.Status = status
	}
	if strings.TrimSpace(update.Phase) != "" {
		job.Phase = strings.TrimSpace(update.Phase)
	}
	if strings.TrimSpace(update.Message) != "" {
		job.Message = strings.TrimSpace(update.Message)
	}
	if strings.TrimSpace(update.CurrentLabel) != "" {
		job.CurrentLabel = strings.TrimSpace(update.CurrentLabel)
	}
	if update.Processed != nil {
		job.Processed = maxInt(*update.Processed, 0)
	}
	if update.Total != nil {
		job.Total = maxInt(*update.Total, 0)
	}
	if update.ItemsImported != nil {
		job.ItemsImported = maxInt(*update.ItemsImported, 0)
	}
	if update.RollupsImported != nil {
		job.RollupsImported = maxInt(*update.RollupsImported, 0)
	}
	if update.UnmappedItems != nil {
		job.UnmappedItems = maxInt(*update.UnmappedItems, 0)
	}
	if strings.TrimSpace(update.Error) != "" {
		job.Error = strings.TrimSpace(update.Error)
	}
	if update.Completed || job.Status == "completed" || job.Status == "failed" || job.Status == "canceled" || job.Status == "stale" {
		if job.CompletedAt.IsZero() {
			job.CompletedAt = time.Now().UTC()
		}
	}
	job.UpdatedAt = time.Now().UTC()
	_, err = store.DB.Exec(`UPDATE jobs SET
		status = ?, phase = ?, message = ?, current_label = ?, processed = ?, total = ?,
		items_imported = ?, rollups_imported = ?, unmapped_items = ?, error = ?,
		updated_at = ?, completed_at = ?
		WHERE id = ?`,
		job.Status,
		job.Phase,
		job.Message,
		job.CurrentLabel,
		job.Processed,
		job.Total,
		job.ItemsImported,
		job.RollupsImported,
		job.UnmappedItems,
		job.Error,
		job.UpdatedAt.Format(time.RFC3339Nano),
		formatOptionalTime(job.CompletedAt),
		job.ID,
	)
	if err != nil {
		return Job{}, err
	}
	return job, nil
}

func (store *Store) GetJob(id string) (Job, error) {
	if store == nil || store.DB == nil {
		return Job{}, errors.New("nil database store")
	}
	return store.jobFromQuery(`SELECT id, kind, target_id, status, phase, message, current_label,
		processed, total, items_imported, rollups_imported, unmapped_items, error, started_at, updated_at, completed_at
		FROM jobs WHERE id = ?`, strings.TrimSpace(id))
}

func (store *Store) LatestJob(kind string, targetID string) (Job, error) {
	if store == nil || store.DB == nil {
		return Job{}, errors.New("nil database store")
	}
	return store.jobFromQuery(`SELECT id, kind, target_id, status, phase, message, current_label,
		processed, total, items_imported, rollups_imported, unmapped_items, error, started_at, updated_at, completed_at
		FROM jobs WHERE kind = ? AND target_id = ? ORDER BY started_at DESC LIMIT 1`, normalizeJobKind(kind), strings.TrimSpace(targetID))
}

func (store *Store) ListJobs(filter JobFilter) ([]Job, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	conditions := []string{}
	args := []any{}
	if kind := normalizeJobKind(filter.Kind); kind != "" {
		conditions = append(conditions, "kind = ?")
		args = append(args, kind)
	}
	if strings.TrimSpace(filter.TargetID) != "" {
		conditions = append(conditions, "target_id = ?")
		args = append(args, strings.TrimSpace(filter.TargetID))
	}
	if status := normalizeJobStatus(filter.Status); status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}
	if filter.Active {
		conditions = append(conditions, "status IN ('queued', 'running')")
	}
	query := `SELECT id, kind, target_id, status, phase, message, current_label,
		processed, total, items_imported, rollups_imported, unmapped_items, error, started_at, updated_at, completed_at
		FROM jobs`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	query += " ORDER BY started_at DESC LIMIT ?"
	args = append(args, limit)
	rows, err := store.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []Job{}
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (store *Store) MarkStaleJobs(cutoff time.Time) (int, error) {
	if store == nil || store.DB == nil {
		return 0, errors.New("nil database store")
	}
	if cutoff.IsZero() {
		return 0, errors.New("stale cutoff is required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := store.DB.Exec(`UPDATE jobs
		SET status = 'stale',
			phase = 'stale',
			message = 'Marked stale after no progress',
			error = 'job became stale before completion',
			updated_at = ?,
			completed_at = COALESCE(completed_at, ?)
		WHERE status IN ('queued', 'running')
		AND updated_at < ?`,
		now,
		now,
		cutoff.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(affected), nil
}

func (store *Store) AddJobEvent(input JobEventInput) (JobEvent, error) {
	if store == nil || store.DB == nil {
		return JobEvent{}, errors.New("nil database store")
	}
	input.JobID = strings.TrimSpace(input.JobID)
	if input.JobID == "" {
		return JobEvent{}, errors.New("job id is required")
	}
	event := JobEvent{
		ID:           randomID("evt"),
		JobID:        input.JobID,
		Level:        normalizeJobEventLevel(input.Level),
		Phase:        strings.TrimSpace(input.Phase),
		Message:      strings.TrimSpace(input.Message),
		CurrentLabel: strings.TrimSpace(input.CurrentLabel),
		Processed:    maxInt(input.Processed, 0),
		Total:        maxInt(input.Total, 0),
		CreatedAt:    time.Now().UTC(),
	}
	if event.Message == "" {
		return JobEvent{}, errors.New("job event message is required")
	}
	_, err := store.DB.Exec(`INSERT INTO job_events (
		id, job_id, level, phase, message, current_label, processed, total, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.JobID,
		event.Level,
		event.Phase,
		event.Message,
		event.CurrentLabel,
		event.Processed,
		event.Total,
		event.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return JobEvent{}, err
	}
	return event, nil
}

func (store *Store) ListJobEvents(jobID string, limit int) ([]JobEvent, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := store.DB.Query(`SELECT id, job_id, level, phase, message, current_label, processed, total, created_at
		FROM job_events WHERE job_id = ? ORDER BY created_at DESC LIMIT ?`, strings.TrimSpace(jobID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []JobEvent{}
	for rows.Next() {
		var event JobEvent
		var createdAt string
		if err := rows.Scan(&event.ID, &event.JobID, &event.Level, &event.Phase, &event.Message, &event.CurrentLabel, &event.Processed, &event.Total, &createdAt); err != nil {
			return nil, err
		}
		event.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (store *Store) UpsertPathMapping(mapping PathMapping) (PathMapping, error) {
	if store == nil || store.DB == nil {
		return PathMapping{}, errors.New("nil database store")
	}
	mapping.ID = strings.TrimSpace(mapping.ID)
	mapping.ServerID = strings.TrimSpace(mapping.ServerID)
	mapping.ServerPathPrefix = strings.TrimRight(strings.TrimSpace(mapping.ServerPathPrefix), "/")
	mapping.LocalPathPrefix = strings.TrimRight(strings.TrimSpace(mapping.LocalPathPrefix), "/")
	if mapping.ServerPathPrefix == "" || mapping.LocalPathPrefix == "" {
		return PathMapping{}, errors.New("server and local path prefixes are required")
	}
	now := time.Now().UTC()
	if mapping.ID == "" {
		mapping.ID = randomID("map")
	}
	if mapping.CreatedAt.IsZero() {
		mapping.CreatedAt = now
	}
	mapping.UpdatedAt = now
	_, err := store.DB.Exec(`INSERT INTO integration_path_mappings (
		id, server_id, server_path_prefix, local_path_prefix, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		server_id = excluded.server_id,
		server_path_prefix = excluded.server_path_prefix,
		local_path_prefix = excluded.local_path_prefix,
		updated_at = excluded.updated_at`,
		mapping.ID,
		mapping.ServerID,
		mapping.ServerPathPrefix,
		mapping.LocalPathPrefix,
		mapping.CreatedAt.Format(time.RFC3339Nano),
		mapping.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return PathMapping{}, err
	}
	return mapping, nil
}

func (store *Store) ListPathMappings() ([]PathMapping, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	rows, err := store.DB.Query(`SELECT id, server_id, server_path_prefix, local_path_prefix, created_at, updated_at
		FROM integration_path_mappings
		ORDER BY server_id, server_path_prefix`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mappings := []PathMapping{}
	for rows.Next() {
		var mapping PathMapping
		var createdAt string
		var updatedAt string
		if err := rows.Scan(&mapping.ID, &mapping.ServerID, &mapping.ServerPathPrefix, &mapping.LocalPathPrefix, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		mapping.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		mapping.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		mappings = append(mappings, mapping)
	}
	return mappings, rows.Err()
}

func (store *Store) GetPathMapping(id string) (PathMapping, error) {
	if store == nil || store.DB == nil {
		return PathMapping{}, errors.New("nil database store")
	}
	var mapping PathMapping
	var createdAt string
	var updatedAt string
	err := store.DB.QueryRow(`SELECT id, server_id, server_path_prefix, local_path_prefix, created_at, updated_at
		FROM integration_path_mappings
		WHERE id = ?`, strings.TrimSpace(id)).Scan(
		&mapping.ID,
		&mapping.ServerID,
		&mapping.ServerPathPrefix,
		&mapping.LocalPathPrefix,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return PathMapping{}, err
	}
	mapping.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	mapping.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return mapping, nil
}

func (store *Store) VerifyPathMapping(id string) (PathMappingVerification, error) {
	if store == nil || store.DB == nil {
		return PathMappingVerification{}, errors.New("nil database store")
	}
	mapping, err := store.GetPathMapping(id)
	if err != nil {
		return PathMappingVerification{}, err
	}
	serverPrefix := strings.TrimRight(strings.TrimSpace(mapping.ServerPathPrefix), "/")
	localPrefix := strings.TrimRight(strings.TrimSpace(mapping.LocalPathPrefix), "/")
	if serverPrefix == "" || localPrefix == "" {
		return PathMappingVerification{}, errors.New("path mapping prefixes are required")
	}

	query := `SELECT server_id, item_external_id, path, size_bytes
		FROM media_server_files
		WHERE (path = ? OR path LIKE ?)`
	args := []any{serverPrefix, serverPrefix + "/%"}
	if strings.TrimSpace(mapping.ServerID) != "" {
		query += " AND server_id = ?"
		args = append(args, strings.TrimSpace(mapping.ServerID))
	}
	query += " ORDER BY server_id, item_external_id, path"
	rows, err := store.DB.Query(query, args...)
	if err != nil {
		return PathMappingVerification{}, err
	}
	defer rows.Close()

	type mappedFile struct {
		serverID       string
		itemExternalID string
		serverPath     string
		localPath      string
		sizeBytes      int64
		verification   string
		confidence     float64
	}
	files := []mappedFile{}
	result := PathMappingVerification{Mapping: mapping, UpdatedAt: time.Now().UTC()}
	for rows.Next() {
		var file mappedFile
		if err := rows.Scan(&file.serverID, &file.itemExternalID, &file.serverPath, &file.sizeBytes); err != nil {
			return PathMappingVerification{}, err
		}
		file.localPath = translateMappedPath(file.serverPath, serverPrefix, localPrefix)
		file.verification = "unmapped"
		file.confidence = 0.4
		result.MatchedFiles++
		if info, err := os.Stat(file.localPath); err == nil && !info.IsDir() {
			result.MappedFiles++
			if file.sizeBytes <= 0 || info.Size() == file.sizeBytes {
				file.verification = "local_verified"
				file.confidence = 0.95
				result.VerifiedFiles++
			} else {
				file.verification = "path_mapped"
				file.confidence = 0.82
			}
		} else {
			result.MissingFiles++
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return PathMappingVerification{}, err
	}

	tx, err := store.DB.Begin()
	if err != nil {
		return PathMappingVerification{}, err
	}
	defer tx.Rollback()
	for _, file := range files {
		if _, err := tx.Exec(`UPDATE media_server_files
			SET local_path = ?, verification = ?, match_confidence = ?
			WHERE server_id = ? AND item_external_id = ? AND path = ?`,
			file.localPath,
			file.verification,
			file.confidence,
			file.serverID,
			file.itemExternalID,
			file.serverPath,
		); err != nil {
			return PathMappingVerification{}, err
		}
		if file.confidence > 0 {
			if _, err := tx.Exec(`UPDATE media_server_items
				SET match_confidence = CASE WHEN match_confidence < ? THEN ? ELSE match_confidence END,
					updated_at = ?
				WHERE server_id = ? AND external_id = ?`,
				file.confidence,
				file.confidence,
				result.UpdatedAt.Format(time.RFC3339Nano),
				file.serverID,
				file.itemExternalID,
			); err != nil {
				return PathMappingVerification{}, err
			}
		}
	}
	if _, err := tx.Exec(`UPDATE integration_path_mappings SET updated_at = ? WHERE id = ?`, result.UpdatedAt.Format(time.RFC3339Nano), mapping.ID); err != nil {
		return PathMappingVerification{}, err
	}
	if err := tx.Commit(); err != nil {
		return PathMappingVerification{}, err
	}
	result.Mapping.UpdatedAt = result.UpdatedAt
	return result, nil
}

func (store *Store) DeletePathMapping(id string) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	_, err := store.DB.Exec(`DELETE FROM integration_path_mappings WHERE id = ?`, strings.TrimSpace(id))
	return err
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

	existingStates, err := recommendationStates(tx)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM recommendations WHERE ignored_at IS NULL AND state NOT IN ('ignored', 'protected')`); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO recommendations (
		id, action, state, title, explanation, space_saved_bytes, confidence, source, affected_paths, destructive,
		ai_rationale, ai_tags, ai_confidence, ai_source,
		server_id, external_item_id, last_played_at, play_count, unique_users, favorite_count, verification, evidence
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, rec := range recs {
		rec = rec.WithDefaults()
		if existingState := existingStates[rec.ID]; existingState == recommendations.StateIgnored || existingState == recommendations.StateProtected {
			continue
		} else if existingState != "" {
			rec.State = existingState
		}
		paths, err := json.Marshal(rec.AffectedPaths)
		if err != nil {
			return err
		}
		tags := rec.AITags
		if tags == nil {
			tags = []string{}
		}
		aiTags, err := json.Marshal(tags)
		if err != nil {
			return err
		}
		evidence := rec.Evidence
		if evidence == nil {
			evidence = map[string]string{}
		}
		evidenceJSON, err := json.Marshal(evidence)
		if err != nil {
			return err
		}
		destructive := 0
		if rec.Destructive {
			destructive = 1
		}
		if _, err := stmt.Exec(
			rec.ID,
			string(rec.Action),
			string(normalizeRecommendationState(rec.State)),
			rec.Title,
			rec.Explanation,
			rec.SpaceSavedBytes,
			rec.Confidence,
			rec.Source,
			string(paths),
			destructive,
			rec.AIRationale,
			string(aiTags),
			rec.AIConfidence,
			rec.AISource,
			rec.ServerID,
			rec.ExternalItemID,
			formatOptionalTime(rec.LastPlayedAt),
			rec.PlayCount,
			rec.UniqueUsers,
			rec.FavoriteCount,
			rec.Verification,
			string(evidenceJSON),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (store *Store) ListRecommendations() ([]recommendations.Recommendation, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	rows, err := store.DB.Query(`SELECT id, action, state, title, explanation, space_saved_bytes, confidence, source, affected_paths, destructive,
			ai_rationale, ai_tags, ai_confidence, ai_source,
			server_id, external_item_id, last_played_at, play_count, unique_users, favorite_count, verification, evidence
		FROM recommendations
		WHERE ignored_at IS NULL
		AND state NOT IN ('ignored', 'protected')
		ORDER BY space_saved_bytes DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recs := []recommendations.Recommendation{}
	for rows.Next() {
		rec, err := scanRecommendation(rows)
		if err != nil {
			return nil, err
		}
		recs = append(recs, rec.WithDefaults())
	}
	return recs, rows.Err()
}

func (store *Store) GetRecommendation(id string) (recommendations.Recommendation, error) {
	if store == nil || store.DB == nil {
		return recommendations.Recommendation{}, errors.New("nil database store")
	}
	return scanRecommendation(store.DB.QueryRow(`SELECT id, action, state, title, explanation, space_saved_bytes, confidence, source, affected_paths, destructive,
			ai_rationale, ai_tags, ai_confidence, ai_source,
			server_id, external_item_id, last_played_at, play_count, unique_users, favorite_count, verification, evidence
		FROM recommendations
		WHERE id = ?`, strings.TrimSpace(id)))
}

func (store *Store) SetRecommendationState(id string, state recommendations.State) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	state = normalizeRecommendationState(state)
	if _, err := store.GetRecommendation(id); err != nil {
		return err
	}
	ignoredAt := sql.NullString{}
	if state == recommendations.StateIgnored {
		ignoredAt = sql.NullString{String: time.Now().UTC().Format(time.RFC3339Nano), Valid: true}
	}
	_, err := store.DB.Exec(`UPDATE recommendations SET state = ?, ignored_at = ? WHERE id = ?`, string(state), ignoredAt, strings.TrimSpace(id))
	return err
}

func (store *Store) IgnoreRecommendation(id string) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	return store.SetRecommendationState(id, recommendations.StateIgnored)
}

func (store *Store) RestoreRecommendation(id string) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	_, err := store.DB.Exec(`UPDATE recommendations SET state = ?, ignored_at = NULL WHERE id = ?`, string(recommendations.StateNew), strings.TrimSpace(id))
	return err
}

func (store *Store) SetProviderCache(provider string, cacheKey string, body string, expiresAt time.Time) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	if provider == "" || cacheKey == "" {
		return errors.New("provider and cache key are required")
	}
	_, err := store.DB.Exec(
		`INSERT INTO provider_cache (provider, cache_key, body, expires_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(provider, cache_key) DO UPDATE SET body = excluded.body, expires_at = excluded.expires_at`,
		provider,
		cacheKey,
		body,
		expiresAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (store *Store) GetProviderCache(provider string, cacheKey string, now time.Time) (string, bool, error) {
	if store == nil || store.DB == nil {
		return "", false, errors.New("nil database store")
	}
	var body string
	var expiresAtRaw string
	err := store.DB.QueryRow(
		`SELECT body, expires_at FROM provider_cache WHERE provider = ? AND cache_key = ?`,
		provider,
		cacheKey,
	).Scan(&body, &expiresAtRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, expiresAtRaw)
	if err != nil {
		return "", false, err
	}
	if !expiresAt.After(now.UTC()) {
		_, _ = store.DB.Exec(`DELETE FROM provider_cache WHERE provider = ? AND cache_key = ?`, provider, cacheKey)
		return "", false, nil
	}
	return body, true, nil
}

func (store *Store) UpsertProviderSetting(input ProviderSettingInput) (ProviderSetting, error) {
	if store == nil || store.DB == nil {
		return ProviderSetting{}, errors.New("nil database store")
	}
	provider := normalizeProviderName(input.Provider)
	if !knownProvider(provider) {
		return ProviderSetting{}, errors.New("unknown provider")
	}

	current := ProviderSetting{}
	var updatedAtRaw string
	err := store.DB.QueryRow(
		`SELECT provider, base_url, api_key, updated_at FROM provider_settings WHERE provider = ?`,
		provider,
	).Scan(&current.Provider, &current.BaseURL, &current.APIKey, &updatedAtRaw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ProviderSetting{}, err
	}

	baseURL := strings.TrimSpace(input.BaseURL)
	if baseURL == "" && !input.ClearBaseURL {
		baseURL = current.BaseURL
	}
	apiKey := current.APIKey
	if input.ClearAPIKey {
		apiKey = ""
	} else if strings.TrimSpace(input.APIKey) != "" {
		apiKey = strings.TrimSpace(input.APIKey)
	}

	updatedAt := time.Now().UTC()
	_, err = store.DB.Exec(
		`INSERT INTO provider_settings (provider, base_url, api_key, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(provider) DO UPDATE SET
			base_url = excluded.base_url,
			api_key = excluded.api_key,
			updated_at = excluded.updated_at`,
		provider,
		baseURL,
		apiKey,
		updatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return ProviderSetting{}, err
	}
	return redactedProviderSetting(provider, baseURL, apiKey, updatedAt), nil
}

func (store *Store) ListProviderSettings() ([]ProviderSetting, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	rows, err := store.DB.Query(`SELECT provider, base_url, api_key, updated_at FROM provider_settings ORDER BY provider`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := []ProviderSetting{}
	for rows.Next() {
		var setting ProviderSetting
		var updatedAtRaw string
		if err := rows.Scan(&setting.Provider, &setting.BaseURL, &setting.APIKey, &updatedAtRaw); err != nil {
			return nil, err
		}
		setting.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtRaw)
		settings = append(settings, redactedProviderSetting(setting.Provider, setting.BaseURL, setting.APIKey, setting.UpdatedAt))
	}
	return settings, rows.Err()
}

func (store *Store) ListProviderSettingSecrets() ([]ProviderSetting, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	rows, err := store.DB.Query(`SELECT provider, base_url, api_key, updated_at FROM provider_settings ORDER BY provider`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := []ProviderSetting{}
	for rows.Next() {
		var setting ProviderSetting
		var updatedAtRaw string
		if err := rows.Scan(&setting.Provider, &setting.BaseURL, &setting.APIKey, &updatedAtRaw); err != nil {
			return nil, err
		}
		setting.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtRaw)
		setting.APIKeyConfigured = strings.TrimSpace(setting.APIKey) != ""
		setting.APIKeyLast4 = last4(setting.APIKey)
		settings = append(settings, setting)
	}
	return settings, rows.Err()
}

func (store *Store) UpsertIntegrationSetting(input IntegrationSettingInput) (IntegrationSetting, error) {
	if store == nil || store.DB == nil {
		return IntegrationSetting{}, errors.New("nil database store")
	}
	integration := normalizeIntegrationName(input.Integration)
	if !knownIntegration(integration) {
		return IntegrationSetting{}, errors.New("unknown integration")
	}

	current := IntegrationSetting{AutoSyncEnabled: true, AutoSyncIntervalMinutes: defaultAutoSyncIntervalMinutes()}
	var updatedAtRaw string
	err := store.DB.QueryRow(
		`SELECT integration, base_url, api_key, auto_sync_enabled, auto_sync_interval_minutes, updated_at FROM integration_settings WHERE integration = ?`,
		integration,
	).Scan(&current.Integration, &current.BaseURL, &current.APIKey, &current.AutoSyncEnabled, &current.AutoSyncIntervalMinutes, &updatedAtRaw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return IntegrationSetting{}, err
	}

	baseURL := strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	if baseURL == "" && !input.ClearBaseURL {
		baseURL = current.BaseURL
	}
	apiKey := current.APIKey
	if input.ClearAPIKey {
		apiKey = ""
	} else if strings.TrimSpace(input.APIKey) != "" {
		apiKey = strings.TrimSpace(input.APIKey)
	}
	autoSyncEnabled := current.AutoSyncEnabled
	if input.AutoSyncEnabled != nil {
		autoSyncEnabled = *input.AutoSyncEnabled
	}
	autoSyncInterval := current.AutoSyncIntervalMinutes
	if autoSyncInterval <= 0 {
		autoSyncInterval = defaultAutoSyncIntervalMinutes()
	}
	if input.AutoSyncIntervalMinutes > 0 {
		autoSyncInterval = normalizeAutoSyncIntervalMinutes(input.AutoSyncIntervalMinutes)
	}

	updatedAt := time.Now().UTC()
	_, err = store.DB.Exec(
		`INSERT INTO integration_settings (integration, base_url, api_key, auto_sync_enabled, auto_sync_interval_minutes, updated_at) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(integration) DO UPDATE SET
			base_url = excluded.base_url,
			api_key = excluded.api_key,
			auto_sync_enabled = excluded.auto_sync_enabled,
			auto_sync_interval_minutes = excluded.auto_sync_interval_minutes,
			updated_at = excluded.updated_at`,
		integration,
		baseURL,
		apiKey,
		boolInt(autoSyncEnabled),
		autoSyncInterval,
		updatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return IntegrationSetting{}, err
	}
	return redactedIntegrationSetting(integration, baseURL, apiKey, autoSyncEnabled, autoSyncInterval, updatedAt), nil
}

func (store *Store) ListIntegrationSettings() ([]IntegrationSetting, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	rows, err := store.DB.Query(`SELECT integration, base_url, api_key, auto_sync_enabled, auto_sync_interval_minutes, updated_at FROM integration_settings ORDER BY integration`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := []IntegrationSetting{}
	for rows.Next() {
		var setting IntegrationSetting
		var updatedAtRaw string
		if err := rows.Scan(&setting.Integration, &setting.BaseURL, &setting.APIKey, &setting.AutoSyncEnabled, &setting.AutoSyncIntervalMinutes, &updatedAtRaw); err != nil {
			return nil, err
		}
		setting.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtRaw)
		settings = append(settings, redactedIntegrationSetting(setting.Integration, setting.BaseURL, setting.APIKey, setting.AutoSyncEnabled, normalizeAutoSyncIntervalMinutes(setting.AutoSyncIntervalMinutes), setting.UpdatedAt))
	}
	return settings, rows.Err()
}

func (store *Store) ListIntegrationSettingSecrets() ([]IntegrationSetting, error) {
	if store == nil || store.DB == nil {
		return nil, errors.New("nil database store")
	}
	rows, err := store.DB.Query(`SELECT integration, base_url, api_key, auto_sync_enabled, auto_sync_interval_minutes, updated_at FROM integration_settings ORDER BY integration`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := []IntegrationSetting{}
	for rows.Next() {
		var setting IntegrationSetting
		var updatedAtRaw string
		if err := rows.Scan(&setting.Integration, &setting.BaseURL, &setting.APIKey, &setting.AutoSyncEnabled, &setting.AutoSyncIntervalMinutes, &updatedAtRaw); err != nil {
			return nil, err
		}
		setting.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtRaw)
		setting.APIKeyConfigured = strings.TrimSpace(setting.APIKey) != ""
		setting.APIKeyLast4 = last4(setting.APIKey)
		setting.AutoSyncIntervalMinutes = normalizeAutoSyncIntervalMinutes(setting.AutoSyncIntervalMinutes)
		settings = append(settings, setting)
	}
	return settings, rows.Err()
}

func (store *Store) UpsertCatalogCorrection(mediaFileID string, input CatalogCorrectionInput) (CatalogCorrection, error) {
	if store == nil || store.DB == nil {
		return CatalogCorrection{}, errors.New("nil database store")
	}
	mediaFileID = strings.TrimSpace(mediaFileID)
	if mediaFileID == "" {
		return CatalogCorrection{}, errors.New("media file id is required")
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return CatalogCorrection{}, errors.New("title is required")
	}
	if !knownCatalogKind(input.Kind) {
		return CatalogCorrection{}, errors.New("unsupported catalog kind")
	}
	var currentKey string
	if err := store.DB.QueryRow(`SELECT canonical_key FROM media_files WHERE id = ?`, mediaFileID).Scan(&currentKey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CatalogCorrection{}, errors.New("media file not found")
		}
		return CatalogCorrection{}, err
	}
	confidence := input.Confidence
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}
	canonicalKey := strings.TrimSpace(input.CanonicalKey)
	if canonicalKey == "" {
		canonicalKey = correctedCanonicalKey(currentKey, input.Kind, input.Title, input.Year)
	}
	now := time.Now().UTC()
	correction := CatalogCorrection{
		MediaFileID:  mediaFileID,
		Title:        input.Title,
		Kind:         input.Kind,
		Year:         input.Year,
		CanonicalKey: canonicalKey,
		Provider:     normalizeProviderName(input.Provider),
		ProviderID:   strings.TrimSpace(input.ProviderID),
		Confidence:   confidence,
		UpdatedAt:    now,
	}
	_, err := store.DB.Exec(
		`INSERT INTO catalog_corrections (
			media_file_id, title, kind, year, canonical_key, provider, provider_id, confidence, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(media_file_id) DO UPDATE SET
			title = excluded.title,
			kind = excluded.kind,
			year = excluded.year,
			canonical_key = excluded.canonical_key,
			provider = excluded.provider,
			provider_id = excluded.provider_id,
			confidence = excluded.confidence,
			updated_at = excluded.updated_at`,
		correction.MediaFileID,
		correction.Title,
		string(correction.Kind),
		correction.Year,
		correction.CanonicalKey,
		correction.Provider,
		correction.ProviderID,
		correction.Confidence,
		correction.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return CatalogCorrection{}, err
	}
	return correction, nil
}

func (store *Store) ClearCatalogCorrection(mediaFileID string) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	_, err := store.DB.Exec(`DELETE FROM catalog_corrections WHERE media_file_id = ?`, strings.TrimSpace(mediaFileID))
	return err
}

func redactedProviderSetting(provider string, baseURL string, apiKey string, updatedAt time.Time) ProviderSetting {
	return ProviderSetting{
		Provider:         provider,
		BaseURL:          baseURL,
		APIKeyConfigured: strings.TrimSpace(apiKey) != "",
		APIKeyLast4:      last4(apiKey),
		UpdatedAt:        updatedAt,
	}
}

func redactedIntegrationSetting(integration string, baseURL string, apiKey string, autoSyncEnabled bool, autoSyncIntervalMinutes int, updatedAt time.Time) IntegrationSetting {
	return IntegrationSetting{
		Integration:             integration,
		BaseURL:                 baseURL,
		APIKeyConfigured:        strings.TrimSpace(apiKey) != "",
		APIKeyLast4:             last4(apiKey),
		AutoSyncEnabled:         autoSyncEnabled,
		AutoSyncIntervalMinutes: normalizeAutoSyncIntervalMinutes(autoSyncIntervalMinutes),
		UpdatedAt:               updatedAt,
	}
}

func defaultAutoSyncIntervalMinutes() int {
	return 360
}

func normalizeAutoSyncIntervalMinutes(minutes int) int {
	if minutes <= 0 {
		return defaultAutoSyncIntervalMinutes()
	}
	if minutes < 15 {
		return 15
	}
	if minutes > 10080 {
		return 10080
	}
	return minutes
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func last4(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 4 {
		return value
	}
	return value[len(value)-4:]
}

func normalizeIntegrationName(integration string) string {
	return strings.ToLower(strings.TrimSpace(integration))
}

func knownIntegration(integration string) bool {
	switch normalizeIntegrationName(integration) {
	case "jellyfin", "plex", "emby":
		return true
	default:
		return false
	}
}

func normalizeJobKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}

func knownJobKind(kind string) bool {
	switch normalizeJobKind(kind) {
	case "filesystem_scan", "jellyfin_sync", "plex_sync", "emby_sync":
		return true
	default:
		return false
	}
}

func normalizeJobStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func knownJobStatus(status string) bool {
	switch normalizeJobStatus(status) {
	case "queued", "running", "completed", "failed", "canceled", "stale":
		return true
	default:
		return false
	}
}

func normalizeRecommendationState(state recommendations.State) recommendations.State {
	switch recommendations.State(strings.ToLower(strings.TrimSpace(string(state)))) {
	case recommendations.StateReviewing:
		return recommendations.StateReviewing
	case recommendations.StateIgnored:
		return recommendations.StateIgnored
	case recommendations.StateProtected:
		return recommendations.StateProtected
	case recommendations.StateAcceptedForManualAction:
		return recommendations.StateAcceptedForManualAction
	default:
		return recommendations.StateNew
	}
}

func translateMappedPath(path string, serverPrefix string, localPrefix string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	serverPrefix = filepath.Clean(strings.TrimSpace(serverPrefix))
	localPrefix = filepath.Clean(strings.TrimSpace(localPrefix))
	if path == serverPrefix {
		return localPrefix
	}
	suffix := strings.TrimPrefix(path, serverPrefix)
	suffix = strings.TrimLeft(suffix, string(os.PathSeparator)+"/")
	if suffix == "" {
		return localPrefix
	}
	return filepath.Join(localPrefix, filepath.FromSlash(suffix))
}

func normalizeJobEventLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug", "warning", "error":
		return strings.ToLower(strings.TrimSpace(level))
	default:
		return "info"
	}
}

func normalizeProviderName(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func knownProvider(provider string) bool {
	switch normalizeProviderName(provider) {
	case "tmdb", "anilist", "thetvdb", "opensubtitles", "local-sidecar":
		return true
	default:
		return false
	}
}

func knownCatalogKind(kind catalog.Kind) bool {
	switch kind {
	case catalog.KindMovie, catalog.KindSeries, catalog.KindAnime, catalog.KindUnknown:
		return true
	default:
		return false
	}
}

func correctedCanonicalKey(current string, kind catalog.Kind, title string, year int) string {
	base := string(kind) + ":" + slugTitle(title)
	if kind == catalog.KindMovie && year > 0 {
		return base + ":" + strconv.Itoa(year)
	}
	parts := strings.Split(current, ":")
	if len(parts) > 2 {
		tail := parts[len(parts)-1]
		if kind == catalog.KindSeries && strings.HasPrefix(tail, "s") {
			return base + ":" + tail
		}
		if kind == catalog.KindAnime && strings.HasPrefix(tail, "e") {
			return base + ":" + tail
		}
	}
	if year > 0 {
		return base + ":" + strconv.Itoa(year)
	}
	return base
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

type jobScanner interface {
	Scan(dest ...any) error
}

type recommendationScanner interface {
	Scan(dest ...any) error
}

func recommendationStates(tx *sql.Tx) (map[string]recommendations.State, error) {
	rows, err := tx.Query(`SELECT id, state FROM recommendations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	states := map[string]recommendations.State{}
	for rows.Next() {
		var id string
		var state string
		if err := rows.Scan(&id, &state); err != nil {
			return nil, err
		}
		states[id] = normalizeRecommendationState(recommendations.State(state))
	}
	return states, rows.Err()
}

func scanRecommendation(scanner recommendationScanner) (recommendations.Recommendation, error) {
	var rec recommendations.Recommendation
	var action string
	var state string
	var paths string
	var aiTags string
	var destructive int
	var lastPlayed sql.NullString
	var evidence string
	if err := scanner.Scan(
		&rec.ID,
		&action,
		&state,
		&rec.Title,
		&rec.Explanation,
		&rec.SpaceSavedBytes,
		&rec.Confidence,
		&rec.Source,
		&paths,
		&destructive,
		&rec.AIRationale,
		&aiTags,
		&rec.AIConfidence,
		&rec.AISource,
		&rec.ServerID,
		&rec.ExternalItemID,
		&lastPlayed,
		&rec.PlayCount,
		&rec.UniqueUsers,
		&rec.FavoriteCount,
		&rec.Verification,
		&evidence,
	); err != nil {
		return recommendations.Recommendation{}, err
	}
	rec.Action = recommendations.Action(action)
	rec.State = normalizeRecommendationState(recommendations.State(state))
	rec.Destructive = destructive == 1
	rec.LastPlayedAt = parseSQLTime(lastPlayed)
	if paths == "" {
		paths = "[]"
	}
	if err := json.Unmarshal([]byte(paths), &rec.AffectedPaths); err != nil {
		return recommendations.Recommendation{}, err
	}
	if aiTags == "" {
		aiTags = "[]"
	}
	if err := json.Unmarshal([]byte(aiTags), &rec.AITags); err != nil {
		return recommendations.Recommendation{}, err
	}
	if evidence == "" {
		evidence = "{}"
	}
	if err := json.Unmarshal([]byte(evidence), &rec.Evidence); err != nil {
		return recommendations.Recommendation{}, err
	}
	return rec.WithDefaults(), nil
}

func (store *Store) jobFromQuery(query string, args ...any) (Job, error) {
	row := store.DB.QueryRow(query, args...)
	return scanJob(row)
}

func scanJob(scanner jobScanner) (Job, error) {
	var job Job
	var startedAt string
	var updatedAt string
	var completedAt sql.NullString
	err := scanner.Scan(
		&job.ID,
		&job.Kind,
		&job.TargetID,
		&job.Status,
		&job.Phase,
		&job.Message,
		&job.CurrentLabel,
		&job.Processed,
		&job.Total,
		&job.ItemsImported,
		&job.RollupsImported,
		&job.UnmappedItems,
		&job.Error,
		&startedAt,
		&updatedAt,
		&completedAt,
	)
	if err != nil {
		return Job{}, err
	}
	job.StartedAt, _ = time.Parse(time.RFC3339Nano, startedAt)
	job.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	job.CompletedAt = parseSQLTime(completedAt)
	return job, nil
}

func maxInt(value int, floor int) int {
	if value < floor {
		return floor
	}
	return value
}

func formatOptionalTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func parseSQLTime(value sql.NullString) time.Time {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return time.Time{}
	}
	parsed, _ := time.Parse(time.RFC3339Nano, value.String)
	return parsed
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func slugTitle(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}
