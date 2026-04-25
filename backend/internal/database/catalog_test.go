package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Fishy97/mediaar/backend/internal/catalog"
	"github.com/Fishy97/mediaar/backend/internal/filescan"
	"github.com/Fishy97/mediaar/backend/internal/recommendations"
)

func TestStorePersistsCatalogItemsFromScan(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	scan := filescan.Result{
		LibraryID:   "movies",
		StartedAt:   time.Now().UTC(),
		CompletedAt: time.Now().UTC(),
		Items: []filescan.Item{
			{
				ID:          "file_1",
				LibraryID:   "movies",
				Path:        filepath.Join("/media/movies", "Arrival.2016.1080p.mkv"),
				SizeBytes:   8_000_000_000,
				Fingerprint: "fingerprint_1",
				Subtitles:   []string{"/media/movies/Arrival.2016.1080p.en.srt"},
				ModifiedAt:  time.Now().UTC(),
				Parsed: catalog.ParsedMedia{
					Kind:         catalog.KindMovie,
					Title:        "Arrival",
					Year:         2016,
					Quality:      "1080p",
					CanonicalKey: "movie:arrival:2016",
				},
			},
		},
	}

	if err := store.SaveScan(scan); err != nil {
		t.Fatal(err)
	}

	items, err := store.ListCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Title != "Arrival" || items[0].CanonicalKey != "movie:arrival:2016" {
		t.Fatalf("unexpected catalog item: %#v", items[0])
	}
	if items[0].SizeBytes != 8_000_000_000 {
		t.Fatalf("size = %d", items[0].SizeBytes)
	}
	if len(items[0].Subtitles) != 1 {
		t.Fatalf("subtitles = %#v", items[0].Subtitles)
	}
}

func TestStoreReturnsEmptyCatalogList(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	items, err := store.ListCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if items == nil {
		t.Fatal("empty catalog should be an empty list, not nil")
	}
}

func TestOpenMigratesLegacyMediaStewardDatabaseFilename(t *testing.T) {
	configDir := t.TempDir()
	legacyPath := filepath.Join(configDir, "media-steward.db")
	if err := os.WriteFile(legacyPath, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}

	store, err := Open(configDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if _, err := os.Stat(filepath.Join(configDir, "mediaar.db")); err != nil {
		t.Fatalf("mediaar db missing after migration: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy database should be renamed, stat err = %v", err)
	}
}

func TestStoreReturnsEmptySubtitleListInsteadOfNull(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.SaveScan(filescan.Result{
		LibraryID:   "series",
		CompletedAt: time.Now().UTC(),
		Items: []filescan.Item{
			{
				ID:          "file_2",
				LibraryID:   "series",
				Path:        "/media/series/Severance.S01E03.mkv",
				SizeBytes:   28,
				Fingerprint: "fingerprint_2",
				ModifiedAt:  time.Now().UTC(),
				Parsed: catalog.ParsedMedia{
					Kind:         catalog.KindSeries,
					Title:        "Severance",
					CanonicalKey: "series:severance:s01e03",
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	items, err := store.ListCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Subtitles == nil {
		t.Fatal("subtitles should be an empty list, not nil")
	}
}

func TestStoreReplacesRecommendationsWithoutLosingSafetyFields(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	recs := []recommendations.Recommendation{
		{
			ID:              "rec_1",
			Action:          recommendations.ActionReviewDuplicate,
			Title:           "Review duplicate media",
			Explanation:     "Multiple files resolve to the same catalog item.",
			SpaceSavedBytes: 8_000_000_000,
			Confidence:      0.92,
			Source:          "rule:duplicate-canonical-key",
			AffectedPaths:   []string{"/media/a.mkv", "/media/b.mkv"},
			Destructive:     false,
		},
	}

	if err := store.ReplaceRecommendations(recs); err != nil {
		t.Fatal(err)
	}

	got, err := store.ListRecommendations()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("recommendations = %d, want 1", len(got))
	}
	if got[0].Destructive {
		t.Fatal("persisted recommendation became destructive")
	}
	if len(got[0].AffectedPaths) != 2 {
		t.Fatalf("affected paths = %#v", got[0].AffectedPaths)
	}
}

func TestStoreReturnsEmptyRecommendationList(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	recs, err := store.ListRecommendations()
	if err != nil {
		t.Fatal(err)
	}
	if recs == nil {
		t.Fatal("empty recommendations should be an empty list, not nil")
	}
}
