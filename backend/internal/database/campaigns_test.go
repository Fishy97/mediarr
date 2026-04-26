package database

import (
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/campaigns"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

func TestCampaignPersistenceRoundTrip(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	input := campaigns.Campaign{
		ID:                  "campaign_cold_movies",
		Name:                "Cold Movies",
		Description:         "Movies that have not been played recently.",
		Enabled:             true,
		TargetKinds:         []string{"movie"},
		TargetLibraryNames:  []string{"Movies"},
		RequireAllRules:     true,
		MinimumConfidence:   0.75,
		MinimumStorageBytes: 10_000_000_000,
		Rules: []campaigns.Rule{
			{Field: campaigns.FieldLastPlayedDays, Operator: campaigns.OperatorGreaterOrEqual, Value: "365"},
		},
	}

	saved, err := store.UpsertCampaign(input)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID != input.ID || saved.Name != input.Name || saved.CreatedAt.IsZero() || saved.UpdatedAt.IsZero() {
		t.Fatalf("saved campaign = %#v", saved)
	}
	got, err := store.GetCampaign(input.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != input.Name || got.TargetKinds[0] != "movie" || got.Rules[0].Value != "365" {
		t.Fatalf("got campaign = %#v", got)
	}
	list, err := store.ListCampaigns()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != input.ID {
		t.Fatalf("list = %#v", list)
	}

	input.Name = "Deep Archive Movies"
	updated, err := store.UpsertCampaign(input)
	if err != nil {
		t.Fatal(err)
	}
	if updated.CreatedAt.IsZero() || updated.UpdatedAt.Before(updated.CreatedAt) || updated.Name != "Deep Archive Movies" {
		t.Fatalf("updated campaign = %#v", updated)
	}

	if err := store.DeleteCampaign(input.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetCampaign(input.ID); err == nil {
		t.Fatal("expected deleted campaign lookup to fail")
	}
}

func TestCampaignRunPersistenceRoundTrip(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	started := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	completed := started.Add(2 * time.Minute)
	run := campaigns.Run{
		ID:                    "run_1",
		CampaignID:            "campaign_cold_movies",
		Status:                "completed",
		Matched:               3,
		Suppressed:            1,
		EstimatedSavingsBytes: 90_000_000_000,
		VerifiedSavingsBytes:  42_000_000_000,
		StartedAt:             started,
		CompletedAt:           completed,
	}
	if err := store.RecordCampaignRun(run); err != nil {
		t.Fatal(err)
	}
	runs, err := store.ListCampaignRuns(run.CampaignID)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("runs = %#v", runs)
	}
	if runs[0].ID != run.ID || runs[0].Matched != 3 || !runs[0].CompletedAt.Equal(completed) {
		t.Fatalf("run = %#v", runs[0])
	}
}

func TestReplaceCampaignRecommendationsPreservesUserDecisions(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	campaignID := "campaign_cold_movies"
	if err := store.ReplaceRecommendations([]recommendations.Recommendation{
		{
			ID:              "existing_open",
			Action:          recommendations.ActionReviewCampaignMatch,
			State:           recommendations.StateNew,
			Title:           "Old open campaign match",
			Explanation:     "old",
			SpaceSavedBytes: 1,
			Confidence:      0.8,
			Source:          "campaign:" + campaignID,
			AffectedPaths:   []string{"/media/old.mkv"},
			Destructive:     false,
		},
		{
			ID:              "protected_match",
			Action:          recommendations.ActionReviewCampaignMatch,
			State:           recommendations.StateProtected,
			Title:           "Protected campaign match",
			Explanation:     "protected",
			SpaceSavedBytes: 2,
			Confidence:      0.8,
			Source:          "campaign:" + campaignID,
			AffectedPaths:   []string{"/media/protected.mkv"},
			Destructive:     false,
		},
		{
			ID:              "other_campaign",
			Action:          recommendations.ActionReviewCampaignMatch,
			State:           recommendations.StateNew,
			Title:           "Other campaign",
			Explanation:     "other",
			SpaceSavedBytes: 3,
			Confidence:      0.8,
			Source:          "campaign:other",
			AffectedPaths:   []string{"/media/other.mkv"},
			Destructive:     false,
		},
	}); err != nil {
		t.Fatal(err)
	}

	next := []recommendations.Recommendation{
		{
			ID:              "fresh_match",
			Action:          recommendations.ActionReviewCampaignMatch,
			State:           recommendations.StateNew,
			Title:           "Fresh match",
			Explanation:     "fresh",
			SpaceSavedBytes: 4,
			Confidence:      0.9,
			Source:          "campaign:" + campaignID,
			AffectedPaths:   []string{"/media/fresh.mkv"},
			Destructive:     false,
		},
	}
	if err := store.ReplaceCampaignRecommendations(campaignID, next); err != nil {
		t.Fatal(err)
	}

	recs, err := store.ListRecommendations()
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]recommendations.Recommendation{}
	for _, rec := range recs {
		byID[rec.ID] = rec
	}
	if _, ok := byID["existing_open"]; ok {
		t.Fatalf("open campaign recommendation was not replaced: %#v", byID["existing_open"])
	}
	protected, err := store.GetRecommendation("protected_match")
	if err != nil {
		t.Fatal(err)
	}
	if protected.State != recommendations.StateProtected {
		t.Fatalf("protected recommendation changed: %#v", protected)
	}
	if _, ok := byID["other_campaign"]; !ok {
		t.Fatalf("unrelated campaign recommendation was removed: %#v", byID)
	}
	if _, ok := byID["fresh_match"]; !ok {
		t.Fatalf("fresh recommendation missing: %#v", byID)
	}
}
