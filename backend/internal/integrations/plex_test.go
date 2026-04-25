package integrations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/database"
)

func TestSyncPlexImportsLibraryPartsAndHistory(t *testing.T) {
	var historyRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Plex-Token") != "plex-token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Path {
		case "/identity":
			_, _ = w.Write([]byte(`<MediaContainer machineIdentifier="plex-machine" friendlyName="Plex Test"/>`))
		case "/library/sections":
			_, _ = w.Write([]byte(`<MediaContainer size="1"><Directory key="1" title="Movies" type="movie"/></MediaContainer>`))
		case "/library/sections/1/all":
			_, _ = w.Write([]byte(`<MediaContainer size="1">
				<Video ratingKey="101" key="/library/metadata/101" title="Arrival" type="movie" year="2016" addedAt="1704067200" duration="6960000">
					<Guid id="tmdb://329865"/>
					<Guid id="imdb://tt2543164"/>
					<Media duration="6960000">
						<Part file="/mnt/media/movies/Arrival (2016).mkv" size="42000000000" container="mkv"/>
					</Media>
				</Video>
			</MediaContainer>`))
		case "/status/sessions/history/all":
			historyRequests++
			_, _ = w.Write([]byte(`<MediaContainer size="1">
				<Video ratingKey="101" title="Arrival" type="movie" viewedAt="1735787045" accountID="1"/>
			</MediaContainer>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	snapshot, err := SyncPlex(context.Background(), Options{PlexURL: server.URL, PlexToken: "plex-token"}, []database.PathMapping{
		{ServerID: "plex", ServerPathPrefix: "/mnt/media", LocalPathPrefix: "/media"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if historyRequests != 1 {
		t.Fatalf("history requests = %d, want 1", historyRequests)
	}
	if snapshot.Server.ID != "plex" || snapshot.Server.Kind != "plex" {
		t.Fatalf("server = %#v", snapshot.Server)
	}
	if len(snapshot.Libraries) != 1 || snapshot.Libraries[0].Name != "Movies" {
		t.Fatalf("libraries = %#v", snapshot.Libraries)
	}
	if len(snapshot.Items) != 1 || snapshot.Items[0].ProviderIDs["tmdb"] != "329865" || snapshot.Items[0].RuntimeSeconds != 6960 {
		t.Fatalf("items = %#v", snapshot.Items)
	}
	if len(snapshot.Files) != 1 || snapshot.Files[0].LocalPath != "/media/movies/Arrival (2016).mkv" {
		t.Fatalf("files = %#v", snapshot.Files)
	}
	if snapshot.Files[0].SizeBytes != 42_000_000_000 || snapshot.Files[0].Verification != "path_mapped" {
		t.Fatalf("file evidence = %#v", snapshot.Files[0])
	}
	if len(snapshot.Rollups) != 1 || snapshot.Rollups[0].PlayCount != 1 || snapshot.Rollups[0].UniqueUsers != 1 {
		t.Fatalf("rollups = %#v", snapshot.Rollups)
	}
	if !snapshot.Rollups[0].LastPlayedAt.Equal(time.Unix(1735787045, 0).UTC()) {
		t.Fatalf("last played = %s", snapshot.Rollups[0].LastPlayedAt)
	}
	if snapshot.Job.Status != "completed" || snapshot.Job.ItemsImported != 1 || snapshot.Job.RollupsImported != 1 {
		t.Fatalf("job = %#v", snapshot.Job)
	}
}
