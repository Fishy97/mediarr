package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/catalog"
	"github.com/Fishy97/mediarr/backend/internal/filescan"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
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

func TestStorePrunesStaleFilesForScannedLibrary(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now().UTC()
	first := filescan.Result{
		LibraryID:   "movies",
		CompletedAt: now,
		Items: []filescan.Item{
			{
				ID:          "file_keep",
				LibraryID:   "movies",
				Path:        "/media/movies/Keep.2020.mkv",
				Fingerprint: "keep",
				ModifiedAt:  now,
				Parsed: catalog.ParsedMedia{
					Kind:         catalog.KindMovie,
					Title:        "Keep",
					Year:         2020,
					CanonicalKey: "movie:keep:2020",
				},
			},
			{
				ID:          "file_gone",
				LibraryID:   "movies",
				Path:        "/media/movies/Gone.2020.mkv",
				Fingerprint: "gone",
				ModifiedAt:  now,
				Parsed: catalog.ParsedMedia{
					Kind:         catalog.KindMovie,
					Title:        "Gone",
					Year:         2020,
					CanonicalKey: "movie:gone:2020",
				},
			},
		},
	}
	if err := store.SaveScan(first); err != nil {
		t.Fatal(err)
	}

	second := filescan.Result{
		LibraryID:   "movies",
		CompletedAt: now.Add(time.Minute),
		Items:       first.Items[:1],
	}
	if err := store.SaveScan(second); err != nil {
		t.Fatal(err)
	}

	items, err := store.ListCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Path != "/media/movies/Keep.2020.mkv" {
		t.Fatalf("stale pruning failed: %#v", items)
	}
}

func TestProviderCacheHonorsExpiry(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now().UTC()
	if err := store.SetProviderCache("tmdb", "movie:arrival", `{"id":1}`, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	body, ok, err := store.GetProviderCache("tmdb", "movie:arrival", now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || body != `{"id":1}` {
		t.Fatalf("cache body = %q ok=%v", body, ok)
	}

	body, ok, err = store.GetProviderCache("tmdb", "movie:arrival", now.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if ok || body != "" {
		t.Fatalf("expired cache body = %q ok=%v", body, ok)
	}
}

func TestOpenMigratesLegacyDatabaseFilenames(t *testing.T) {
	for _, legacyName := range []string{"mediaar.db", "media-steward.db"} {
		t.Run(legacyName, func(t *testing.T) {
			configDir := t.TempDir()
			legacyPath := filepath.Join(configDir, legacyName)
			if err := os.WriteFile(legacyPath, []byte{}, 0o600); err != nil {
				t.Fatal(err)
			}

			store, err := Open(configDir)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()

			if _, err := os.Stat(filepath.Join(configDir, "mediarr.db")); err != nil {
				t.Fatalf("mediarr db missing after migration: %v", err)
			}
			if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
				t.Fatalf("legacy database should be renamed, stat err = %v", err)
			}
		})
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

func TestStoreIgnoresAndRestoresRecommendations(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	rec := recommendations.Recommendation{
		ID:              "rec_ignore",
		Action:          recommendations.ActionReviewDuplicate,
		Title:           "Review duplicate media",
		Explanation:     "Multiple files resolve to the same catalog item.",
		SpaceSavedBytes: 10,
		Confidence:      0.92,
		Source:          "rule:test",
		AffectedPaths:   []string{"/media/a.mkv"},
	}
	if err := store.ReplaceRecommendations([]recommendations.Recommendation{rec}); err != nil {
		t.Fatal(err)
	}
	if err := store.IgnoreRecommendation("rec_ignore"); err != nil {
		t.Fatal(err)
	}
	recs, err := store.ListRecommendations()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 {
		t.Fatalf("ignored recommendations = %#v, want none", recs)
	}
	if err := store.RestoreRecommendation("rec_ignore"); err != nil {
		t.Fatal(err)
	}
	recs, err = store.ListRecommendations()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("restored recommendations = %d, want 1", len(recs))
	}
}

func TestStorePersistsProviderSettingsWithRedactedSecrets(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	setting, err := store.UpsertProviderSetting(ProviderSettingInput{
		Provider: "tmdb",
		BaseURL:  "https://metadata.example.test",
		APIKey:   "secret-token-abcd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !setting.APIKeyConfigured || setting.APIKeyLast4 != "abcd" {
		t.Fatalf("redacted setting = %#v", setting)
	}

	settings, err := store.ListProviderSettings()
	if err != nil {
		t.Fatal(err)
	}
	if len(settings) != 1 || settings[0].APIKeyLast4 != "abcd" {
		t.Fatalf("settings = %#v", settings)
	}
	if settings[0].APIKey == "secret-token-abcd" {
		t.Fatal("provider API secret must not be returned by ListProviderSettings")
	}

	secrets, err := store.ListProviderSettingSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if len(secrets) != 1 || secrets[0].APIKey != "secret-token-abcd" {
		t.Fatalf("provider setting secret not retained for runtime use: %#v", secrets)
	}

	setting, err = store.UpsertProviderSetting(ProviderSettingInput{Provider: "tmdb", ClearAPIKey: true})
	if err != nil {
		t.Fatal(err)
	}
	if setting.APIKeyConfigured || setting.APIKeyLast4 != "" {
		t.Fatalf("cleared setting = %#v", setting)
	}
}

func TestStoreAppliesAndClearsCatalogCorrections(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now().UTC()
	if err := store.SaveScan(filescan.Result{
		LibraryID:   "movies",
		CompletedAt: now,
		Items: []filescan.Item{
			{
				ID:          "file_correct",
				LibraryID:   "movies",
				Path:        "/media/movies/Arrivall.2016.mkv",
				Fingerprint: "fingerprint_correct",
				ModifiedAt:  now,
				Parsed: catalog.ParsedMedia{
					Kind:         catalog.KindMovie,
					Title:        "Arrivall",
					Year:         2016,
					CanonicalKey: "movie:arrivall:2016",
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	correction, err := store.UpsertCatalogCorrection("file_correct", CatalogCorrectionInput{
		Title:      "Arrival",
		Kind:       catalog.KindMovie,
		Year:       2016,
		Provider:   "tmdb",
		ProviderID: "329865",
		Confidence: 0.97,
	})
	if err != nil {
		t.Fatal(err)
	}
	if correction.CanonicalKey != "movie:arrival:2016" {
		t.Fatalf("canonical key = %q", correction.CanonicalKey)
	}

	items, err := store.ListCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Title != "Arrival" || items[0].CanonicalKey != "movie:arrival:2016" {
		t.Fatalf("correction not applied to catalog: %#v", items[0])
	}
	if !items[0].MetadataCorrected || items[0].MetadataProvider != "tmdb" || items[0].MetadataProviderID != "329865" {
		t.Fatalf("correction metadata missing: %#v", items[0])
	}

	if err := store.ClearCatalogCorrection("file_correct"); err != nil {
		t.Fatal(err)
	}
	items, err = store.ListCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Title != "Arrivall" || items[0].MetadataCorrected {
		t.Fatalf("correction should be cleared: %#v", items[0])
	}
}

func TestCatalogCorrectionSurvivesSamePathRescan(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now().UTC()
	path := "/media/movies/Arrivall.2016.mkv"
	first := filescan.Result{
		LibraryID:   "movies",
		CompletedAt: now,
		Items: []filescan.Item{
			{
				ID:          "file_original",
				LibraryID:   "movies",
				Path:        path,
				Fingerprint: "fingerprint_original",
				ModifiedAt:  now,
				Parsed: catalog.ParsedMedia{
					Kind:         catalog.KindMovie,
					Title:        "Arrivall",
					Year:         2016,
					CanonicalKey: "movie:arrivall:2016",
				},
			},
		},
	}
	if err := store.SaveScan(first); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertCatalogCorrection("file_original", CatalogCorrectionInput{
		Title:      "Arrival",
		Kind:       catalog.KindMovie,
		Year:       2016,
		Confidence: 0.97,
	}); err != nil {
		t.Fatal(err)
	}

	second := first
	second.Items = []filescan.Item{first.Items[0]}
	second.Items[0].ID = "file_after_rescan"
	second.Items[0].Fingerprint = "fingerprint_after_rescan"
	second.Items[0].ModifiedAt = now.Add(time.Hour)
	second.CompletedAt = now.Add(time.Hour)
	if err := store.SaveScan(second); err != nil {
		t.Fatal(err)
	}

	items, err := store.ListCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "file_original" || items[0].Title != "Arrival" || !items[0].MetadataCorrected {
		t.Fatalf("correction should survive same-path rescan: %#v", items)
	}
}
