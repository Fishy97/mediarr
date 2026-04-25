package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/ai"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

func TestAIRationaleIsAdvisoryAndKeepsDeterministicRecommendationTruth(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:0.6b"}]}`))
		case "/api/generate":
			_, _ = w.Write([]byte(`{"response":"{\"rationale\":\"Keep the best edition and review both paths before reclaiming space.\",\"tags\":[\"duplicate\"],\"confidence\":0.81}"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ollama.Close()

	server := NewServer(Deps{AI: &ai.OllamaClient{BaseURL: ollama.URL, Model: "qwen3:0.6b"}})
	recs := []recommendations.Recommendation{
		{
			ID:              "rec_1",
			Action:          recommendations.ActionReviewDuplicate,
			Title:           "Review duplicate media",
			Explanation:     "Multiple files resolve to the same catalog item.",
			SpaceSavedBytes: 10,
			Confidence:      0.92,
			Source:          "rule:duplicate-canonical-key",
			AffectedPaths:   []string{"/media/a.mkv", "/media/b.mkv"},
			Destructive:     false,
		},
	}

	enriched := server.enrichRecommendationsWithAI(context.Background(), recs)
	if enriched[0].Source != "rule:duplicate-canonical-key" || enriched[0].Confidence != 0.92 || enriched[0].Destructive {
		t.Fatalf("AI must not alter deterministic recommendation truth: %#v", enriched[0])
	}
	if enriched[0].AIRationale == "" || enriched[0].AISource != "ollama:qwen3:0.6b" || enriched[0].AIConfidence != 0.81 {
		t.Fatalf("AI rationale not attached correctly: %#v", enriched[0])
	}
}
