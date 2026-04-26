package integrations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/database"
)

func TestSyncJellyfinImportsInventoryAndUserActivity(t *testing.T) {
	var itemRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "jellyfin-key" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Test Jellyfin"}`))
		case "/Users":
			_, _ = w.Write([]byte(`[{"Id":"user_1","Name":"Alex"}]`))
		case "/Items":
			itemRequests++
			if r.URL.Query().Get("Recursive") != "true" || r.URL.Query().Get("userId") != "user_1" {
				t.Fatalf("unexpected items query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{
				"Items": [{
					"Id": "item_1",
					"Name": "Arrival",
					"Type": "Movie",
					"ProductionYear": 2016,
					"Path": "/mnt/media/movies/Arrival (2016).mkv",
					"ProviderIds": {"Tmdb":"329865","Imdb":"tt2543164"},
					"RunTimeTicks": 69600000000,
					"DateCreated": "2024-01-01T00:00:00Z",
					"MediaSources": [{
						"Path": "/mnt/media/movies/Arrival (2016).mkv",
						"Size": 42000000000,
						"Container": "mkv"
					}],
					"UserData": {
						"PlayCount": 2,
						"LastPlayedDate": "2025-01-02T03:04:05Z",
						"Played": true,
						"IsFavorite": false
					}
				}],
				"TotalRecordCount": 1
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	snapshot, err := SyncJellyfin(context.Background(), Options{JellyfinURL: server.URL, JellyfinKey: "jellyfin-key"}, []database.PathMapping{
		{ServerID: "jellyfin", ServerPathPrefix: "/mnt/media", LocalPathPrefix: "/media"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if itemRequests != 1 {
		t.Fatalf("item requests = %d, want 1", itemRequests)
	}
	if snapshot.Server.ID != "jellyfin" || snapshot.Server.Kind != "jellyfin" {
		t.Fatalf("server = %#v", snapshot.Server)
	}
	if len(snapshot.Users) != 1 || snapshot.Users[0].DisplayName != "Alex" {
		t.Fatalf("users = %#v", snapshot.Users)
	}
	if len(snapshot.Items) != 1 || snapshot.Items[0].ProviderIDs["Tmdb"] != "329865" || snapshot.Items[0].RuntimeSeconds != 6960 {
		t.Fatalf("items = %#v", snapshot.Items)
	}
	if len(snapshot.Files) != 1 {
		t.Fatalf("files = %#v", snapshot.Files)
	}
	if snapshot.Files[0].Path != "/mnt/media/movies/Arrival (2016).mkv" || snapshot.Files[0].LocalPath != "/media/movies/Arrival (2016).mkv" {
		t.Fatalf("file mapping = %#v", snapshot.Files[0])
	}
	if snapshot.Files[0].SizeBytes != 42_000_000_000 || snapshot.Files[0].Verification != "path_mapped" {
		t.Fatalf("file evidence = %#v", snapshot.Files[0])
	}
	if len(snapshot.Rollups) != 1 || snapshot.Rollups[0].PlayCount != 2 || snapshot.Rollups[0].UniqueUsers != 1 || snapshot.Rollups[0].FavoriteCount != 0 {
		t.Fatalf("rollups = %#v", snapshot.Rollups)
	}
	if snapshot.Rollups[0].LastPlayedAt.Format("2006-01-02T15:04:05Z") != "2025-01-02T03:04:05Z" {
		t.Fatalf("last played = %s", snapshot.Rollups[0].LastPlayedAt)
	}
	if snapshot.Job.Status != "completed" || snapshot.Job.ItemsImported != 1 || snapshot.Job.RollupsImported != 1 {
		t.Fatalf("job = %#v", snapshot.Job)
	}
}

func TestSyncJellyfinKeepsServerReportedEvidenceWithoutPathMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "jellyfin-key" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Test Jellyfin"}`))
		case "/Users":
			_, _ = w.Write([]byte(`[{"Id":"user_1","Name":"Alex"}]`))
		case "/Items":
			_, _ = w.Write([]byte(`{
				"Items": [{
					"Id": "item_1",
					"Name": "Arrival",
					"Type": "Movie",
					"Path": "/Volume1/Media/Movies/Arrival (2016).mkv",
					"DateCreated": "2024-01-01T00:00:00Z",
					"MediaSources": [{"Path":"/Volume1/Media/Movies/Arrival (2016).mkv","Size":42000000000,"Container":"mkv"}],
					"UserData": {"PlayCount":0,"Played":false}
				}],
				"TotalRecordCount": 1
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	snapshot, err := SyncJellyfin(context.Background(), Options{JellyfinURL: server.URL, JellyfinKey: "jellyfin-key"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Files) != 1 {
		t.Fatalf("files = %#v", snapshot.Files)
	}
	file := snapshot.Files[0]
	if file.Verification != "server_reported" || file.LocalPath != "" {
		t.Fatalf("file evidence = %#v, want server-reported without local path", file)
	}
	if file.MatchConfidence < 0.65 {
		t.Fatalf("server-reported confidence = %f, want recommendation-eligible confidence", file.MatchConfidence)
	}
}

func TestSyncJellyfinReportsProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "jellyfin-key" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Test Jellyfin"}`))
		case "/Users":
			_, _ = w.Write([]byte(`[{"Id":"user_1","Name":"Alex"}]`))
		case "/Items":
			_, _ = w.Write([]byte(`{
				"Items": [{
					"Id": "item_1",
					"Name": "Arrival",
					"Type": "Movie",
					"Path": "/mnt/media/movies/Arrival (2016).mkv",
					"MediaSources": [{"Path":"/mnt/media/movies/Arrival (2016).mkv","Size":42000000000,"Container":"mkv"}],
					"UserData": {"PlayCount":1,"Played":true}
				}],
				"TotalRecordCount": 1
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var events []Progress
	_, err := SyncJellyfin(context.Background(), Options{
		JellyfinURL: server.URL,
		JellyfinKey: "jellyfin-key",
		Progress: func(progress Progress) {
			events = append(events, progress)
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(events) < 4 {
		t.Fatalf("progress events = %#v", events)
	}
	if events[0].TargetID != "jellyfin" || events[0].Phase != "connecting" {
		t.Fatalf("first event = %#v", events[0])
	}
	var sawItem bool
	for _, event := range events {
		if event.CurrentLabel == "/mnt/media/movies/Arrival (2016).mkv" {
			t.Fatalf("progress leaked full path: %#v", event)
		}
		if event.CurrentLabel == "Arrival" && event.Processed == 1 && event.Total == 1 {
			sawItem = true
		}
	}
	if !sawItem {
		t.Fatalf("did not see item progress: %#v", events)
	}
	last := events[len(events)-1]
	if last.Phase != "complete" || last.ItemsImported != 1 || last.RollupsImported != 1 {
		t.Fatalf("last event = %#v", last)
	}
}
