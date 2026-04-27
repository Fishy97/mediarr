package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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

func TestAIRecommendationEnrichmentSkipsLargeBatches(t *testing.T) {
	var generateCalls atomic.Int32
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:0.6b"}]}`))
		case "/api/generate":
			generateCalls.Add(1)
			_, _ = w.Write([]byte(`{"response":"{\"rationale\":\"Review this candidate using the verified facts shown in Mediarr.\",\"tags\":[\"review\"],\"confidence\":0.7}"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ollama.Close()

	server := NewServer(Deps{AI: &ai.OllamaClient{BaseURL: ollama.URL, Model: "qwen3:0.6b"}})
	recs := make([]recommendations.Recommendation, 0, maxAIEnrichedRecommendations+4)
	for index := 0; index < maxAIEnrichedRecommendations+4; index++ {
		recs = append(recs, recommendations.Recommendation{
			ID:              "rec_large_sync_" + string(rune('a'+index)),
			Action:          recommendations.ActionReviewDuplicate,
			Title:           "Review candidate",
			Explanation:     "Large media-server sync generated many advisory candidates.",
			SpaceSavedBytes: int64(1000 + index),
			Confidence:      0.68,
			Source:          "rule:activity-never-watched",
		})
	}

	enriched := server.enrichRecommendationsWithAI(context.Background(), recs)
	if generateCalls.Load() != 0 {
		t.Fatalf("generate calls = %d, want 0 for a large recommendation batch", generateCalls.Load())
	}
	var aiTagged int
	for _, rec := range enriched {
		if rec.AISource != "" {
			aiTagged++
		}
	}
	if aiTagged != 0 {
		t.Fatalf("AI-enriched recommendations = %d, want 0 for a large recommendation batch", aiTagged)
	}
}
