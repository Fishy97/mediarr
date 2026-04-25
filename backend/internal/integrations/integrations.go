package integrations

import (
	"context"
	"errors"
	"net/http"
	"net/url"
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

type RefreshResult struct {
	TargetID    string    `json:"targetId"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	RequestedAt time.Time `json:"requestedAt"`
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

func Refresh(ctx context.Context, options Options, targetID string) (RefreshResult, error) {
	targetID = strings.ToLower(strings.TrimSpace(targetID))
	requestedAt := time.Now().UTC()
	endpoint, token, method, headerName, err := refreshConfig(options, targetID)
	if err != nil {
		return RefreshResult{TargetID: targetID, Status: "failed", Message: err.Error(), RequestedAt: requestedAt}, err
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return RefreshResult{TargetID: targetID, Status: "failed", Message: "invalid refresh endpoint", RequestedAt: requestedAt}, err
	}
	if headerName != "" {
		req.Header.Set(headerName, token)
	}
	res, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		return RefreshResult{TargetID: targetID, Status: "failed", Message: err.Error(), RequestedAt: requestedAt}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		message := "refresh failed with status " + res.Status
		return RefreshResult{TargetID: targetID, Status: "failed", Message: message, RequestedAt: requestedAt}, errors.New(message)
	}
	return RefreshResult{TargetID: targetID, Status: "requested", Message: "library refresh requested", RequestedAt: requestedAt}, nil
}

func refreshConfig(options Options, targetID string) (endpoint string, token string, method string, headerName string, err error) {
	switch targetID {
	case "jellyfin":
		baseURL := strings.TrimRight(strings.TrimSpace(options.JellyfinURL), "/")
		token := strings.TrimSpace(options.JellyfinKey)
		if baseURL == "" || token == "" {
			return "", "", "", "", errors.New("jellyfin is not configured")
		}
		return baseURL + "/Library/Refresh", token, http.MethodPost, "X-Emby-Token", nil
	case "emby":
		baseURL := strings.TrimRight(strings.TrimSpace(options.EmbyURL), "/")
		token := strings.TrimSpace(options.EmbyKey)
		if baseURL == "" || token == "" {
			return "", "", "", "", errors.New("emby is not configured")
		}
		return baseURL + "/Library/Refresh", token, http.MethodPost, "X-Emby-Token", nil
	case "plex":
		baseURL := strings.TrimRight(strings.TrimSpace(options.PlexURL), "/")
		token := strings.TrimSpace(options.PlexToken)
		if baseURL == "" || token == "" {
			return "", "", "", "", errors.New("plex is not configured")
		}
		values := url.Values{}
		values.Set("X-Plex-Token", token)
		return baseURL + "/library/sections/all/refresh?" + values.Encode(), token, http.MethodGet, "", nil
	default:
		return "", "", "", "", errors.New("unknown integration target")
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
