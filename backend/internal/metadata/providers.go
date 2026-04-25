package metadata

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type Provider interface {
	Name() string
	Attribution() string
	Health(context.Context) Health
}

type Health struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Attribution string    `json:"attribution"`
	RateLimit   string    `json:"rateLimit,omitempty"`
	CheckedAt   time.Time `json:"checkedAt"`
}

type StaticProvider struct {
	ProviderName string
	Text         string
	Limit        string
	Status       string
}

type HTTPProvider struct {
	ProviderName string
	Text         string
	Limit        string
	BaseURL      string
	APIKey       string
	CheckPath    string
	Client       *http.Client
	Anonymous    bool
}

type Options struct {
	TMDbToken           string
	TheTVDBAPIKey       string
	OpenSubtitlesAPIKey string
	TMDbBaseURL         string
	TheTVDBBaseURL      string
	OpenSubtitlesURL    string
}

func (provider StaticProvider) Name() string {
	return provider.ProviderName
}

func (provider StaticProvider) Attribution() string {
	return provider.Text
}

func (provider StaticProvider) Health(context.Context) Health {
	status := provider.Status
	if status == "" {
		status = "not_configured"
	}
	return Health{
		Name:        provider.ProviderName,
		Status:      status,
		Attribution: provider.Text,
		RateLimit:   provider.Limit,
		CheckedAt:   time.Now().UTC(),
	}
}

func (provider HTTPProvider) Name() string {
	return provider.ProviderName
}

func (provider HTTPProvider) Attribution() string {
	return provider.Text
}

func (provider HTTPProvider) Health(ctx context.Context) Health {
	health := Health{
		Name:        provider.ProviderName,
		Status:      "not_configured",
		Attribution: provider.Text,
		RateLimit:   provider.Limit,
		CheckedAt:   time.Now().UTC(),
	}
	if !provider.Anonymous && strings.TrimSpace(provider.APIKey) == "" {
		return health
	}
	baseURL := strings.TrimRight(provider.BaseURL, "/")
	checkPath := provider.CheckPath
	if checkPath == "" {
		checkPath = "/"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+checkPath, nil)
	if err != nil {
		health.Status = "invalid_config"
		return health
	}
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	client := provider.Client
	if client == nil {
		client = &http.Client{Timeout: 6 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		health.Status = "unavailable"
		return health
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode >= 200 && res.StatusCode < 300:
		health.Status = "configured"
	case res.StatusCode == http.StatusTooManyRequests:
		health.Status = "rate_limited"
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		health.Status = "invalid_credentials"
	default:
		health.Status = "degraded"
	}
	return health
}

func Defaults() []Provider {
	return DefaultsWithOptions(Options{})
}

func DefaultsWithOptions(options Options) []Provider {
	tmdbURL := fallback(options.TMDbBaseURL, "https://api.themoviedb.org")
	tvdbURL := fallback(options.TheTVDBBaseURL, "https://api4.thetvdb.com")
	openSubtitlesURL := fallback(options.OpenSubtitlesURL, "https://api.opensubtitles.com")
	return []Provider{
		HTTPProvider{ProviderName: "tmdb", Text: "Metadata provided by TMDb.", BaseURL: tmdbURL, CheckPath: "/3/configuration", APIKey: options.TMDbToken},
		StaticProvider{ProviderName: "anilist", Text: "Anime metadata provided by AniList.", Limit: "90 requests/minute nominal; degraded windows may be lower", Status: "configured"},
		HTTPProvider{ProviderName: "thetvdb", Text: "Metadata provided by TheTVDB. Attribution required when used.", BaseURL: tvdbURL, CheckPath: "/v4/search?query=health&type=series", APIKey: options.TheTVDBAPIKey},
		HTTPProvider{ProviderName: "opensubtitles", Text: "Subtitle metadata provided by OpenSubtitles when configured.", Limit: "provider rate limits apply", BaseURL: openSubtitlesURL, CheckPath: "/api/v1/infos/user", APIKey: options.OpenSubtitlesAPIKey},
		StaticProvider{ProviderName: "local-sidecar", Text: "Local NFO, artwork, and sidecar files are read from mounted libraries.", Status: "configured"},
	}
}

func fallback(value string, defaultValue string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue
	}
	return value
}
