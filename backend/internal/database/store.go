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

func (store *Store) IgnoreRecommendation(id string) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	_, err := store.DB.Exec(`UPDATE recommendations SET ignored_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

func (store *Store) RestoreRecommendation(id string) error {
	if store == nil || store.DB == nil {
		return errors.New("nil database store")
	}
	_, err := store.DB.Exec(`UPDATE recommendations SET ignored_at = NULL WHERE id = ?`, id)
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

func last4(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 4 {
		return value
	}
	return value[len(value)-4:]
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
