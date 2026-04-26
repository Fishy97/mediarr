package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/acceptance"
	"github.com/Fishy97/mediarr/backend/internal/auth"
	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/integrations"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

func TestJellyfinSyncRoutePersistsNormalizedActivity(t *testing.T) {
	jellyfin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "jellyfin-key" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin Test"}`))
		case "/Users":
			_, _ = w.Write([]byte(`[{"Id":"user_1","Name":"Alex"}]`))
		case "/Items":
			_, _ = w.Write([]byte(`{
				"Items": [{
					"Id": "item_1",
					"Name": "Arrival",
					"Type": "Movie",
					"ProductionYear": 2016,
					"Path": "/media/movies/Arrival (2016).mkv",
					"ProviderIds": {"Tmdb":"329865"},
					"MediaSources": [{"Path":"/media/movies/Arrival (2016).mkv","Size":42000000000,"Container":"mkv"}],
					"UserData": {"PlayCount":1,"LastPlayedDate":"2025-01-02T03:04:05Z","Played":true}
				}],
				"TotalRecordCount": 1
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer jellyfin.Close()

	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := NewServer(Deps{
		Store: store,
		IntegrationOptions: integrations.Options{
			JellyfinURL: jellyfin.URL,
			JellyfinKey: "jellyfin-key",
		},
	})
	handler := auth.Middleware{AdminToken: "secret"}.Wrap(server)

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/integrations/jellyfin/sync", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated sync status = %d, want 401", res.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/jellyfin/sync", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("sync status = %d, want 202: %s", res.Code, res.Body.String())
	}
	var syncBody struct {
		Data database.MediaSyncJob `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&syncBody); err != nil {
		t.Fatal(err)
	}
	if syncBody.Data.Status != "queued" || syncBody.Data.ID == "" {
		t.Fatalf("sync body = %#v", syncBody.Data)
	}
	job := waitForJobStatus(t, store, syncBody.Data.ID, "completed")
	if job.ItemsImported != 1 {
		t.Fatalf("completed sync job = %#v", job)
	}

	items, err := store.ListMediaServerItems(database.MediaServerItemFilter{ServerID: "jellyfin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "Arrival" {
		t.Fatalf("items = %#v", items)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/activity/rollups", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("rollups status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var rollupBody struct {
		Data []database.MediaActivityRollup `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&rollupBody); err != nil {
		t.Fatal(err)
	}
	if len(rollupBody.Data) != 1 || rollupBody.Data[0].PlayCount != 1 {
		t.Fatalf("rollups = %#v", rollupBody.Data)
	}
}

func parseAPITestTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func TestEmbySyncRoutePersistsNormalizedActivity(t *testing.T) {
	emby := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "emby-key" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Emby Test"}`))
		case "/Users":
			_, _ = w.Write([]byte(`[{"Id":"user_1","Name":"Sam"}]`))
		case "/Items":
			_, _ = w.Write([]byte(`{
				"Items": [{
					"Id": "item_1",
					"Name": "Arrival",
					"Type": "Movie",
					"ProductionYear": 2016,
					"Path": "/media/movies/Arrival (2016).mkv",
					"ProviderIds": {"Tmdb":"329865"},
					"MediaSources": [{"Path":"/media/movies/Arrival (2016).mkv","Size":42000000000,"Container":"mkv"}],
					"UserData": {"PlayCount":1,"LastPlayedDate":"2025-01-02T03:04:05Z","Played":true}
				}],
				"TotalRecordCount": 1
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer emby.Close()

	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := NewServer(Deps{
		Store: store,
		IntegrationOptions: integrations.Options{
			EmbyURL: emby.URL,
			EmbyKey: "emby-key",
		},
	})

	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/integrations/emby/sync", nil))
	if res.Code != http.StatusAccepted {
		t.Fatalf("sync status = %d, want 202: %s", res.Code, res.Body.String())
	}
	var syncBody struct {
		Data database.MediaSyncJob `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&syncBody); err != nil {
		t.Fatal(err)
	}
	job := waitForJobStatus(t, store, syncBody.Data.ID, "completed")
	if job.ItemsImported != 1 || job.RollupsImported != 1 {
		t.Fatalf("completed sync job = %#v", job)
	}

	items, err := store.ListMediaServerItems(database.MediaServerItemFilter{ServerID: "emby"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "Arrival" {
		t.Fatalf("items = %#v", items)
	}
}

func TestJellyfinSyncCreatesActivityRecommendations(t *testing.T) {
	jellyfin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "jellyfin-key" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin Test"}`))
		case "/Users":
			_, _ = w.Write([]byte(`[{"Id":"user_1","Name":"Alex"}]`))
		case "/Items":
			_, _ = w.Write([]byte(`{
				"Items": [{
					"Id": "item_cold",
					"Name": "Cold Movie",
					"Type": "Movie",
					"ProductionYear": 2020,
					"Path": "/media/movies/Cold Movie (2020).mkv",
					"ProviderIds": {"Tmdb":"1"},
					"DateCreated": "2020-01-01T00:00:00Z",
					"MediaSources": [{"Path":"/media/movies/Cold Movie (2020).mkv","Size":64000000000,"Container":"mkv"}],
					"UserData": {"PlayCount":0,"Played":false,"IsFavorite":false}
				}],
				"TotalRecordCount": 1
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer jellyfin.Close()

	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := NewServer(Deps{
		Store: store,
		IntegrationOptions: integrations.Options{
			JellyfinURL: jellyfin.URL,
			JellyfinKey: "jellyfin-key",
		},
	})

	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/integrations/jellyfin/sync", nil))
	if res.Code != http.StatusAccepted {
		t.Fatalf("sync status = %d, want 202: %s", res.Code, res.Body.String())
	}
	var syncBody struct {
		Data database.MediaSyncJob `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&syncBody); err != nil {
		t.Fatal(err)
	}
	waitForJobStatus(t, store, syncBody.Data.ID, "completed")

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("recommendations status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var body struct {
		Data []recommendations.Recommendation `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("recommendations = %#v, want one activity recommendation", body.Data)
	}
	if body.Data[0].Action != recommendations.ActionReviewNeverWatchedMovie || body.Data[0].ServerID != "jellyfin" {
		t.Fatalf("recommendation = %#v", body.Data[0])
	}
	if body.Data[0].Destructive {
		t.Fatal("activity recommendation must be non-destructive")
	}
}

func TestJellyfinSyncCreatesServerReportedActivityRecommendations(t *testing.T) {
	jellyfin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "jellyfin-key" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin Test"}`))
		case "/Users":
			_, _ = w.Write([]byte(`[{"Id":"user_1","Name":"Alex"}]`))
		case "/Items":
			_, _ = w.Write([]byte(`{
				"Items": [{
					"Id": "item_cold",
					"Name": "Cold Movie",
					"Type": "Movie",
					"ProductionYear": 2020,
					"Path": "/Volume1/Media/Movies/Cold Movie (2020).mkv",
					"ProviderIds": {"Tmdb":"1"},
					"DateCreated": "2020-01-01T00:00:00Z",
					"MediaSources": [{"Path":"/Volume1/Media/Movies/Cold Movie (2020).mkv","Size":64000000000,"Container":"mkv"}],
					"UserData": {"PlayCount":0,"Played":false,"IsFavorite":false}
				}],
				"TotalRecordCount": 1
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer jellyfin.Close()

	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := NewServer(Deps{
		Store: store,
		IntegrationOptions: integrations.Options{
			JellyfinURL: jellyfin.URL,
			JellyfinKey: "jellyfin-key",
		},
	})

	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/integrations/jellyfin/sync", nil))
	if res.Code != http.StatusAccepted {
		t.Fatalf("sync status = %d, want 202: %s", res.Code, res.Body.String())
	}
	var syncBody struct {
		Data database.MediaSyncJob `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&syncBody); err != nil {
		t.Fatal(err)
	}
	waitForJobStatus(t, store, syncBody.Data.ID, "completed")

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("recommendations status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var body struct {
		Data []recommendations.Recommendation `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("recommendations = %#v, want one server-reported activity recommendation", body.Data)
	}
	if body.Data[0].Verification != "server_reported" || body.Data[0].Confidence < 0.65 {
		t.Fatalf("recommendation evidence = %#v", body.Data[0])
	}
}

func TestJellyfinSyncReportsRecommendationGenerationPhase(t *testing.T) {
	jellyfin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "jellyfin-key" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin Test"}`))
		case "/Users":
			_, _ = w.Write([]byte(`[{"Id":"user_1","Name":"Alex"}]`))
		case "/Items":
			_, _ = w.Write([]byte(`{
				"Items": [{
					"Id": "item_cold",
					"Name": "Cold Movie",
					"Type": "Movie",
					"ProductionYear": 2020,
					"Path": "/Volume1/Media/Movies/Cold Movie (2020).mkv",
					"DateCreated": "2020-01-01T00:00:00Z",
					"MediaSources": [{"Path":"/Volume1/Media/Movies/Cold Movie (2020).mkv","Size":64000000000,"Container":"mkv"}],
					"UserData": {"PlayCount":0,"Played":false,"IsFavorite":false}
				}],
				"TotalRecordCount": 1
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer jellyfin.Close()

	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := NewServer(Deps{
		Store: store,
		IntegrationOptions: integrations.Options{
			JellyfinURL: jellyfin.URL,
			JellyfinKey: "jellyfin-key",
		},
	})

	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/integrations/jellyfin/sync", nil))
	if res.Code != http.StatusAccepted {
		t.Fatalf("sync status = %d, want 202: %s", res.Code, res.Body.String())
	}
	var syncBody struct {
		Data database.MediaSyncJob `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&syncBody); err != nil {
		t.Fatal(err)
	}
	waitForJobStatus(t, store, syncBody.Data.ID, "completed")

	events, err := store.ListJobEvents(syncBody.Data.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Phase == "recommendations" && event.Message == "Generating evidence-based recommendations" && event.CurrentLabel == "Deterministic review rules" {
			return
		}
	}
	t.Fatalf("missing recommendation generation event: %#v", events)
}

func TestIntegrationDiagnosticsRouteSummarizesPersistedIngestion(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	completedAt := parseAPITestTime("2026-04-26T10:00:00Z")
	snapshot := database.MediaServerSnapshot{
		Server: database.MediaServer{ID: "jellyfin", Kind: "jellyfin", Name: "Jellyfin", BaseURL: "http://jellyfin.local", Status: "configured", LastSyncedAt: completedAt, UpdatedAt: completedAt},
		Libraries: []database.MediaServerLibrary{
			{ServerID: "jellyfin", ExternalID: "movies", Name: "Movies", Kind: "movie", ItemCount: 1},
		},
		Items: []database.MediaServerItem{
			{ServerID: "jellyfin", ExternalID: "movie_1", LibraryExternalID: "movies", Kind: "movie", Title: "Arrival", Path: "/mnt/movies/Arrival.mkv", DateCreated: completedAt.AddDate(-2, 0, 0), MatchConfidence: 0.9, UpdatedAt: completedAt},
		},
		Files: []database.MediaServerFile{
			{ServerID: "jellyfin", ItemExternalID: "movie_1", Path: "/mnt/movies/Arrival.mkv", LocalPath: "/media/movies/Arrival.mkv", SizeBytes: 42_000_000_000, Verification: "local_verified", MatchConfidence: 0.95},
		},
		Rollups: []database.MediaActivityRollup{
			{ServerID: "jellyfin", ItemExternalID: "movie_1", PlayCount: 1, UniqueUsers: 1, WatchedUsers: 1, LastPlayedAt: completedAt.AddDate(-2, 0, 0), UpdatedAt: completedAt},
		},
		Job: database.MediaSyncJob{ID: "sync_1", ServerID: "jellyfin", Status: "completed", ItemsImported: 1, RollupsImported: 1, UnmappedItems: 0, StartedAt: completedAt.Add(-time.Minute), CompletedAt: completedAt},
	}
	if err := store.ReplaceMediaServerSnapshot(snapshot); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceRecommendations([]recommendations.Recommendation{{
		ID:              "rec_1",
		Action:          recommendations.ActionReviewInactiveMovie,
		Title:           "Review inactive movie",
		Explanation:     "Arrival has not been watched recently.",
		SpaceSavedBytes: 42_000_000_000,
		Confidence:      0.9,
		Source:          "rule:activity-inactive-movie",
		AffectedPaths:   []string{"/media/movies/Arrival.mkv"},
		ServerID:        "jellyfin",
		ExternalItemID:  "movie_1",
		Verification:    "local_verified",
	}}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(Deps{Store: store})
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/integrations/jellyfin/diagnostics", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("diagnostics status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var body struct {
		Data acceptance.Report `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Summary.Movies != 1 || body.Data.Summary.ServerReportedBytes != 42_000_000_000 || body.Data.Summary.LocallyVerifiedBytes != 42_000_000_000 {
		t.Fatalf("diagnostics summary = %#v", body.Data.Summary)
	}
	if len(body.Data.TopRecommendations) != 1 || body.Data.TopRecommendations[0].ID != "rec_1" {
		t.Fatalf("diagnostics recommendations = %#v", body.Data.TopRecommendations)
	}
}

func TestIntegrationListRoutesRespectLimit(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := parseAPITestTime("2026-04-26T10:00:00Z")
	snapshot := database.MediaServerSnapshot{
		Server: database.MediaServer{ID: "jellyfin", Kind: "jellyfin", Name: "Jellyfin", BaseURL: "http://jellyfin.local", Status: "configured", LastSyncedAt: now, UpdatedAt: now},
		Items: []database.MediaServerItem{
			{ServerID: "jellyfin", ExternalID: "movie_1", Kind: "movie", Title: "Arrival", Path: "/nas/media/Arrival.mkv", UpdatedAt: now},
			{ServerID: "jellyfin", ExternalID: "movie_2", Kind: "movie", Title: "Blade Runner", Path: "/nas/media/Blade Runner.mkv", UpdatedAt: now},
			{ServerID: "jellyfin", ExternalID: "movie_3", Kind: "movie", Title: "Contact", Path: "/nas/media/Contact.mkv", UpdatedAt: now},
		},
		Files: []database.MediaServerFile{
			{ServerID: "jellyfin", ItemExternalID: "movie_1", Path: "/nas/media/Arrival.mkv", SizeBytes: 1, Verification: "server_reported", MatchConfidence: 0.68},
			{ServerID: "jellyfin", ItemExternalID: "movie_2", Path: "/nas/media/Blade Runner.mkv", SizeBytes: 1, Verification: "server_reported", MatchConfidence: 0.68},
			{ServerID: "jellyfin", ItemExternalID: "movie_3", Path: "/nas/media/Contact.mkv", SizeBytes: 1, Verification: "server_reported", MatchConfidence: 0.68},
		},
		Rollups: []database.MediaActivityRollup{
			{ServerID: "jellyfin", ItemExternalID: "movie_1", UpdatedAt: now},
			{ServerID: "jellyfin", ItemExternalID: "movie_2", UpdatedAt: now},
			{ServerID: "jellyfin", ItemExternalID: "movie_3", UpdatedAt: now},
		},
		Job: database.MediaSyncJob{ID: "sync_1", ServerID: "jellyfin", Status: "completed", ItemsImported: 3, RollupsImported: 3, UnmappedItems: 3, StartedAt: now, CompletedAt: now},
	}
	if err := store.ReplaceMediaServerSnapshot(snapshot); err != nil {
		t.Fatal(err)
	}
	server := NewServer(Deps{Store: store})

	for _, tc := range []struct {
		path string
		want int
	}{
		{path: "/api/v1/integrations/jellyfin/items?limit=2", want: 2},
		{path: "/api/v1/path-mappings/unmapped?serverId=jellyfin&limit=2", want: 2},
		{path: "/api/v1/activity/rollups?serverId=jellyfin&limit=1", want: 1},
	} {
		res := httptest.NewRecorder()
		server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if res.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200: %s", tc.path, res.Code, res.Body.String())
		}
		var body struct {
			Data []json.RawMessage `json:"data"`
		}
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Data) != tc.want {
			t.Fatalf("%s returned %d rows, want %d", tc.path, len(body.Data), tc.want)
		}
	}
}
