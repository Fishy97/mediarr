package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/database"
)

func TestPathMappingUnmappedAndVerifyRoutes(t *testing.T) {
	localRoot := t.TempDir()
	localMovie := filepath.Join(localRoot, "movies", "Arrival (2016).mkv")
	if err := os.MkdirAll(filepath.Dir(localMovie), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localMovie, []byte("verified media bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.ReplaceMediaServerSnapshot(database.MediaServerSnapshot{
		Server: database.MediaServer{ID: "jellyfin", Kind: "jellyfin", Name: "Jellyfin", BaseURL: "http://jellyfin.local", Status: "configured"},
		Items: []database.MediaServerItem{{
			ServerID:          "jellyfin",
			ExternalID:        "item_1",
			LibraryExternalID: "movies",
			Kind:              "movie",
			Title:             "Arrival",
			Path:              "/mnt/media/movies/Arrival (2016).mkv",
			MatchConfidence:   0.7,
		}},
		Files: []database.MediaServerFile{{
			ServerID:        "jellyfin",
			ItemExternalID:  "item_1",
			Path:            "/mnt/media/movies/Arrival (2016).mkv",
			SizeBytes:       int64(len("verified media bytes")),
			Verification:    "server_reported",
			MatchConfidence: 0.7,
		}},
		Job: database.MediaSyncJob{ID: "sync_1", ServerID: "jellyfin", Status: "completed", StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC()},
	}); err != nil {
		t.Fatal(err)
	}
	mapping, err := store.UpsertPathMapping(database.PathMapping{
		ServerID:         "jellyfin",
		ServerPathPrefix: "/mnt/media",
		LocalPathPrefix:  localRoot,
	})
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(Deps{Store: store})
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/path-mappings/unmapped", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("unmapped status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var unmappedBefore struct {
		Data []database.MediaServerItem `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&unmappedBefore); err != nil {
		t.Fatal(err)
	}
	if len(unmappedBefore.Data) != 1 {
		t.Fatalf("unmapped before verify = %#v", unmappedBefore.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/path-mappings/"+mapping.ID+"/verify", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("verify status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var verifyBody struct {
		Data database.PathMappingVerification `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&verifyBody); err != nil {
		t.Fatal(err)
	}
	if verifyBody.Data.VerifiedFiles != 1 || verifyBody.Data.MatchedFiles != 1 {
		t.Fatalf("verify result = %#v", verifyBody.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/path-mappings/unmapped", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("unmapped status after verify = %d, want 200: %s", res.Code, res.Body.String())
	}
	var unmappedAfter struct {
		Data []database.MediaServerItem `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&unmappedAfter); err != nil {
		t.Fatal(err)
	}
	if len(unmappedAfter.Data) != 0 {
		t.Fatalf("unmapped after verify = %#v, want none", unmappedAfter.Data)
	}
}
