package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/metadata"
)

func TestProviderSettingsRoutesRedactSecretsAndRefreshHealth(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	providerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/3/configuration" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer secret-token-abcd" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer providerServer.Close()

	server := NewServer(Deps{
		Store:           store,
		ProviderOptions: metadata.Options{TMDbBaseURL: providerServer.URL},
	})

	body := bytes.NewBufferString(`{"apiKey":"secret-token-abcd"}`)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPut, "/api/v1/provider-settings/tmdb", body))
	if res.Code != http.StatusOK {
		t.Fatalf("provider setting update status = %d, want 200: %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/provider-settings", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("provider settings status = %d, want 200", res.Code)
	}
	var settings struct {
		Data []database.ProviderSetting `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.Data) != 1 || !settings.Data[0].APIKeyConfigured || settings.Data[0].APIKeyLast4 != "abcd" {
		t.Fatalf("settings = %#v", settings.Data)
	}
	if settings.Data[0].APIKey != "" {
		t.Fatal("settings API must not return provider secrets")
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("provider health status = %d, want 200", res.Code)
	}
	var health struct {
		Data []metadata.Health `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	for _, row := range health.Data {
		if row.Name == "tmdb" && row.Status == "configured" {
			return
		}
	}
	t.Fatalf("tmdb health was not refreshed after setting update: %#v", health.Data)
}
