package campaigns

import (
	"strings"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

func TestRuleOperatorsEvaluateCandidates(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	candidate := Candidate{
		Key:                   "jellyfin:item_1:/media/movies/Arrival.mkv",
		ServerID:              "jellyfin",
		ExternalItemID:        "item_1",
		Title:                 "Arrival",
		Kind:                  "movie",
		LibraryName:           "Movies",
		Verification:          "local_verified",
		EstimatedSavingsBytes: 42_000_000_000,
		VerifiedSavingsBytes:  42_000_000_000,
		Confidence:            0.92,
		AddedAt:               now.AddDate(-1, 0, 0),
		LastPlayedAt:          now.AddDate(-2, 0, 0),
		PlayCount:             3,
		UniqueUsers:           1,
		FavoriteCount:         0,
		AffectedPaths:         []string{"/media/movies/Arrival.mkv"},
	}

	tests := []struct {
		name string
		rule Rule
		want bool
	}{
		{name: "equals", rule: Rule{Field: FieldKind, Operator: OperatorEquals, Value: "movie"}, want: true},
		{name: "not equals", rule: Rule{Field: FieldKind, Operator: OperatorNotEquals, Value: "series"}, want: true},
		{name: "in", rule: Rule{Field: FieldVerification, Operator: OperatorIn, Values: []string{"path_mapped", "local_verified"}}, want: true},
		{name: "not in", rule: Rule{Field: FieldLibraryName, Operator: OperatorNotIn, Values: []string{"Anime", "Kids"}}, want: true},
		{name: "greater equal bytes", rule: Rule{Field: FieldEstimatedSavingsBytes, Operator: OperatorGreaterOrEqual, Value: "40000000000"}, want: true},
		{name: "greater equal days", rule: Rule{Field: FieldLastPlayedDays, Operator: OperatorGreaterOrEqual, Value: "540"}, want: true},
		{name: "less than activity", rule: Rule{Field: FieldPlayCount, Operator: OperatorLessThan, Value: "5"}, want: true},
		{name: "is not empty", rule: Rule{Field: FieldLastPlayedDays, Operator: OperatorIsNotEmpty}, want: true},
		{name: "is empty false", rule: Rule{Field: FieldLastPlayedDays, Operator: OperatorIsEmpty}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateRule(tt.rule, candidate, now)
			if got.Matched != tt.want {
				t.Fatalf("matched = %v, want %v; result=%#v", got.Matched, tt.want, got)
			}
			if got.Reason == "" {
				t.Fatalf("expected an explanatory reason for rule %#v", tt.rule)
			}
		})
	}
}

func TestSimulateSuppressesRiskyCampaignMatches(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	campaign := Campaign{
		ID:                  "campaign_cold_movies",
		Name:                "Cold Movies",
		Enabled:             false,
		TargetKinds:         []string{"movie"},
		RequireAllRules:     true,
		MinimumConfidence:   0.75,
		MinimumStorageBytes: 10_000_000_000,
		Rules: []Rule{
			{Field: FieldLastPlayedDays, Operator: OperatorGreaterOrEqual, Value: "365"},
			{Field: FieldEstimatedSavingsBytes, Operator: OperatorGreaterOrEqual, Value: "10000000000"},
		},
	}
	candidates := []Candidate{
		{
			Key:                   "jellyfin:arrival",
			ServerID:              "jellyfin",
			ExternalItemID:        "arrival",
			Title:                 "Arrival",
			Kind:                  "movie",
			Verification:          "local_verified",
			EstimatedSavingsBytes: 42_000_000_000,
			VerifiedSavingsBytes:  42_000_000_000,
			Confidence:            0.91,
			LastPlayedAt:          now.AddDate(-2, 0, 0),
			PlayCount:             2,
			AffectedPaths:         []string{"/media/movies/Arrival.mkv"},
		},
		{
			Key:                   "jellyfin:favorite",
			Title:                 "Protected Favorite",
			Kind:                  "movie",
			Verification:          "local_verified",
			EstimatedSavingsBytes: 30_000_000_000,
			VerifiedSavingsBytes:  30_000_000_000,
			Confidence:            0.92,
			LastPlayedAt:          now.AddDate(-3, 0, 0),
			FavoriteCount:         1,
			AffectedPaths:         []string{"/media/movies/Favorite.mkv"},
		},
		{
			Key:                   "jellyfin:weak",
			Title:                 "Weak Evidence",
			Kind:                  "movie",
			Verification:          "unmapped",
			EstimatedSavingsBytes: 20_000_000_000,
			Confidence:            0.44,
			LastPlayedAt:          now.AddDate(-3, 0, 0),
			AffectedPaths:         []string{"/media/movies/Weak.mkv"},
		},
	}

	result := Simulate(campaign, candidates, now)

	if result.Enabled {
		t.Fatal("disabled campaign simulation should keep enabled=false")
	}
	if result.Matched != 1 || result.Suppressed != 2 {
		t.Fatalf("matched/suppressed = %d/%d, want 1/2", result.Matched, result.Suppressed)
	}
	if result.TotalEstimatedSavingsBytes != 42_000_000_000 || result.TotalVerifiedSavingsBytes != 42_000_000_000 {
		t.Fatalf("savings = %d/%d", result.TotalEstimatedSavingsBytes, result.TotalVerifiedSavingsBytes)
	}
	suppressed := map[string][]string{}
	for _, item := range result.Items {
		if item.Suppressed {
			suppressed[item.Candidate.Title] = item.SuppressionReasons
		}
	}
	if !containsReason(suppressed["Protected Favorite"], "favorite_or_protected") {
		t.Fatalf("favorite suppression missing: %#v", suppressed["Protected Favorite"])
	}
	if !containsReason(suppressed["Weak Evidence"], "low_evidence_confidence") || !containsReason(suppressed["Weak Evidence"], "below_campaign_confidence") {
		t.Fatalf("weak evidence suppression missing: %#v", suppressed["Weak Evidence"])
	}
}

func TestRecommendationForMatchIsAdvisoryAndEvidenceRich(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	campaign := Campaign{ID: "campaign_archive", Name: "Deep Archive"}
	item := ResultItem{
		Candidate: Candidate{
			Key:                   "jellyfin:item_1:/media/movies/Arrival.mkv",
			ServerID:              "jellyfin",
			ExternalItemID:        "item_1",
			Title:                 "Arrival",
			Kind:                  "movie",
			Verification:          "local_verified",
			EstimatedSavingsBytes: 42_000_000_000,
			VerifiedSavingsBytes:  42_000_000_000,
			Confidence:            0.92,
			LastPlayedAt:          now.AddDate(-2, 0, 0),
			PlayCount:             3,
			UniqueUsers:           1,
			AffectedPaths:         []string{"/media/movies/Arrival.mkv"},
		},
		MatchedRules: []RuleResult{
			{Rule: Rule{Field: FieldLastPlayedDays, Operator: OperatorGreaterOrEqual, Value: "365"}, Matched: true, Reason: "matched"},
		},
	}

	rec := RecommendationForMatch(campaign, "run_1", item)

	if rec.Action != recommendations.ActionReviewCampaignMatch {
		t.Fatalf("action = %s, want %s", rec.Action, recommendations.ActionReviewCampaignMatch)
	}
	if rec.Destructive {
		t.Fatal("campaign recommendations must remain suggest-only")
	}
	if rec.Source != "campaign:campaign_archive" {
		t.Fatalf("source = %q", rec.Source)
	}
	for _, key := range []string{"campaignId", "campaignName", "campaignRunId", "matchedRules", "estimatedSavingsBytes", "verifiedSavingsBytes"} {
		if strings.TrimSpace(rec.Evidence[key]) == "" {
			t.Fatalf("missing evidence %q in %#v", key, rec.Evidence)
		}
	}
	if rec.Evidence["campaignId"] != "campaign_archive" || rec.Evidence["campaignName"] != "Deep Archive" {
		t.Fatalf("campaign evidence = %#v", rec.Evidence)
	}
}

func containsReason(reasons []string, want string) bool {
	for _, reason := range reasons {
		if reason == want {
			return true
		}
	}
	return false
}
