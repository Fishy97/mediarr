package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/audit"
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

func TestRecommendationEvidenceProtectAndAcceptRoutes(t *testing.T) {
	configDir := t.TempDir()
	logger, err := audit.New(configDir)
	if err != nil {
		t.Fatal(err)
	}
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.ReplaceRecommendations([]recommendations.Recommendation{
		{
			ID:              "rec_trust",
			Action:          recommendations.ActionReviewInactiveMovie,
			Title:           "Review inactive movie",
			Explanation:     "This movie has not been watched recently.",
			SpaceSavedBytes: 42_000_000_000,
			Confidence:      0.91,
			Source:          "rule:activity-inactive-movie",
			AffectedPaths:   []string{"/media/movies/Arrival (2016).mkv"},
			ServerID:        "jellyfin",
			ExternalItemID:  "item_1",
			PlayCount:       2,
			UniqueUsers:     1,
			FavoriteCount:   0,
			Verification:    "local_verified",
			Evidence: map[string]string{
				"inactiveDays":  "800",
				"thresholdDays": "540",
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(Deps{Store: store, Audit: logger})
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/recommendations/rec_trust/evidence", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("evidence status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var body struct {
		Data recommendations.Evidence `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Data.State != recommendations.StateNew || body.Data.Storage.Verification != "local_verified" || body.Data.Activity.PlayCount != 2 {
		t.Fatalf("evidence = %#v", body.Data)
	}
	if body.Data.Source.Rule != "rule:activity-inactive-movie" || len(body.Data.Proof) == 0 {
		t.Fatalf("evidence source/proof = %#v", body.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/rec_trust/accept-manual", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("accept status = %d, want 200: %s", res.Code, res.Body.String())
	}
	rec, err := store.GetRecommendation("rec_trust")
	if err != nil {
		t.Fatal(err)
	}
	if rec.State != recommendations.StateAcceptedForManualAction {
		t.Fatalf("accepted state = %s", rec.State)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/rec_trust/protect", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("protect status = %d, want 200: %s", res.Code, res.Body.String())
	}
	open, err := store.ListRecommendations()
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 0 {
		t.Fatalf("protected recommendation should leave open queue: %#v", open)
	}
	auditBytes, err := os.ReadFile(filepath.Join(configDir, "audit", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	auditText := string(auditBytes)
	if !strings.Contains(auditText, "recommendation.accepted_for_manual_action") || !strings.Contains(auditText, "recommendation.protected") {
		t.Fatalf("audit log missing state transitions: %s", auditText)
	}
}
