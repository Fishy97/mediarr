package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr             string
	ConfigDir        string
	FrontendDir      string
	AdminToken       string
	OllamaURL        string
	AIModel          string
	TMDbToken        string
	TheTVDBAPIKey    string
	OpenSubtitlesKey string
	JellyfinURL      string
	JellyfinAPIKey   string
	PlexURL          string
	PlexToken        string
	EmbyURL          string
	EmbyAPIKey       string
	OversizedBytes   int64
	DefaultLibraries []LibraryConfig
}

type LibraryConfig struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Root string `json:"root"`
}

func Load() Config {
	cfg := Config{
		Addr:             envAny(":8080", "MEDIARR_ADDR", "MEDIAAR_ADDR", "MEDIA_STEWARD_ADDR"),
		ConfigDir:        envAny("/config", "MEDIARR_CONFIG_DIR", "MEDIAAR_CONFIG_DIR", "MEDIA_STEWARD_CONFIG_DIR"),
		FrontendDir:      envAny("./web", "MEDIARR_FRONTEND_DIR", "MEDIAAR_FRONTEND_DIR", "MEDIA_STEWARD_FRONTEND_DIR"),
		AdminToken:       envAny("", "MEDIARR_ADMIN_TOKEN", "MEDIAAR_ADMIN_TOKEN", "MEDIA_STEWARD_ADMIN_TOKEN"),
		OllamaURL:        envAny("http://ollama:11434", "MEDIARR_OLLAMA_URL", "MEDIAAR_OLLAMA_URL", "MEDIA_STEWARD_OLLAMA_URL"),
		AIModel:          envAny("qwen3:0.6b", "MEDIARR_AI_MODEL", "MEDIAAR_AI_MODEL", "MEDIA_STEWARD_AI_MODEL"),
		TMDbToken:        envAny("", "MEDIARR_TMDB_TOKEN", "MEDIAAR_TMDB_TOKEN", "MEDIA_STEWARD_TMDB_TOKEN"),
		TheTVDBAPIKey:    envAny("", "MEDIARR_THETVDB_API_KEY", "MEDIAAR_THETVDB_API_KEY", "MEDIA_STEWARD_THETVDB_API_KEY"),
		OpenSubtitlesKey: envAny("", "MEDIARR_OPENSUBTITLES_API_KEY", "MEDIAAR_OPENSUBTITLES_API_KEY", "MEDIA_STEWARD_OPENSUBTITLES_API_KEY"),
		JellyfinURL:      envAny("", "MEDIARR_JELLYFIN_URL", "MEDIAAR_JELLYFIN_URL", "MEDIA_STEWARD_JELLYFIN_URL"),
		JellyfinAPIKey:   envAny("", "MEDIARR_JELLYFIN_API_KEY", "MEDIAAR_JELLYFIN_API_KEY", "MEDIA_STEWARD_JELLYFIN_API_KEY"),
		PlexURL:          envAny("", "MEDIARR_PLEX_URL", "MEDIAAR_PLEX_URL", "MEDIA_STEWARD_PLEX_URL"),
		PlexToken:        envAny("", "MEDIARR_PLEX_TOKEN", "MEDIAAR_PLEX_TOKEN", "MEDIA_STEWARD_PLEX_TOKEN"),
		EmbyURL:          envAny("", "MEDIARR_EMBY_URL", "MEDIAAR_EMBY_URL", "MEDIA_STEWARD_EMBY_URL"),
		EmbyAPIKey:       envAny("", "MEDIARR_EMBY_API_KEY", "MEDIAAR_EMBY_API_KEY", "MEDIA_STEWARD_EMBY_API_KEY"),
		OversizedBytes:   envInt64Any(60_000_000_000, "MEDIARR_OVERSIZED_BYTES", "MEDIAAR_OVERSIZED_BYTES", "MEDIA_STEWARD_OVERSIZED_BYTES"),
	}
	cfg.DefaultLibraries = []LibraryConfig{
		{ID: "movies", Name: "Movies", Kind: "movies", Root: envAny("/media/movies", "MEDIARR_MOVIES_DIR", "MEDIAAR_MOVIES_DIR", "MEDIA_STEWARD_MOVIES_DIR")},
		{ID: "series", Name: "Series", Kind: "series", Root: envAny("/media/series", "MEDIARR_SERIES_DIR", "MEDIAAR_SERIES_DIR", "MEDIA_STEWARD_SERIES_DIR")},
		{ID: "anime", Name: "Anime", Kind: "anime", Root: envAny("/media/anime", "MEDIARR_ANIME_DIR", "MEDIAAR_ANIME_DIR", "MEDIA_STEWARD_ANIME_DIR")},
	}
	return cfg
}

func envAny(fallback string, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return fallback
}

func envInt64Any(fallback int64, keys ...string) int64 {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return parsed
		}
	}
	return fallback
}
