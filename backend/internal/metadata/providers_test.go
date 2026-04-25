package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPProviderHealthReportsConfiguredAndRateLimitedStates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	provider := HTTPProvider{
		ProviderName: "tmdb",
		Text:         "Metadata provided by TMDb.",
		BaseURL:      server.URL,
		APIKey:       "secret",
		CheckPath:    "/health",
	}

	health := provider.Health(context.Background())
	if health.Status != "rate_limited" {
		t.Fatalf("status = %q, want rate_limited", health.Status)
	}
}

func TestHTTPProviderHealthReportsNotConfiguredWithoutKey(t *testing.T) {
	provider := HTTPProvider{
		ProviderName: "tmdb",
		Text:         "Metadata provided by TMDb.",
		BaseURL:      "https://api.themoviedb.org",
		CheckPath:    "/3/configuration",
	}

	health := provider.Health(context.Background())
	if health.Status != "not_configured" {
		t.Fatalf("status = %q, want not_configured", health.Status)
	}
}
