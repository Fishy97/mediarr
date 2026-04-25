package catalog

import "testing"

func TestParseMovieFilenameExtractsTitleYearAndQuality(t *testing.T) {
	got := ParseMediaPath("/media/movies/Arrival (2016)/Arrival.2016.2160p.BluRay.x265.mkv")

	if got.Kind != KindMovie {
		t.Fatalf("kind = %s, want %s", got.Kind, KindMovie)
	}
	if got.Title != "Arrival" {
		t.Fatalf("title = %q, want Arrival", got.Title)
	}
	if got.Year != 2016 {
		t.Fatalf("year = %d, want 2016", got.Year)
	}
	if got.Quality != "2160p" {
		t.Fatalf("quality = %q, want 2160p", got.Quality)
	}
	if got.CanonicalKey != "movie:arrival:2016" {
		t.Fatalf("canonical key = %q", got.CanonicalKey)
	}
}

func TestParseSeriesFilenameExtractsSeasonEpisode(t *testing.T) {
	got := ParseMediaPath("/media/series/Severance/Season 01/Severance.S01E03.In.Perpetuity.1080p.WEB-DL.mkv")

	if got.Kind != KindSeries {
		t.Fatalf("kind = %s, want %s", got.Kind, KindSeries)
	}
	if got.Title != "Severance" {
		t.Fatalf("title = %q, want Severance", got.Title)
	}
	if got.Season != 1 || got.Episode != 3 {
		t.Fatalf("season/episode = S%02dE%02d, want S01E03", got.Season, got.Episode)
	}
	if got.CanonicalKey != "series:severance:s01e03" {
		t.Fatalf("canonical key = %q", got.CanonicalKey)
	}
}

func TestParseAnimeAbsoluteEpisode(t *testing.T) {
	got := ParseMediaPath("/media/anime/Frieren Beyond Journey's End/[SubsPlease] Sousou no Frieren - 028 (1080p).mkv")

	if got.Kind != KindAnime {
		t.Fatalf("kind = %s, want %s", got.Kind, KindAnime)
	}
	if got.Title != "Sousou no Frieren" {
		t.Fatalf("title = %q, want Sousou no Frieren", got.Title)
	}
	if got.AbsoluteEpisode != 28 {
		t.Fatalf("absolute episode = %d, want 28", got.AbsoluteEpisode)
	}
	if got.CanonicalKey != "anime:sousou-no-frieren:e028" {
		t.Fatalf("canonical key = %q", got.CanonicalKey)
	}
}
