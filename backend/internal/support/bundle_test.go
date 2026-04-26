package support

import (
	"archive/zip"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

func TestCreateBundleRedactsSecretsAndIncludesDiagnostics(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if _, err := store.UpsertProviderSetting(database.ProviderSettingInput{Provider: "tmdb", APIKey: "tmdb-secret-token"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertIntegrationSetting(database.IntegrationSettingInput{Integration: "jellyfin", BaseURL: "http://jellyfin.local", APIKey: "jellyfin-secret-token"}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	if err := store.ReplaceMediaServerSnapshot(database.MediaServerSnapshot{
		Server: database.MediaServer{ID: "jellyfin", Kind: "jellyfin", Name: "Jellyfin", BaseURL: "http://jellyfin.local", Status: "configured", LastSyncedAt: now, UpdatedAt: now},
		Libraries: []database.MediaServerLibrary{
			{ServerID: "jellyfin", ExternalID: "movies", Name: "Movies", Kind: "movie", ItemCount: 1},
		},
		Items: []database.MediaServerItem{
			{ServerID: "jellyfin", ExternalID: "movie_1", LibraryExternalID: "movies", Kind: "movie", Title: "Arrival", Path: "/mnt/movies/Arrival.mkv", DateCreated: now.AddDate(-2, 0, 0), MatchConfidence: 0.9, UpdatedAt: now},
		},
		Files: []database.MediaServerFile{
			{ServerID: "jellyfin", ItemExternalID: "movie_1", Path: "/mnt/movies/Arrival.mkv", LocalPath: "/media/movies/Arrival.mkv", SizeBytes: 42_000_000_000, Verification: "local_verified", MatchConfidence: 0.95},
		},
		Rollups: []database.MediaActivityRollup{
			{ServerID: "jellyfin", ItemExternalID: "movie_1", PlayCount: 1, UniqueUsers: 1, WatchedUsers: 1, LastPlayedAt: now.AddDate(-2, 0, 0), UpdatedAt: now},
		},
		Job: database.MediaSyncJob{ID: "sync_1", ServerID: "jellyfin", Status: "completed", ItemsImported: 1, RollupsImported: 1, StartedAt: now.Add(-time.Minute), CompletedAt: now},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceRecommendations([]recommendations.Recommendation{{
		ID:              "rec_1",
		Action:          recommendations.ActionReviewInactiveMovie,
		Title:           "Review inactive movie",
		Explanation:     "Arrival has not been watched recently.",
		SpaceSavedBytes: 42_000_000_000,
		Confidence:      0.9,
		Source:          "rule:activity-inactive-movie",
		AffectedPaths:   []string{"/media/movies/Arrival.mkv"},
		ServerID:        "jellyfin",
		ExternalItemID:  "movie_1",
		Verification:    "local_verified",
	}}); err != nil {
		t.Fatal(err)
	}

	result, err := CreateBundle(Config{
		Store:     store,
		OutputDir: t.TempDir(),
		Service:   "mediarr",
		Version:   "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Path == "" || result.SizeBytes <= 0 {
		t.Fatalf("bundle result = %#v", result)
	}

	reader, err := zip.OpenReader(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	contents := map[string]string{}
	for _, file := range reader.File {
		body, err := readZipFile(file)
		if err != nil {
			t.Fatal(err)
		}
		contents[file.Name] = body
	}
	for _, name := range []string{"manifest.json", "settings/providers.json", "settings/integrations.json", "diagnostics/jellyfin.json", "recommendations.json", "jobs.json", "safety.json"} {
		if _, ok := contents[name]; !ok {
			t.Fatalf("bundle missing %s; entries=%v", name, result.Files)
		}
	}
	all := strings.Join(mapsValues(contents), "\n")
	if strings.Contains(all, "tmdb-secret-token") || strings.Contains(all, "jellyfin-secret-token") {
		t.Fatalf("bundle leaked secret data: %s", all)
	}
	if !strings.Contains(contents["diagnostics/jellyfin.json"], "locallyVerifiedBytes") {
		t.Fatalf("diagnostics entry = %s", contents["diagnostics/jellyfin.json"])
	}

	var manifest Manifest
	if err := json.Unmarshal([]byte(contents["manifest.json"]), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Service != "mediarr" || manifest.Version != "test" || manifest.GeneratedAt.IsZero() {
		t.Fatalf("manifest = %#v", manifest)
	}
}

func readZipFile(file *zip.File) (string, error) {
	handle, err := file.Open()
	if err != nil {
		return "", err
	}
	defer handle.Close()
	body, err := io.ReadAll(handle)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func mapsValues(values map[string]string) []string {
	output := make([]string, 0, len(values))
	for _, value := range values {
		output = append(output, value)
	}
	return output
}
