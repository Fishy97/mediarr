package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

func TestRecommendationIgnoreAndRestoreRoutes(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.ReplaceRecommendations([]recommendations.Recommendation{
		{
			ID:              "rec_1",
			Action:          recommendations.ActionReviewDuplicate,
			Title:           "Review duplicate media",
			Explanation:     "Multiple files resolve to the same catalog item.",
			SpaceSavedBytes: 10,
			Confidence:      0.9,
			Source:          "rule:test",
			AffectedPaths:   []string{"/media/a.mkv"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(Deps{Store: store})
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/rec_1/ignore", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("ignore status = %d, want 200", res.Code)
	}
	recs, err := store.ListRecommendations()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 {
		t.Fatalf("recommendations after ignore = %#v", recs)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/rec_1/restore", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("restore status = %d, want 200", res.Code)
	}
	recs, err = store.ListRecommendations()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("recommendations after restore = %d, want 1", len(recs))
	}
}
