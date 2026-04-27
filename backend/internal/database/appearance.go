package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	appearanceSettingsKey = "appearance"
	maxCustomCSSBytes     = 20_000
)

type AppearanceSettings struct {
	Theme     string    `json:"theme"`
	CustomCSS string    `json:"customCss"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

type AppearanceSettingsInput struct {
	Theme     string `json:"theme"`
	CustomCSS string `json:"customCss"`
}

func (store *Store) GetAppearanceSettings() (AppearanceSettings, error) {
	if store == nil || store.DB == nil {
		return AppearanceSettings{}, errors.New("nil database store")
	}
	defaults := defaultAppearanceSettings()
	var raw string
	var updatedAtRaw string
	err := store.DB.QueryRow(`SELECT value, updated_at FROM settings WHERE key = ?`, appearanceSettingsKey).Scan(&raw, &updatedAtRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaults, nil
		}
		return AppearanceSettings{}, err
	}
	var stored AppearanceSettings
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return AppearanceSettings{}, err
	}
	settings, err := normalizeAppearanceSettings(AppearanceSettingsInput{Theme: stored.Theme, CustomCSS: stored.CustomCSS})
	if err != nil {
		return AppearanceSettings{}, err
	}
	settings.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtRaw)
	return settings, nil
}

func (store *Store) UpdateAppearanceSettings(input AppearanceSettingsInput) (AppearanceSettings, error) {
	if store == nil || store.DB == nil {
		return AppearanceSettings{}, errors.New("nil database store")
	}
	settings, err := normalizeAppearanceSettings(input)
	if err != nil {
		return AppearanceSettings{}, err
	}
	settings.UpdatedAt = time.Now().UTC()
	body, err := json.Marshal(settings)
	if err != nil {
		return AppearanceSettings{}, err
	}
	_, err = store.DB.Exec(
		`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		appearanceSettingsKey,
		string(body),
		settings.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return AppearanceSettings{}, err
	}
	return settings, nil
}

func defaultAppearanceSettings() AppearanceSettings {
	return AppearanceSettings{Theme: "system", CustomCSS: ""}
}

func normalizeAppearanceSettings(input AppearanceSettingsInput) (AppearanceSettings, error) {
	theme := strings.ToLower(strings.TrimSpace(input.Theme))
	if theme == "" {
		theme = "system"
	}
	switch theme {
	case "system", "dark", "light":
	default:
		return AppearanceSettings{}, errors.New("appearance theme must be system, dark, or light")
	}
	customCSS := strings.TrimSpace(input.CustomCSS)
	if !utf8.ValidString(customCSS) {
		return AppearanceSettings{}, errors.New("custom css must be valid utf-8")
	}
	if len([]byte(customCSS)) > maxCustomCSSBytes {
		return AppearanceSettings{}, errors.New("custom css must be 20000 bytes or less")
	}
	if containsUnsafeCSS(customCSS) {
		return AppearanceSettings{}, errors.New("custom css cannot include imports, urls, or legacy scriptable css")
	}
	return AppearanceSettings{Theme: theme, CustomCSS: customCSS}, nil
}

func containsUnsafeCSS(css string) bool {
	lower := strings.ToLower(css)
	for _, token := range []string{"@import", "url(", "javascript:", "expression(", "behavior:", "-moz-binding"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	compact := strings.NewReplacer(" ", "", "\n", "", "\r", "", "\t", "", "\f", "").Replace(lower)
	for _, token := range []string{"@import", "url(", "javascript:", "expression(", "behavior:", "-moz-binding"} {
		if strings.Contains(compact, token) {
			return true
		}
	}
	return false
}
