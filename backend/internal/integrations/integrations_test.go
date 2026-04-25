package integrations

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultsWithOptionsChecksConfiguredMediaServers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") == "" && r.URL.Query().Get("X-Plex-Token") == "" {
			t.Fatalf("missing auth on %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	targets := DefaultsWithOptions(Options{
		JellyfinURL: server.URL,
		JellyfinKey: "jellyfin-key",
		PlexURL:     server.URL,
		PlexToken:   "plex-token",
		EmbyURL:     server.URL,
		EmbyKey:     "emby-key",
	})

	for _, target := range targets {
		if target.Kind == "media_server" && target.Status != "configured" {
			t.Fatalf("%s status = %s, want configured", target.ID, target.Status)
		}
	}
}
