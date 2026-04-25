package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/auth"
	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/integrations"
)

func TestJellyfinSyncRoutePersistsNormalizedActivity(t *testing.T) {
	jellyfin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "jellyfin-key" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin Test"}`))
		case "/Users":
			_, _ = w.Write([]byte(`[{"Id":"user_1","Name":"Alex"}]`))
		case "/Items":
			_, _ = w.Write([]byte(`{
				"Items": [{
					"Id": "item_1",
					"Name": "Arrival",
					"Type": "Movie",
					"ProductionYear": 2016,
					"Path": "/media/movies/Arrival (2016).mkv",
					"ProviderIds": {"Tmdb":"329865"},
					"MediaSources": [{"Path":"/media/movies/Arrival (2016).mkv","Size":42000000000,"Container":"mkv"}],
					"UserData": {"PlayCount":1,"LastPlayedDate":"2025-01-02T03:04:05Z","Played":true}
				}],
				"TotalRecordCount": 1
			}`))
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

	server := NewServer(Deps{
		Store: store,
		IntegrationOptions: integrations.Options{
			JellyfinURL: jellyfin.URL,
			JellyfinKey: "jellyfin-key",
		},
	})
	handler := auth.Middleware{AdminToken: "secret"}.Wrap(server)

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/integrations/jellyfin/sync", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated sync status = %d, want 401", res.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/jellyfin/sync", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("sync status = %d, want 202: %s", res.Code, res.Body.String())
	}
	var syncBody struct {
		Data database.MediaSyncJob `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&syncBody); err != nil {
		t.Fatal(err)
	}
	if syncBody.Data.Status != "completed" || syncBody.Data.ItemsImported != 1 {
		t.Fatalf("sync body = %#v", syncBody.Data)
	}

	items, err := store.ListMediaServerItems(database.MediaServerItemFilter{ServerID: "jellyfin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "Arrival" {
		t.Fatalf("items = %#v", items)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/activity/rollups", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("rollups status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var rollupBody struct {
		Data []database.MediaActivityRollup `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&rollupBody); err != nil {
		t.Fatal(err)
	}
	if len(rollupBody.Data) != 1 || rollupBody.Data[0].PlayCount != 1 {
		t.Fatalf("rollups = %#v", rollupBody.Data)
	}
}
