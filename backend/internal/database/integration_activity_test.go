package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

func TestMediaServerSnapshotPersistsNormalizedActivity(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	completedAt := parseIntegrationTestTime("2026-04-26T10:00:00Z")
	snapshot := MediaServerSnapshot{
		Server: MediaServer{
			ID:      "srv_jellyfin",
			Kind:    "jellyfin",
			Name:    "Jellyfin",
			BaseURL: "http://jellyfin.local",
			Status:  "configured",
		},
		Users: []MediaServerUser{
			{ServerID: "srv_jellyfin", ExternalID: "u1", DisplayName: "Alex"},
		},
		Libraries: []MediaServerLibrary{
			{ServerID: "srv_jellyfin", ExternalID: "lib_movies", Name: "Movies", Kind: "movie", ItemCount: 1},
		},
		Items: []MediaServerItem{
			{
				ServerID:          "srv_jellyfin",
				ExternalID:        "item_1",
				LibraryExternalID: "lib_movies",
				Kind:              "movie",
				Title:             "Arrival",
				Year:              2016,
				Path:              "/media/movies/Arrival (2016).mkv",
				ProviderIDs:       map[string]string{"Tmdb": "329865", "Imdb": "tt2543164"},
				RuntimeSeconds:    6960,
				MatchConfidence:   0.95,
			},
		},
		Files: []MediaServerFile{
			{
				ServerID:        "srv_jellyfin",
				ItemExternalID:  "item_1",
				Path:            "/media/movies/Arrival (2016).mkv",
				SizeBytes:       42_000_000_000,
				Container:       "mkv",
				Verification:    "server_reported",
				MatchConfidence: 0.7,
			},
		},
		Rollups: []MediaActivityRollup{
			{
				ServerID:       "srv_jellyfin",
				ItemExternalID: "item_1",
				PlayCount:      2,
				UniqueUsers:    1,
				WatchedUsers:   1,
				FavoriteCount:  0,
				LastPlayedAt:   parseIntegrationTestTime("2025-01-02T03:04:05Z"),
			},
		},
		Job: MediaSyncJob{
			ID:              "sync_1",
			ServerID:        "srv_jellyfin",
			Status:          "completed",
			ItemsImported:   1,
			RollupsImported: 1,
			UnmappedItems:   0,
			StartedAt:       completedAt.Add(-time.Minute),
			CompletedAt:     completedAt,
		},
	}

	if err := store.ReplaceMediaServerSnapshot(snapshot); err != nil {
		t.Fatal(err)
	}

	rollups, err := store.ListMediaActivityRollups("")
	if err != nil {
		t.Fatal(err)
	}
	if len(rollups) != 1 || rollups[0].PlayCount != 2 || rollups[0].UniqueUsers != 1 {
		t.Fatalf("rollups = %#v", rollups)
	}
	if rollups[0].LastPlayedAt.IsZero() || rollups[0].LastPlayedAt.Format(time.RFC3339) != "2025-01-02T03:04:05Z" {
		t.Fatalf("last played = %s", rollups[0].LastPlayedAt.Format(time.RFC3339))
	}

	items, err := store.ListMediaServerItems(MediaServerItemFilter{ServerID: "srv_jellyfin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Title != "Arrival" || items[0].ProviderIDs["Tmdb"] != "329865" {
		t.Fatalf("unexpected item: %#v", items[0])
	}

	job, err := store.LatestMediaSyncJob("srv_jellyfin")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "completed" || job.ItemsImported != 1 || job.RollupsImported != 1 {
		t.Fatalf("job = %#v", job)
	}
}

func TestPathMappingsCanBeUpsertedListedAndDeleted(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	mapping, err := store.UpsertPathMapping(PathMapping{
		ServerID:         "srv_jellyfin",
		ServerPathPrefix: "/mnt/media",
		LocalPathPrefix:  "/media",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mapping.ID == "" {
		t.Fatal("mapping id was not assigned")
	}

	updated, err := store.UpsertPathMapping(PathMapping{
		ID:               mapping.ID,
		ServerID:         "srv_jellyfin",
		ServerPathPrefix: "/srv/media",
		LocalPathPrefix:  "/media",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ServerPathPrefix != "/srv/media" {
		t.Fatalf("updated mapping = %#v", updated)
	}

	mappings, err := store.ListPathMappings()
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 1 || mappings[0].ID != mapping.ID {
		t.Fatalf("mappings = %#v", mappings)
	}

	if err := store.DeletePathMapping(mapping.ID); err != nil {
		t.Fatal(err)
	}
	mappings, err = store.ListPathMappings()
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 0 {
		t.Fatalf("mappings after delete = %#v", mappings)
	}
}

func TestPathMappingVerificationUpgradesFileEvidence(t *testing.T) {
	localRoot := t.TempDir()
	localMovie := filepath.Join(localRoot, "movies", "Arrival (2016).mkv")
	if err := os.MkdirAll(filepath.Dir(localMovie), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localMovie, []byte("verified media bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	snapshot := MediaServerSnapshot{
		Server: MediaServer{ID: "jellyfin", Kind: "jellyfin", Name: "Jellyfin", BaseURL: "http://jellyfin.local", Status: "configured"},
		Items: []MediaServerItem{{
			ServerID:          "jellyfin",
			ExternalID:        "item_1",
			LibraryExternalID: "movies",
			Kind:              "movie",
			Title:             "Arrival",
			Path:              "/mnt/media/movies/Arrival (2016).mkv",
			MatchConfidence:   0.7,
		}},
		Files: []MediaServerFile{{
			ServerID:        "jellyfin",
			ItemExternalID:  "item_1",
			Path:            "/mnt/media/movies/Arrival (2016).mkv",
			SizeBytes:       int64(len("verified media bytes")),
			Verification:    "server_reported",
			MatchConfidence: 0.7,
		}},
		Job: MediaSyncJob{ID: "sync_1", ServerID: "jellyfin", Status: "completed", StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC()},
	}
	if err := store.ReplaceMediaServerSnapshot(snapshot); err != nil {
		t.Fatal(err)
	}
	unmapped, err := store.ListMediaServerItems(MediaServerItemFilter{ServerID: "jellyfin", UnmappedOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(unmapped) != 1 {
		t.Fatalf("unmapped before verify = %#v, want one item", unmapped)
	}
	mapping, err := store.UpsertPathMapping(PathMapping{
		ServerID:         "jellyfin",
		ServerPathPrefix: "/mnt/media",
		LocalPathPrefix:  localRoot,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := store.VerifyPathMapping(mapping.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.MatchedFiles != 1 || result.VerifiedFiles != 1 || result.MissingFiles != 0 {
		t.Fatalf("verification result = %#v", result)
	}

	media, err := store.ListActivityRecommendationMedia()
	if err != nil {
		t.Fatal(err)
	}
	if len(media) != 1 || media[0].Path != localMovie || media[0].Verification != "local_verified" || media[0].MatchConfidence < 0.95 {
		t.Fatalf("activity media after verify = %#v", media)
	}
	unmapped, err = store.ListMediaServerItems(MediaServerItemFilter{ServerID: "jellyfin", UnmappedOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(unmapped) != 0 {
		t.Fatalf("unmapped after verify = %#v, want none", unmapped)
	}
}

func TestActivityRecommendationEvidencePersists(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	rec := recommendations.Recommendation{
		ID:              "rec_activity",
		Action:          recommendations.ActionReviewInactiveMovie,
		Title:           "Review inactive movie",
		Explanation:     "No one has watched this movie recently.",
		SpaceSavedBytes: 42_000_000_000,
		Confidence:      0.86,
		Source:          "rule:activity-inactive-movie",
		AffectedPaths:   []string{"/media/movies/Arrival (2016).mkv"},
		Destructive:     false,
		ServerID:        "jellyfin",
		ExternalItemID:  "item_1",
		LastPlayedAt:    parseIntegrationTestTime("2025-01-02T03:04:05Z"),
		PlayCount:       2,
		UniqueUsers:     1,
		FavoriteCount:   0,
		Verification:    "path_mapped",
		Evidence:        map[string]string{"inactiveDays": "600", "thresholdDays": "540"},
	}
	if err := store.ReplaceRecommendations([]recommendations.Recommendation{rec}); err != nil {
		t.Fatal(err)
	}

	recs, err := store.ListRecommendations()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("recommendations = %d, want 1", len(recs))
	}
	got := recs[0]
	if got.ServerID != "jellyfin" || got.ExternalItemID != "item_1" || got.PlayCount != 2 || got.UniqueUsers != 1 {
		t.Fatalf("activity evidence = %#v", got)
	}
	if got.Verification != "path_mapped" || got.Evidence["inactiveDays"] != "600" {
		t.Fatalf("evidence = %#v", got.Evidence)
	}
	if got.LastPlayedAt.Format(time.RFC3339) != "2025-01-02T03:04:05Z" {
		t.Fatalf("last played = %s", got.LastPlayedAt.Format(time.RFC3339))
	}
}

func parseIntegrationTestTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
