package integrations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/database"
)

func TestSyncEmbyImportsInventoryAndUserActivity(t *testing.T) {
	var itemRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "emby-key" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Emby Test"}`))
		case "/Users":
			_, _ = w.Write([]byte(`[{"Id":"user_1","Name":"Sam"}]`))
		case "/Items":
			itemRequests++
			if r.URL.Query().Get("Recursive") != "true" || r.URL.Query().Get("userId") != "user_1" {
				t.Fatalf("unexpected items query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{
				"Items": [{
					"Id": "emby_item_1",
					"Name": "Station Eleven",
					"Type": "Series",
					"ProductionYear": 2021,
					"Path": "/srv/media/series/Station Eleven",
					"ProviderIds": {"Tvdb":"367088"},
					"RunTimeTicks": 36000000000,
					"DateCreated": "2024-02-01T00:00:00Z",
					"MediaSources": [{
						"Path": "/srv/media/series/Station Eleven/S01E01.mkv",
						"Size": 12000000000,
						"Container": "mkv"
					}],
					"UserData": {
						"PlayCount": 3,
						"LastPlayedDate": "2025-03-02T03:04:05Z",
						"Played": true,
						"IsFavorite": true
					}
				}],
				"TotalRecordCount": 1
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	snapshot, err := SyncEmby(context.Background(), Options{EmbyURL: server.URL, EmbyKey: "emby-key"}, []database.PathMapping{
		{ServerID: "emby", ServerPathPrefix: "/srv/media", LocalPathPrefix: "/media"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if itemRequests != 1 {
		t.Fatalf("item requests = %d, want 1", itemRequests)
	}
	if snapshot.Server.ID != "emby" || snapshot.Server.Kind != "emby" || snapshot.Server.Name != "Emby Test" {
		t.Fatalf("server = %#v", snapshot.Server)
	}
	if len(snapshot.Users) != 1 || snapshot.Users[0].DisplayName != "Sam" {
		t.Fatalf("users = %#v", snapshot.Users)
	}
	if len(snapshot.Items) != 1 || snapshot.Items[0].Kind != "series" || snapshot.Items[0].ProviderIDs["Tvdb"] != "367088" {
		t.Fatalf("items = %#v", snapshot.Items)
	}
	if len(snapshot.Files) != 1 || snapshot.Files[0].LocalPath != "/media/series/Station Eleven/S01E01.mkv" {
		t.Fatalf("files = %#v", snapshot.Files)
	}
	if snapshot.Files[0].Verification != "path_mapped" || snapshot.Files[0].SizeBytes != 12_000_000_000 {
		t.Fatalf("file evidence = %#v", snapshot.Files[0])
	}
	if len(snapshot.Rollups) != 1 || snapshot.Rollups[0].PlayCount != 3 || snapshot.Rollups[0].FavoriteCount != 1 {
		t.Fatalf("rollups = %#v", snapshot.Rollups)
	}
	if snapshot.Job.Status != "completed" || snapshot.Job.ItemsImported != 1 || snapshot.Job.RollupsImported != 1 {
		t.Fatalf("job = %#v", snapshot.Job)
	}
}
