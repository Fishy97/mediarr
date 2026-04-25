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

func TestParseSeriesMultiEpisodeAndSpecials(t *testing.T) {
	got := ParseMediaPath("/media/series/Doctor Who/Specials/Doctor.Who.S00E10.The.Christmas.Invasion.1080p.WEB-DL.mkv")
	if got.Kind != KindSeries {
		t.Fatalf("kind = %s, want %s", got.Kind, KindSeries)
	}
	if got.Season != 0 || got.Episode != 10 || got.EpisodeEnd != 10 {
		t.Fatalf("episode range = S%02dE%02d-E%02d, want S00E10-E10", got.Season, got.Episode, got.EpisodeEnd)
	}
	if got.CanonicalKey != "series:doctor-who:s00e10" {
		t.Fatalf("canonical key = %q", got.CanonicalKey)
	}

	got = ParseMediaPath("/media/series/Severance/Season 01/Severance.S01E01E02.Good.News.About.Hell.Half.Loop.1080p.WEB-DL.mkv")
	if got.Episode != 1 || got.EpisodeEnd != 2 {
		t.Fatalf("episode range = %d-%d, want 1-2", got.Episode, got.EpisodeEnd)
	}
	if got.CanonicalKey != "series:severance:s01e01-e02" {
		t.Fatalf("canonical key = %q", got.CanonicalKey)
	}
}

func TestParseMovieEditionAndCodecTags(t *testing.T) {
	got := ParseMediaPath("/media/movies/Blade Runner (1982)/Blade.Runner.1982.Final.Cut.2160p.UHD.BluRay.HDR10.DV.x265.mkv")

	if got.Kind != KindMovie {
		t.Fatalf("kind = %s, want movie", got.Kind)
	}
	if got.Edition != "Final Cut" {
		t.Fatalf("edition = %q, want Final Cut", got.Edition)
	}
	if got.Source != "bluray" {
		t.Fatalf("source = %q, want bluray", got.Source)
	}
	if got.VideoCodec != "hevc" {
		t.Fatalf("video codec = %q, want hevc", got.VideoCodec)
	}
	if got.DynamicRange != "dv,hdr10" {
		t.Fatalf("dynamic range = %q, want dv,hdr10", got.DynamicRange)
	}
}
