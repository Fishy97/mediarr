package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/integrations"
)

func TestIntegrationSettingsRoutesConfigureJellyfinWithoutEnv(t *testing.T) {
	jellyfin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "jellyfin-secret-abcd" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin UI Config"}`))
		case "/Users":
			_, _ = w.Write([]byte(`[{"Id":"user_1","Name":"Alex"}]`))
		case "/Items":
			_, _ = w.Write([]byte(`{"Items":[],"TotalRecordCount":0}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer jellyfin.Close()

	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := NewServer(Deps{Store: store})
	body := bytes.NewBufferString(`{"baseUrl":"` + jellyfin.URL + `","apiKey":"jellyfin-secret-abcd"}`)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPut, "/api/v1/integration-settings/jellyfin", body))
	if res.Code != http.StatusOK {
		t.Fatalf("integration setting update status = %d, want 200: %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/integration-settings", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("integration settings status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var settings struct {
		Data []database.IntegrationSetting `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.Data) != 1 || settings.Data[0].Integration != "jellyfin" || !settings.Data[0].APIKeyConfigured || settings.Data[0].APIKeyLast4 != "abcd" {
		t.Fatalf("settings = %#v", settings.Data)
	}
	if settings.Data[0].APIKey != "" {
		t.Fatal("integration settings API must not return secrets")
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("integrations status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var integrationBody struct {
		Data []integrations.Target `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&integrationBody); err != nil {
		t.Fatal(err)
	}
	for _, integration := range integrationBody.Data {
		if integration.ID == "jellyfin" && integration.Status == "configured" {
			goto sync
		}
	}
	t.Fatalf("jellyfin did not use stored settings: %#v", integrationBody.Data)

sync:
	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/integrations/jellyfin/sync", nil))
	if res.Code != http.StatusAccepted {
		t.Fatalf("sync status = %d, want 202: %s", res.Code, res.Body.String())
	}
}
