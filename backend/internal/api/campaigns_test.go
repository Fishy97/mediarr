package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/campaigns"
	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

func TestCampaignAPICreateListSimulateAndRun(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	seedCampaignActivity(t, store, now)
	server := NewServer(Deps{Store: store})

	campaign := campaigns.Campaign{
		ID:                  "campaign_cold_movies",
		Name:                "Cold Movies",
		Description:         "Movies that have been cold for at least one year.",
		Enabled:             true,
		TargetKinds:         []string{"movie"},
		RequireAllRules:     true,
		MinimumConfidence:   0.7,
		MinimumStorageBytes: 10_000_000_000,
		Rules: []campaigns.Rule{
			{Field: campaigns.FieldLastPlayedDays, Operator: campaigns.OperatorGreaterOrEqual, Value: "365"},
		},
	}
	createBody := campaignRequest(t, campaign)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", createBody))
	if res.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201: %s", res.Code, res.Body.String())
	}
	var created struct {
		Data campaigns.Campaign `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Data.ID != campaign.ID || created.Data.Name != campaign.Name {
		t.Fatalf("created = %#v", created.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/campaigns", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var list struct {
		Data []campaigns.Campaign `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Data) != 1 || list.Data[0].ID != campaign.ID {
		t.Fatalf("campaign list = %#v", list.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/"+campaign.ID+"/simulate", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("simulate status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var simulation struct {
		Data campaigns.Result `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&simulation); err != nil {
		t.Fatal(err)
	}
	if simulation.Data.Matched != 1 || simulation.Data.Suppressed != 0 || simulation.Data.TotalEstimatedSavingsBytes != 42_000_000_000 {
		t.Fatalf("simulation = %#v", simulation.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/"+campaign.ID+"/run", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("run status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var runBody struct {
		Data struct {
			Run    campaigns.Run    `json:"run"`
			Result campaigns.Result `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&runBody); err != nil {
		t.Fatal(err)
	}
	if runBody.Data.Run.Status != "completed" || runBody.Data.Run.Matched != 1 {
		t.Fatalf("run = %#v", runBody.Data.Run)
	}
	recs, err := store.ListRecommendations()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Action != recommendations.ActionReviewCampaignMatch || recs[0].Destructive {
		t.Fatalf("campaign recommendations = %#v", recs)
	}
	if recs[0].Evidence["campaignId"] != campaign.ID {
		t.Fatalf("recommendation evidence = %#v", recs[0].Evidence)
	}
}

func TestCampaignAPIListReturnsEmptyArrayOnFreshStore(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := NewServer(Deps{Store: store})

	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/campaigns", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200: %s", res.Code, res.Body.String())
	}
	if !bytes.Contains(res.Body.Bytes(), []byte(`"data":[]`)) {
		t.Fatalf("fresh campaigns list should be an empty array, got %s", res.Body.String())
	}
}

func TestCampaignAPIDeleteRemovesDefinitionOnly(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	campaign := campaigns.Campaign{ID: "campaign_delete", Name: "Delete definition only", Enabled: true}
	if _, err := store.UpsertCampaign(campaign); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceCampaignRecommendations(campaign.ID, []recommendations.Recommendation{
		{
			ID:              "campaign_rec",
			Action:          recommendations.ActionReviewCampaignMatch,
			Title:           "Existing campaign suggestion",
			Explanation:     "suggest-only",
			SpaceSavedBytes: 8_000_000_000,
			Confidence:      0.88,
			Source:          "campaign:" + campaign.ID,
			AffectedPaths:   []string{"/media/movie.mkv"},
			Destructive:     false,
		},
	}); err != nil {
		t.Fatal(err)
	}
	server := NewServer(Deps{Store: store})

	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodDelete, "/api/v1/campaigns/"+campaign.ID, nil))
	if res.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200: %s", res.Code, res.Body.String())
	}
	if _, err := store.GetCampaign(campaign.ID); err == nil {
		t.Fatal("campaign definition still exists after delete")
	}
	rec, err := store.GetRecommendation("campaign_rec")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Destructive || rec.Source != "campaign:"+campaign.ID {
		t.Fatalf("recommendation changed during campaign delete: %#v", rec)
	}
}

func campaignRequest(t *testing.T, campaign campaigns.Campaign) *bytes.Reader {
	t.Helper()
	body, err := json.Marshal(campaign)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(body)
}

func seedCampaignActivity(t *testing.T, store *database.Store, now time.Time) {
	t.Helper()
	snapshot := database.MediaServerSnapshot{
		Server: database.MediaServer{ID: "jellyfin", Kind: "jellyfin", Name: "Jellyfin", BaseURL: "http://jellyfin.local", Status: "configured"},
		Libraries: []database.MediaServerLibrary{
			{ServerID: "jellyfin", ExternalID: "movies", Name: "Movies", Kind: "movie", ItemCount: 1},
		},
		Items: []database.MediaServerItem{
			{
				ServerID:          "jellyfin",
				ExternalID:        "movie_1",
				LibraryExternalID: "movies",
				Kind:              "movie",
				Title:             "Arrival",
				Path:              "/mnt/movies/Arrival.mkv",
				DateCreated:       now.AddDate(-3, 0, 0),
				MatchConfidence:   0.91,
			},
		},
		Files: []database.MediaServerFile{
			{
				ServerID:        "jellyfin",
				ItemExternalID:  "movie_1",
				Path:            "/mnt/movies/Arrival.mkv",
				SizeBytes:       42_000_000_000,
				Verification:    "local_verified",
				MatchConfidence: 0.91,
			},
		},
		Rollups: []database.MediaActivityRollup{
			{ServerID: "jellyfin", ItemExternalID: "movie_1", PlayCount: 2, UniqueUsers: 1, LastPlayedAt: now.AddDate(-2, 0, 0), UpdatedAt: now},
		},
		Job: database.MediaSyncJob{ID: "sync_1", ServerID: "jellyfin", Status: "completed", StartedAt: now.Add(-time.Minute), CompletedAt: now},
	}
	if err := store.ReplaceMediaServerSnapshot(snapshot); err != nil {
		t.Fatal(err)
	}
}
