package config

import "testing"

func TestLoadPrefersMediarrEnvironmentNames(t *testing.T) {
	t.Setenv("MEDIARR_ADDR", ":9090")
	t.Setenv("MEDIAAR_ADDR", ":8082")
	t.Setenv("MEDIA_STEWARD_ADDR", ":8081")
	t.Setenv("MEDIARR_ADMIN_TOKEN", "new-token")
	t.Setenv("MEDIARR_AI_MODEL", "qwen3:0.6b")
	t.Setenv("MEDIARR_TMDB_TOKEN", "tmdb-token")
	t.Setenv("MEDIAAR_ADMIN_TOKEN", "old-mediaar-token")
	t.Setenv("MEDIA_STEWARD_ADMIN_TOKEN", "old-token")

	cfg := Load()

	if cfg.Addr != ":9090" {
		t.Fatalf("addr = %q, want :9090", cfg.Addr)
	}
	if cfg.AdminToken != "new-token" {
		t.Fatalf("admin token = %q, want new-token", cfg.AdminToken)
	}
	if cfg.AIModel != "qwen3:0.6b" {
		t.Fatalf("ai model = %q, want qwen3:0.6b", cfg.AIModel)
	}
	if cfg.TMDbToken != "tmdb-token" {
		t.Fatalf("tmdb token = %q, want tmdb-token", cfg.TMDbToken)
	}
}

func TestLoadFallsBackToLegacyMediaarEnvironmentNames(t *testing.T) {
	t.Setenv("MEDIAAR_CONFIG_DIR", "/legacy-mediaar-config")
	t.Setenv("MEDIA_STEWARD_CONFIG_DIR", "/legacy-steward-config")
	t.Setenv("MEDIAAR_MOVIES_DIR", "/legacy-mediaar-movies")
	t.Setenv("MEDIA_STEWARD_MOVIES_DIR", "/legacy-steward-movies")

	cfg := Load()

	if cfg.ConfigDir != "/legacy-mediaar-config" {
		t.Fatalf("config dir = %q, want /legacy-mediaar-config", cfg.ConfigDir)
	}
	if cfg.DefaultLibraries[0].Root != "/legacy-mediaar-movies" {
		t.Fatalf("movies dir = %q, want /legacy-mediaar-movies", cfg.DefaultLibraries[0].Root)
	}
}

func TestLoadFallsBackToLegacyMediaStewardEnvironmentNames(t *testing.T) {
	t.Setenv("MEDIA_STEWARD_CONFIG_DIR", "/legacy-config")
	t.Setenv("MEDIA_STEWARD_MOVIES_DIR", "/legacy-movies")

	cfg := Load()

	if cfg.ConfigDir != "/legacy-config" {
		t.Fatalf("config dir = %q, want /legacy-config", cfg.ConfigDir)
	}
	if cfg.DefaultLibraries[0].Root != "/legacy-movies" {
		t.Fatalf("movies dir = %q, want /legacy-movies", cfg.DefaultLibraries[0].Root)
	}
}
