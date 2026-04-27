package stewardship

import (
	"testing"
	"time"
)

func TestNormalizeSeerrRequestsExtractsRequestSignals(t *testing.T) {
	payload := []byte(`{
		"pageInfo": {"page": 1, "pages": 1, "results": 1},
		"results": [{
			"id": 42,
			"status": 2,
			"createdAt": "2026-04-20T10:00:00.000Z",
			"updatedAt": "2026-04-21T12:00:00.000Z",
			"requestedBy": {"id": 7, "username": "alex", "plexUsername": "plexalex"},
			"media": {
				"id": 99,
				"tmdbId": 680,
				"tvdbId": 12345,
				"status": 5,
				"mediaType": "movie",
				"title": "Pulp Fiction",
				"imdbId": "tt0110912"
			}
		}]
	}`)

	signals, err := NormalizeSeerrRequests("seerr", payload)
	if err != nil {
		t.Fatalf("NormalizeSeerrRequests returned error: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("signals = %d, want 1", len(signals))
	}
	signal := signals[0]
	if signal.SourceID != "seerr" || signal.ExternalRequestID != "42" {
		t.Fatalf("unexpected source/request ids: %#v", signal)
	}
	if signal.MediaType != "movie" || signal.Title != "Pulp Fiction" {
		t.Fatalf("unexpected media identity: %#v", signal)
	}
	if signal.Status != "approved" || signal.Availability != "available" {
		t.Fatalf("unexpected status mapping: %#v", signal)
	}
	if signal.RequestedBy != "alex" {
		t.Fatalf("RequestedBy = %q, want alex", signal.RequestedBy)
	}
	if signal.ProviderIDs["tmdb"] != "680" || signal.ProviderIDs["tvdb"] != "12345" || signal.ProviderIDs["imdb"] != "tt0110912" {
		t.Fatalf("provider ids not normalized: %#v", signal.ProviderIDs)
	}
	if signal.RequestedAt.IsZero() || signal.UpdatedAt.IsZero() {
		t.Fatalf("timestamps were not parsed: %#v", signal)
	}
}

func TestNormalizeSeerrRequestsRejectsMalformedPayload(t *testing.T) {
	if _, err := NormalizeSeerrRequests("seerr", []byte(`{"results":`)); err == nil {
		t.Fatal("expected malformed JSON to fail")
	}
}

func TestRequestSignalMatchKeyPrefersProviderIds(t *testing.T) {
	signal := RequestSignal{
		SourceID:          "seerr",
		ExternalRequestID: "1",
		MediaType:         "movie",
		ProviderIDs:       map[string]string{"tmdb": "680"},
		RequestedAt:       time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC),
	}
	if got := signal.MatchKey(); got != "movie:tmdb:680" {
		t.Fatalf("MatchKey() = %q", got)
	}
}
