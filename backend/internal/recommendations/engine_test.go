package recommendations

import (
	"strconv"
	"testing"
	"time"
)

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

func TestEngineCreatesNeverWatchedActivityRecommendation(t *testing.T) {
	engine := Engine{}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	recs := engine.GenerateActivity([]ActivityMedia{
		{
			ServerID:        "jellyfin",
			ExternalItemID:  "item_1",
			Kind:            "movie",
			Title:           "Arrival",
			Path:            "/media/movies/Arrival (2016).mkv",
			SizeBytes:       42_000_000_000,
			AddedAt:         now.AddDate(0, -8, 0),
			PlayCount:       0,
			UniqueUsers:     0,
			FavoriteCount:   0,
			Verification:    "path_mapped",
			MatchConfidence: 0.86,
		},
	}, now)

	if len(recs) != 1 {
		t.Fatalf("recommendations = %d, want 1", len(recs))
	}
	if recs[0].Action != ActionReviewNeverWatchedMovie {
		t.Fatalf("action = %s, want %s", recs[0].Action, ActionReviewNeverWatchedMovie)
	}
	if recs[0].SpaceSavedBytes != 42_000_000_000 || recs[0].ServerID != "jellyfin" {
		t.Fatalf("recommendation evidence = %#v", recs[0])
	}
	if recs[0].Destructive {
		t.Fatal("activity recommendations must be suggest-only")
	}
}

func TestEngineCreatesInactiveActivityRecommendation(t *testing.T) {
	engine := Engine{}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	recs := engine.GenerateActivity([]ActivityMedia{
		{
			ServerID:        "plex",
			ExternalItemID:  "101",
			Kind:            "movie",
			Title:           "Arrival",
			Path:            "/media/movies/Arrival (2016).mkv",
			SizeBytes:       42_000_000_000,
			LastPlayedAt:    now.AddDate(-2, 0, 0),
			PlayCount:       2,
			UniqueUsers:     1,
			FavoriteCount:   0,
			Verification:    "local_verified",
			MatchConfidence: 0.95,
		},
	}, now)

	if len(recs) != 1 {
		t.Fatalf("recommendations = %d, want 1", len(recs))
	}
	if recs[0].Action != ActionReviewInactiveMovie {
		t.Fatalf("action = %s, want %s", recs[0].Action, ActionReviewInactiveMovie)
	}
	if recs[0].PlayCount != 2 || recs[0].UniqueUsers != 1 || recs[0].Verification != "local_verified" {
		t.Fatalf("activity evidence = %#v", recs[0])
	}
}

func TestStorageCertaintyEvidence(t *testing.T) {
	engine := Engine{}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name          string
		verification  string
		confidence    float64
		wantCertainty string
		wantVerified  string
	}{
		{name: "server reported", verification: "server_reported", confidence: 0.68, wantCertainty: "estimate", wantVerified: "0"},
		{name: "path mapped", verification: "path_mapped", confidence: 0.86, wantCertainty: "mapped_estimate", wantVerified: "0"},
		{name: "local verified", verification: "local_verified", confidence: 0.95, wantCertainty: "verified", wantVerified: "42000000000"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recs := engine.GenerateActivity([]ActivityMedia{
				{
					ServerID:        "jellyfin",
					ExternalItemID:  "item_1",
					Kind:            "movie",
					Title:           "Arrival",
					Path:            "/media/movies/Arrival (2016).mkv",
					SizeBytes:       42_000_000_000,
					AddedAt:         now.AddDate(0, -8, 0),
					PlayCount:       0,
					UniqueUsers:     0,
					FavoriteCount:   0,
					Verification:    tc.verification,
					MatchConfidence: tc.confidence,
				},
			}, now)

			if len(recs) != 1 {
				t.Fatalf("recommendations = %#v, want one recommendation", recs)
			}
			rec := recs[0]
			if rec.Confidence != activityConfidence(tc.confidence, tc.verification) {
				t.Fatalf("confidence = %f, want deterministic confidence", rec.Confidence)
			}
			if rec.Evidence["storageBasis"] != tc.verification {
				t.Fatalf("storageBasis = %q, want %q in %#v", rec.Evidence["storageBasis"], tc.verification, rec.Evidence)
			}
			if rec.Evidence["estimatedSavingsBytes"] != "42000000000" {
				t.Fatalf("estimatedSavingsBytes = %q", rec.Evidence["estimatedSavingsBytes"])
			}
			if rec.Evidence["verifiedSavingsBytes"] != tc.wantVerified {
				t.Fatalf("verifiedSavingsBytes = %q, want %q", rec.Evidence["verifiedSavingsBytes"], tc.wantVerified)
			}
			if rec.Evidence["storageCertainty"] != tc.wantCertainty {
				t.Fatalf("storageCertainty = %q, want %q", rec.Evidence["storageCertainty"], tc.wantCertainty)
			}
			evidence := BuildEvidence(rec)
			if evidence.Storage.EstimatedSavingsBytes != 42_000_000_000 {
				t.Fatalf("estimated storage = %d", evidence.Storage.EstimatedSavingsBytes)
			}
			if evidence.Storage.VerifiedSavingsBytes != mustParseTestInt64(t, tc.wantVerified) {
				t.Fatalf("verified storage = %d, want %s", evidence.Storage.VerifiedSavingsBytes, tc.wantVerified)
			}
			if evidence.Storage.Certainty != tc.wantCertainty {
				t.Fatalf("storage certainty = %q, want %q", evidence.Storage.Certainty, tc.wantCertainty)
			}
		})
	}
}

func TestEngineSuppressesActivityRecommendationsForFavorites(t *testing.T) {
	engine := Engine{}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	recs := engine.GenerateActivity([]ActivityMedia{
		{
			ServerID:        "jellyfin",
			ExternalItemID:  "item_1",
			Kind:            "movie",
			Title:           "Protected Favorite",
			Path:            "/media/movies/Favorite.mkv",
			SizeBytes:       42_000_000_000,
			AddedAt:         now.AddDate(-2, 0, 0),
			PlayCount:       0,
			FavoriteCount:   1,
			Verification:    "path_mapped",
			MatchConfidence: 0.86,
		},
	}, now)

	if len(recs) != 0 {
		t.Fatalf("favorite recommendations = %#v, want none", recs)
	}
}

func TestEngineAggregatesInactiveSeriesActivityRecommendation(t *testing.T) {
	engine := Engine{}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	recs := engine.GenerateActivity([]ActivityMedia{
		{
			ServerID:             "jellyfin",
			ExternalItemID:       "episode_1",
			ParentExternalItemID: "series_1",
			ParentTitle:          "The Expanse",
			Kind:                 "episode",
			Title:                "Dulcinea",
			Path:                 "/media/series/The Expanse/S01E01.mkv",
			SizeBytes:            12_000_000_000,
			AddedAt:              now.AddDate(-3, 0, 0),
			LastPlayedAt:         now.AddDate(-2, 0, 0),
			PlayCount:            1,
			UniqueUsers:          1,
			Verification:         "local_verified",
			MatchConfidence:      0.95,
		},
		{
			ServerID:             "jellyfin",
			ExternalItemID:       "episode_2",
			ParentExternalItemID: "series_1",
			ParentTitle:          "The Expanse",
			Kind:                 "episode",
			Title:                "The Big Empty",
			Path:                 "/media/series/The Expanse/S01E02.mkv",
			SizeBytes:            13_000_000_000,
			AddedAt:              now.AddDate(-3, 0, 0),
			LastPlayedAt:         now.AddDate(-2, 0, -10),
			PlayCount:            1,
			UniqueUsers:          1,
			Verification:         "path_mapped",
			MatchConfidence:      0.86,
		},
	}, now)

	if len(recs) != 1 {
		t.Fatalf("recommendations = %d, want 1: %#v", len(recs), recs)
	}
	if recs[0].Action != ActionReviewInactiveSeries {
		t.Fatalf("action = %s, want %s", recs[0].Action, ActionReviewInactiveSeries)
	}
	if recs[0].SpaceSavedBytes != 25_000_000_000 {
		t.Fatalf("space saved = %d, want 25000000000", recs[0].SpaceSavedBytes)
	}
	if len(recs[0].AffectedPaths) != 2 {
		t.Fatalf("affected paths = %#v, want both episode files", recs[0].AffectedPaths)
	}
	if recs[0].ExternalItemID != "series_1" || recs[0].PlayCount != 2 || recs[0].UniqueUsers != 1 {
		t.Fatalf("series evidence = %#v", recs[0])
	}
	if recs[0].Verification != "path_mapped" {
		t.Fatalf("verification = %s, want lowest group proof path_mapped", recs[0].Verification)
	}
	if recs[0].Destructive {
		t.Fatal("series recommendations must be suggest-only")
	}
}

func TestActivityRecommendationsUseSubjectTitles(t *testing.T) {
	engine := Engine{}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	recs := engine.GenerateActivity([]ActivityMedia{
		{
			ServerID:        "jellyfin",
			ExternalItemID:  "movie_1",
			Kind:            "movie",
			Title:           "Arrival",
			Path:            "/media/movies/Arrival (2016).mkv",
			SizeBytes:       42_000_000_000,
			AddedAt:         now.AddDate(-1, 0, 0),
			PlayCount:       0,
			Verification:    "server_reported",
			MatchConfidence: 0.68,
		},
		{
			ServerID:             "jellyfin",
			ExternalItemID:       "episode_1",
			ParentExternalItemID: "series_1",
			ParentTitle:          "The Sopranos",
			Kind:                 "episode",
			Title:                "Pilot",
			Path:                 "/media/series/The Sopranos/Season 1/The Sopranos - S01E01.mkv",
			SizeBytes:            10_000_000_000,
			AddedAt:              now.AddDate(-1, 0, 0),
			PlayCount:            0,
			Verification:         "server_reported",
			MatchConfidence:      0.68,
		},
	}, now)

	if len(recs) != 2 {
		t.Fatalf("recommendations = %#v, want movie and series recommendations", recs)
	}
	byAction := map[Action]Recommendation{}
	for _, rec := range recs {
		byAction[rec.Action] = rec
	}
	movie := byAction[ActionReviewNeverWatchedMovie]
	if movie.Title != "Arrival" || movie.Evidence["subjectTitle"] != "Arrival" || movie.Evidence["subjectKind"] != "movie" || movie.Evidence["itemCount"] != "1" {
		t.Fatalf("movie subject evidence = %#v", movie)
	}
	series := byAction[ActionReviewAbandonedSeries]
	if series.Title != "The Sopranos" || series.Evidence["seriesTitle"] != "The Sopranos" || series.Evidence["subjectTitle"] != "The Sopranos" || series.Evidence["subjectKind"] != "series" {
		t.Fatalf("series subject evidence = %#v", series)
	}
	if series.Evidence["itemCount"] != "1" || series.Action != ActionReviewAbandonedSeries {
		t.Fatalf("series item/action evidence = %#v", series)
	}
}

func TestEngineCreatesAbandonedAnimeSeriesRecommendation(t *testing.T) {
	engine := Engine{}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	recs := engine.GenerateActivity([]ActivityMedia{
		{
			ServerID:             "jellyfin",
			ExternalItemID:       "anime_episode_1",
			ParentExternalItemID: "anime_series_1",
			ParentTitle:          "Cowboy Bebop",
			LibraryName:          "Anime",
			Kind:                 "episode",
			Title:                "Asteroid Blues",
			Path:                 "/media/anime/Cowboy Bebop/S01E01.mkv",
			SizeBytes:            8_000_000_000,
			AddedAt:              now.AddDate(-1, 0, 0),
			PlayCount:            0,
			Verification:         "path_mapped",
			MatchConfidence:      0.88,
		},
	}, now)

	if len(recs) != 1 {
		t.Fatalf("recommendations = %d, want 1: %#v", len(recs), recs)
	}
	if recs[0].Action != ActionReviewAbandonedSeries {
		t.Fatalf("action = %s, want %s", recs[0].Action, ActionReviewAbandonedSeries)
	}
	if recs[0].Evidence["category"] != "anime" {
		t.Fatalf("evidence = %#v, want anime category", recs[0].Evidence)
	}
}

func TestEngineSuppressesActiveSeriesWithRecentlyAddedEpisode(t *testing.T) {
	engine := Engine{}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	recs := engine.GenerateActivity([]ActivityMedia{
		{
			ServerID:             "jellyfin",
			ExternalItemID:       "episode_1",
			ParentExternalItemID: "series_1",
			ParentTitle:          "Currently Airing",
			Kind:                 "episode",
			Title:                "Recent Episode",
			Path:                 "/media/series/Currently Airing/S01E01.mkv",
			SizeBytes:            10_000_000_000,
			AddedAt:              now.AddDate(0, 0, -12),
			LastPlayedAt:         now.AddDate(-3, 0, 0),
			PlayCount:            1,
			Verification:         "local_verified",
			MatchConfidence:      0.95,
		},
	}, now)

	if len(recs) != 0 {
		t.Fatalf("active series recommendations = %#v, want none", recs)
	}
}

func mustParseTestInt64(t *testing.T, value string) int64 {
	t.Helper()
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
