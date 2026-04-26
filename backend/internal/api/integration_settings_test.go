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
	if !settings.Data[0].AutoSyncEnabled || settings.Data[0].AutoSyncIntervalMinutes != 360 {
		t.Fatalf("auto sync settings = %#v", settings.Data[0])
	}
	if settings.Data[0].APIKey != "" {
		t.Fatal("integration settings API must not return secrets")
	}

	jobs, err := store.ListJobs(database.JobFilter{Kind: "jellyfin_sync", TargetID: "jellyfin", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("saving configured integration should queue first auto sync, jobs = %#v", jobs)
	}
	waitForJobStatus(t, store, jobs[0].ID, "completed")

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

func TestQueueDueAutoSyncsSkipsDisabledAndActiveIntegrations(t *testing.T) {
	jellyfin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "jellyfin-secret-abcd" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin Auto"}`))
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

	disabled := false
	if _, err := store.UpsertIntegrationSetting(database.IntegrationSettingInput{
		Integration:             "plex",
		BaseURL:                 "http://plex:32400",
		APIKey:                  "plex-token",
		AutoSyncEnabled:         &disabled,
		AutoSyncIntervalMinutes: 60,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertIntegrationSetting(database.IntegrationSettingInput{
		Integration: "jellyfin",
		BaseURL:     jellyfin.URL,
		APIKey:      "jellyfin-secret-abcd",
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(Deps{Store: store})
	queued, err := server.QueueDueAutoSyncs()
	if err != nil {
		t.Fatal(err)
	}
	if len(queued) != 1 || queued[0].TargetID != "jellyfin" {
		t.Fatalf("queued jobs = %#v, want only jellyfin", queued)
	}

	duplicate, err := server.QueueDueAutoSyncs()
	if err != nil {
		t.Fatal(err)
	}
	if len(duplicate) != 0 {
		t.Fatalf("active auto sync should not be duplicated: %#v", duplicate)
	}

	waitForJobStatus(t, store, queued[0].ID, "completed")
	fresh, err := server.QueueDueAutoSyncs()
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh) != 0 {
		t.Fatalf("freshly completed auto sync should not be due: %#v", fresh)
	}
}
