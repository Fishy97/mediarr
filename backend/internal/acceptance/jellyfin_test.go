package acceptance

import (
	"strings"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/integrations"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

func TestParsePathMappings(t *testing.T) {
	mappings, err := ParsePathMappings(" /mnt/media = /media ; /volume1/anime=/media/anime ", "jellyfin")
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 2 {
		t.Fatalf("mappings = %d, want 2", len(mappings))
	}
	if mappings[0].ServerID != "jellyfin" || mappings[0].ServerPathPrefix != "/mnt/media" || mappings[0].LocalPathPrefix != "/media" {
		t.Fatalf("first mapping = %#v", mappings[0])
	}
	if mappings[1].ServerPathPrefix != "/volume1/anime" || mappings[1].LocalPathPrefix != "/media/anime" {
		t.Fatalf("second mapping = %#v", mappings[1])
	}
}

func TestParsePathMappingsRejectsMalformedEntry(t *testing.T) {
	if _, err := ParsePathMappings("/mnt/media:/media", "jellyfin"); err == nil {
		t.Fatal("expected malformed path mapping to fail")
	}
}

func TestBuildReportSummarizesAndRedactsLiveSignals(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	snapshot := database.MediaServerSnapshot{
		Server: database.MediaServer{ID: "jellyfin", Kind: "jellyfin", Name: "NAS Jellyfin", Status: "configured"},
		Libraries: []database.MediaServerLibrary{
			{ServerID: "jellyfin", ExternalID: "movies", Name: "Movies", Kind: "movie", ItemCount: 1},
			{ServerID: "jellyfin", ExternalID: "anime", Name: "Anime", Kind: "series", ItemCount: 2},
		},
		Items: []database.MediaServerItem{
			{ServerID: "jellyfin", ExternalID: "movie_1", LibraryExternalID: "movies", Kind: "movie", Title: "Arrival"},
			{ServerID: "jellyfin", ExternalID: "series_1", LibraryExternalID: "anime", Kind: "series", Title: "Cowboy Bebop"},
			{ServerID: "jellyfin", ExternalID: "episode_1", LibraryExternalID: "anime", ParentExternalID: "series_1", Kind: "episode", Title: "Asteroid Blues"},
		},
		Files: []database.MediaServerFile{
			{ServerID: "jellyfin", ItemExternalID: "movie_1", Path: "/mnt/movies/Arrival.mkv", LocalPath: "/media/movies/Arrival.mkv", SizeBytes: 42_000_000_000, Verification: "local_verified"},
			{ServerID: "jellyfin", ItemExternalID: "episode_1", Path: "/mnt/anime/Cowboy Bebop/S01E01.mkv", SizeBytes: 8_000_000_000, Verification: "server_reported"},
		},
		Rollups: []database.MediaActivityRollup{
			{ServerID: "jellyfin", ItemExternalID: "movie_1", PlayCount: 2, UniqueUsers: 1, LastPlayedAt: now.AddDate(-2, 0, 0)},
		},
		Job: database.MediaSyncJob{ID: "sync_1", ServerID: "jellyfin", Status: "completed", ItemsImported: 3, RollupsImported: 1, UnmappedItems: 1, StartedAt: now.Add(-time.Minute), CompletedAt: now},
	}
	recs := []recommendations.Recommendation{
		{
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
		},
	}
	progress := []integrations.Progress{
		{TargetID: "jellyfin", Phase: "items", Message: "Imported Arrival", CurrentLabel: "Arrival", Processed: 1, Total: 3},
	}

	report := BuildReport(snapshot, recs, progress, ReportOptions{
		TargetID:                 "jellyfin",
		GeneratedAt:              now,
		RedactTitles:             true,
		RequireLocalVerification: true,
	})

	if report.Summary.Movies != 1 || report.Summary.Series != 1 || report.Summary.Episodes != 1 || report.Summary.AnimeItems != 2 {
		t.Fatalf("summary counts = %#v", report.Summary)
	}
	if report.Summary.ServerReportedBytes != 50_000_000_000 || report.Summary.LocallyVerifiedBytes != 42_000_000_000 {
		t.Fatalf("summary bytes = %#v", report.Summary)
	}
	if report.Summary.Recommendations != 1 || report.TopRecommendations[0].Title != "[redacted]" {
		t.Fatalf("recommendations = %#v", report.TopRecommendations)
	}
	if len(report.Warnings) == 0 || !strings.Contains(strings.Join(report.Warnings, "\n"), "local verification") {
		t.Fatalf("warnings = %#v, want local verification warning", report.Warnings)
	}
	if report.ProgressSamples[0].CurrentLabel != "[redacted]" {
		t.Fatalf("progress sample = %#v", report.ProgressSamples[0])
	}
}
