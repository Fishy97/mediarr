package metadata

import (
	"context"
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
}

func (provider StaticProvider) Name() string {
	return provider.ProviderName
}

func (provider StaticProvider) Attribution() string {
	return provider.Text
}

func (provider StaticProvider) Health(context.Context) Health {
	return Health{
		Name:        provider.ProviderName,
		Status:      "not_configured",
		Attribution: provider.Text,
		RateLimit:   provider.Limit,
		CheckedAt:   time.Now().UTC(),
	}
}

func Defaults() []Provider {
	return []Provider{
		StaticProvider{ProviderName: "tmdb", Text: "Metadata provided by TMDb when configured."},
		StaticProvider{ProviderName: "anilist", Text: "Anime metadata provided by AniList when configured.", Limit: "90 requests/minute nominal; degraded windows may be lower"},
		StaticProvider{ProviderName: "thetvdb", Text: "Metadata provided by TheTVDB. Attribution required when used."},
		StaticProvider{ProviderName: "opensubtitles", Text: "Subtitle metadata provided by OpenSubtitles when configured.", Limit: "provider rate limits apply"},
		StaticProvider{ProviderName: "local-sidecar", Text: "Local NFO, artwork, and sidecar files are read from mounted libraries."},
	}
}
