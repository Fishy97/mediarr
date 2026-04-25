package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fishy97/mediaar/backend/internal/recommendations"
)

func TestServerExposesHealthAndRecommendationAPIs(t *testing.T) {
	server := NewServer(Deps{
		Recommendations: []recommendations.Recommendation{
			{
				ID:              "rec_1",
				Action:          recommendations.ActionReviewDuplicate,
				Title:           "Review duplicate media",
				Explanation:     "Keep the better copy and review the smaller duplicate.",
				SpaceSavedBytes: 8_000_000_000,
				AffectedPaths:   []string{"/media/a.mkv", "/media/b.mkv"},
				Confidence:      0.92,
				Source:          "rule:duplicate-canonical-key",
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	res = httptest.NewRecorder()
	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("recommendations status = %d, want 200", res.Code)
	}

	var body struct {
		Data []recommendations.Recommendation `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 1 || body.Data[0].SpaceSavedBytes == 0 {
		t.Fatalf("unexpected recommendations body: %#v", body)
	}
}

func TestServerDoesNotExposeDestructiveMediaDeleteEndpoint(t *testing.T) {
	server := NewServer(Deps{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/media/files/example", nil)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed && res.Code != http.StatusNotFound {
		t.Fatalf("DELETE media returned %d; expected not found or method not allowed", res.Code)
	}
}
