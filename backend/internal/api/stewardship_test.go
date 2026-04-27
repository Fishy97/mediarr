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
	"github.com/Fishy97/mediarr/backend/internal/stewardship"
)

func TestStewardshipAPIRequestSourceSyncAndSignals(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	seerr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/request" {
			t.Fatalf("path = %s, want /api/v1/request", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "seerr-key" {
			t.Fatalf("api key header = %q", r.Header.Get("X-Api-Key"))
		}
		_, _ = w.Write([]byte(`{"results":[{"id":100,"status":2,"createdAt":"2026-01-02T03:04:05Z","updatedAt":"2026-01-03T03:04:05Z","requestedBy":{"username":"alex"},"media":{"id":500,"tmdbId":329865,"status":5,"mediaType":"movie","title":"Arrival"}}]}`))
	}))
	defer seerr.Close()

	server := NewServer(Deps{Store: store})
	enabled := true
	body := jsonBody(t, stewardship.RequestSourceInput{Kind: "seerr", Name: "Jellyseerr", BaseURL: seerr.URL, APIKey: "seerr-key", Enabled: &enabled})
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPut, "/api/v1/request-sources/seerr", body))
	if res.Code != http.StatusOK {
		t.Fatalf("source save status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var saved struct {
		Data stewardship.RequestSource `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if !saved.Data.APIKeyConfigured || saved.Data.APIKeyLast4 != "-key" {
		t.Fatalf("saved source = %#v", saved.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/request-sources/seerr/sync", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("source sync status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var syncBody struct {
		Data struct {
			Imported int `json:"imported"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&syncBody); err != nil {
		t.Fatal(err)
	}
	if syncBody.Data.Imported != 1 {
		t.Fatalf("sync body = %#v", syncBody.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/request-signals?sourceId=seerr", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("signals status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var signals struct {
		Data []stewardship.RequestSignal `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&signals); err != nil {
		t.Fatal(err)
	}
	if len(signals.Data) != 1 || signals.Data[0].Title != "Arrival" || signals.Data[0].ProviderIDs["tmdb"] != "329865" {
		t.Fatalf("signals = %#v", signals.Data)
	}
}

func TestStewardshipAPITautulliSyncEnrichesPlexActivity(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	seedPlexSnapshot(t, store)

	tautulli := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2" {
			t.Fatalf("unexpected tautulli request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		if r.URL.Query().Get("apikey") != "tautulli-key" {
			t.Fatalf("tautulli api key = %q", r.URL.Query().Get("apikey"))
		}
		if r.URL.Query().Get("cmd") == "status" {
			_, _ = w.Write([]byte(`{"response":{"result":"success"}}`))
			return
		}
		if r.URL.Query().Get("cmd") != "get_history" {
			t.Fatalf("unexpected tautulli command: %s", r.URL.Query().Get("cmd"))
		}
		_, _ = w.Write([]byte(`{"response":{"result":"success","data":{"data":[{"rating_key":"plex_1","user":"alex","date":1776700000,"percent_complete":95}]}}}`))
	}))
	defer tautulli.Close()

	server := NewServer(Deps{Store: store})
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPut, "/api/v1/integration-settings/tautulli", jsonBody(t, database.IntegrationSettingInput{
		BaseURL: tautulli.URL,
		APIKey:  "tautulli-key",
	})))
	if res.Code != http.StatusOK {
		t.Fatalf("tautulli setting status = %d, want 200: %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/integrations/tautulli/sync", nil))
	if res.Code != http.StatusAccepted {
		t.Fatalf("tautulli sync status = %d, want 202: %s", res.Code, res.Body.String())
	}
	var syncBody struct {
		Data database.Job `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&syncBody); err != nil {
		t.Fatal(err)
	}
	job := waitForJobStatus(t, store, syncBody.Data.ID, "completed")
	if job.ItemsImported != 1 {
		t.Fatalf("job = %#v", job)
	}
	rollups, err := store.ListMediaActivityRollups(database.MediaActivityRollupFilter{ServerID: "plex"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rollups) != 1 || rollups[0].PlayCount != 1 || rollups[0].UniqueUsers != 1 || rollups[0].LastPlayedAt.IsZero() {
		t.Fatalf("rollups = %#v", rollups)
	}
}

func TestStewardshipAPICampaignTemplatesWhatIfAndPublicationPreview(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	seedCampaignActivity(t, store, now)
	if err := store.ReplaceRequestSignals("seerr", []stewardship.RequestSignal{{SourceID: "seerr", ExternalRequestID: "1", MediaType: "movie", Title: "Arrival", Status: "approved", Availability: "available", ProviderIDs: map[string]string{"tmdb": "329865"}, EstimatedBytes: 42_000_000_000}}); err != nil {
		t.Fatal(err)
	}
	server := NewServer(Deps{Store: store})

	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/campaign-templates", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("templates status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var templates struct {
		Data []stewardship.CampaignTemplate `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&templates); err != nil {
		t.Fatal(err)
	}
	if len(templates.Data) < 4 {
		t.Fatalf("templates = %#v", templates.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/campaign-templates/cold-movies/create", nil))
	if res.Code != http.StatusCreated {
		t.Fatalf("template create status = %d, want 201: %s", res.Code, res.Body.String())
	}
	var created struct {
		Data campaigns.Campaign `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/"+created.Data.ID+"/what-if", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("what-if status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var whatIf struct {
		Data stewardship.WhatIfSimulation `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&whatIf); err != nil {
		t.Fatal(err)
	}
	if whatIf.Data.Matched != 1 || whatIf.Data.EstimatedBytes != 42_000_000_000 || whatIf.Data.RequestConflicts != 1 {
		t.Fatalf("what-if = %#v", whatIf.Data)
	}

	previewReq := bytes.NewBufferString(`{"serverId":"jellyfin","collectionTitle":"Leaving Soon","minimumVerification":"local_verified"}`)
	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/"+created.Data.ID+"/publish-preview", previewReq))
	if res.Code != http.StatusOK {
		t.Fatalf("publish preview status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var preview struct {
		Data stewardship.PublicationPlan `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}
	if preview.Data.DryRun != true || preview.Data.PublishableItems != 1 || preview.Data.Status != "preview" {
		t.Fatalf("preview = %#v", preview.Data)
	}
}

func TestStewardshipAPILedgerNotificationsAndProtectionRequests(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.ReplaceRecommendations(seedRecommendationsForStewardship()); err != nil {
		t.Fatal(err)
	}
	notification, err := store.CreateNotification(stewardship.Notification{Title: "Sync complete", EventType: "tautulli.sync"})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(Deps{Store: store})

	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/storage-ledger", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("ledger status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var ledger struct {
		Data stewardship.StorageLedger `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&ledger); err != nil {
		t.Fatal(err)
	}
	if ledger.Data.LocallyVerifiedBytes != 100 {
		t.Fatalf("ledger = %#v", ledger.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("notifications status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var notifications struct {
		Data []stewardship.Notification `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&notifications); err != nil {
		t.Fatal(err)
	}
	if len(notifications.Data) != 1 || notifications.Data[0].ID != notification.ID {
		t.Fatalf("notifications = %#v", notifications.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+notification.ID+"/read", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("read notification status = %d, want 200: %s", res.Code, res.Body.String())
	}

	request := stewardship.ProtectionRequest{RecommendationID: "rec_local", Title: "Arrival", RequestedBy: "alex", Reason: "Family favourite"}
	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/protection-requests", jsonBody(t, request)))
	if res.Code != http.StatusCreated {
		t.Fatalf("protection create status = %d, want 201: %s", res.Code, res.Body.String())
	}
	var created struct {
		Data stewardship.ProtectionRequest `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/protection-requests/"+created.Data.ID+"/approve", bytes.NewBufferString(`{"decisionBy":"admin","note":"protected"}`)))
	if res.Code != http.StatusOK {
		t.Fatalf("protection approve status = %d, want 200: %s", res.Code, res.Body.String())
	}
	approved, err := store.GetProtectionRequest(created.Data.ID)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != "approved" {
		t.Fatalf("approved = %#v", approved)
	}
}

func jsonBody(t *testing.T, value any) *bytes.Reader {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(body)
}

func seedPlexSnapshot(t *testing.T, store *database.Store) {
	t.Helper()
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	if err := store.ReplaceMediaServerSnapshot(database.MediaServerSnapshot{
		Server: database.MediaServer{ID: "plex", Kind: "plex", Name: "Plex", BaseURL: "http://plex.local", Status: "configured"},
		Libraries: []database.MediaServerLibrary{
			{ServerID: "plex", ExternalID: "1", Name: "Movies", Kind: "movie", ItemCount: 1},
		},
		Items: []database.MediaServerItem{
			{ServerID: "plex", ExternalID: "plex_1", LibraryExternalID: "1", Kind: "movie", Title: "Arrival", MatchConfidence: 0.9, UpdatedAt: now},
		},
		Files: []database.MediaServerFile{
			{ServerID: "plex", ItemExternalID: "plex_1", Path: "/data/movies/Arrival.mkv", SizeBytes: 42_000_000_000, Verification: "local_verified", MatchConfidence: 0.9},
		},
		Job: database.MediaSyncJob{ID: "sync_plex", ServerID: "plex", Status: "completed", ItemsImported: 1, StartedAt: now.Add(-time.Minute), CompletedAt: now},
	}); err != nil {
		t.Fatal(err)
	}
}

func seedRecommendationsForStewardship() []recommendations.Recommendation {
	return []recommendations.Recommendation{
		{ID: "rec_local", Action: recommendations.ActionReviewInactiveMovie, State: recommendations.StateNew, Title: "Arrival", Explanation: "Review", SpaceSavedBytes: 100, Confidence: 0.9, Source: "rule:test", AffectedPaths: []string{"/media/arrival.mkv"}, Verification: "local_verified", Evidence: map[string]string{"verifiedSavingsBytes": "100"}},
	}
}
