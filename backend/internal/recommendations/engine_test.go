package recommendations

import "testing"

func TestDuplicateRecommendationIncludesSpaceSavedAndAllPaths(t *testing.T) {
	engine := Engine{}
	recs := engine.Generate([]MediaFile{
		{ID: "a", CanonicalKey: "movie:arrival:2016", Path: "/media/movies/Arrival.2016.1080p.mkv", SizeBytes: 8_000_000_000, Quality: "1080p"},
		{ID: "b", CanonicalKey: "movie:arrival:2016", Path: "/media/movies/Arrival.2016.2160p.mkv", SizeBytes: 19_000_000_000, Quality: "2160p"},
	})

	if len(recs) != 1 {
		t.Fatalf("recommendations = %d, want 1", len(recs))
	}
	if recs[0].Action != ActionReviewDuplicate {
		t.Fatalf("action = %s", recs[0].Action)
	}
	if recs[0].SpaceSavedBytes != 8_000_000_000 {
		t.Fatalf("space saved = %d, want 8000000000", recs[0].SpaceSavedBytes)
	}
	if len(recs[0].AffectedPaths) != 2 {
		t.Fatalf("affected paths = %#v, want both duplicate paths", recs[0].AffectedPaths)
	}
	if recs[0].Destructive {
		t.Fatal("cleanup recommendations must be suggest-only and non-destructive")
	}
}

func TestOversizedRecommendationIsAdvisory(t *testing.T) {
	engine := Engine{OversizedThresholdBytes: 50_000_000_000}
	recs := engine.Generate([]MediaFile{
		{ID: "a", CanonicalKey: "movie:example:2020", Path: "/media/movies/Example.2020.1080p.mkv", SizeBytes: 71_000_000_000, Quality: "1080p"},
	})

	if len(recs) != 1 {
		t.Fatalf("recommendations = %d, want 1", len(recs))
	}
	if recs[0].Action != ActionReviewOversized {
		t.Fatalf("action = %s", recs[0].Action)
	}
	if recs[0].Destructive {
		t.Fatal("oversized recommendations must not delete media")
	}
}

func TestEngineCreatesMissingSubtitleRecommendations(t *testing.T) {
	engine := Engine{}
	recs := engine.Generate([]MediaFile{
		{ID: "movie_1", CanonicalKey: "movie:arrival:2016", Path: "/media/Arrival.2016.mkv", SizeBytes: 10, Quality: "1080p", WantsSubtitles: true, HasSubtitles: false},
	})

	if len(recs) != 1 {
		t.Fatalf("recommendations = %d, want 1", len(recs))
	}
	if recs[0].Action != ActionReviewMissingSubtitles {
		t.Fatalf("action = %s, want %s", recs[0].Action, ActionReviewMissingSubtitles)
	}
	if recs[0].Destructive {
		t.Fatal("missing subtitle recommendations must not delete media")
	}
}
