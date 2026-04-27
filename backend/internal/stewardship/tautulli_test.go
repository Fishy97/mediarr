package stewardship

import (
	"testing"
	"time"
)

func TestNormalizeTautulliHistoryParsesRows(t *testing.T) {
	payload := []byte(`{
		"response": {
			"result": "success",
			"data": {
				"data": [{
					"rating_key": "1234",
					"user": "Alex",
					"date": 1776700000,
					"percent_complete": 96
				}]
			}
		}
	}`)

	plays, err := NormalizeTautulliHistory(payload)
	if err != nil {
		t.Fatalf("NormalizeTautulliHistory returned error: %v", err)
	}
	if len(plays) != 1 {
		t.Fatalf("plays = %d, want 1", len(plays))
	}
	if plays[0].RatingKey != "1234" || plays[0].User != "Alex" || plays[0].PercentComplete != 96 {
		t.Fatalf("unexpected play row: %#v", plays[0])
	}
	if plays[0].PlayedAt.IsZero() {
		t.Fatalf("PlayedAt was not parsed: %#v", plays[0])
	}
}

func TestApplyTautulliHistoryEnrichesExistingRollups(t *testing.T) {
	existing := []ActivityRollup{
		{ServerID: "plex", ItemExternalID: "1234", PlayCount: 1, UniqueUsers: 1, WatchedUsers: 1, LastPlayedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	plays := []TautulliPlay{
		{RatingKey: "1234", User: "Alex", PlayedAt: time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC), PercentComplete: 95},
		{RatingKey: "9999", User: "Ignored", PlayedAt: time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC), PercentComplete: 95},
		{RatingKey: "1234", User: "Skipped", PlayedAt: time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC), PercentComplete: 30},
	}

	enriched := ApplyTautulliHistory(existing, plays, 80)
	if len(enriched) != 1 {
		t.Fatalf("enriched = %d, want 1", len(enriched))
	}
	if enriched[0].PlayCount != 2 || enriched[0].UniqueUsers != 2 || enriched[0].WatchedUsers != 2 {
		t.Fatalf("rollup not enriched: %#v", enriched[0])
	}
	if !enriched[0].LastPlayedAt.Equal(time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("LastPlayedAt = %s", enriched[0].LastPlayedAt)
	}
	if enriched[0].EvidenceSource != "tautulli" {
		t.Fatalf("EvidenceSource = %q", enriched[0].EvidenceSource)
	}
}
