package integrations

import (
	"net/http"
	"strings"
	"time"
)

type Target struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CheckedAt   time.Time `json:"checkedAt"`
}

type Options struct {
	JellyfinURL string
	JellyfinKey string
	PlexURL     string
	PlexToken   string
	EmbyURL     string
	EmbyKey     string
}

func Defaults() []Target {
	return DefaultsWithOptions(Options{})
}

func DefaultsWithOptions(options Options) []Target {
	now := time.Now().UTC()
	return []Target{
		{ID: "jellyfin", Name: "Jellyfin", Kind: "media_server", Status: checkServer(options.JellyfinURL, options.JellyfinKey, "jellyfin"), Description: "Sync metadata, artwork, collections, and library refresh events.", CheckedAt: now},
		{ID: "plex", Name: "Plex", Kind: "media_server", Status: checkServer(options.PlexURL, options.PlexToken, "plex"), Description: "Sync metadata, artwork, collections, and library refresh events.", CheckedAt: now},
		{ID: "emby", Name: "Emby", Kind: "media_server", Status: checkServer(options.EmbyURL, options.EmbyKey, "emby"), Description: "Sync metadata, artwork, collections, and library refresh events.", CheckedAt: now},
		{ID: "ollama", Name: "Ollama", Kind: "local_ai", Status: "optional", Description: "Local-only advisory AI for matching, tags, and cleanup rationales.", CheckedAt: now},
	}
}

func checkServer(baseURL string, token string, kind string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	token = strings.TrimSpace(token)
	if baseURL == "" || token == "" {
		return "not_configured"
	}
	path := "/System/Info"
	if kind == "plex" {
		path = "/identity?X-Plex-Token=" + token
	}
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return "invalid_config"
	}
	if kind != "plex" {
		req.Header.Set("X-Emby-Token", token)
	}
	client := &http.Client{Timeout: 4 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "unavailable"
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode >= 200 && res.StatusCode < 300:
		return "configured"
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return "invalid_credentials"
	default:
		return "degraded"
	}
}
