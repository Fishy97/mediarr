package database

import (
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/recommendations"
	"github.com/Fishy97/mediarr/backend/internal/stewardship"
)

func TestStewardshipRequestSourcesPersistAndRedactSecrets(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	enabled := true
	source, err := store.UpsertRequestSource(stewardship.RequestSourceInput{
		Kind:    "seerr",
		Name:    "Jellyseerr",
		BaseURL: "http://seerr.local:5055/",
		APIKey:  "seerr-secret-abcd",
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if source.ID != "seerr" || source.BaseURL != "http://seerr.local:5055" || !source.APIKeyConfigured || source.APIKeyLast4 != "abcd" {
		t.Fatalf("source = %#v", source)
	}
	if source.APIKey != "" {
		t.Fatal("redacted source exposed api key")
	}

	secrets, err := store.ListRequestSourceSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if len(secrets) != 1 || secrets[0].APIKey != "seerr-secret-abcd" {
		t.Fatalf("secrets = %#v", secrets)
	}
}

func TestStewardshipRequestSignalsReplacePerSource(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	signals := []stewardship.RequestSignal{
		{SourceID: "seerr", ExternalRequestID: "1", MediaType: "movie", Title: "Arrival", Status: "approved", Availability: "available", ProviderIDs: map[string]string{"tmdb": "329865"}, RequestedBy: "Alex", EstimatedBytes: 42},
	}
	if err := store.ReplaceRequestSignals("seerr", signals); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceRequestSignals("other", []stewardship.RequestSignal{{SourceID: "other", ExternalRequestID: "2", MediaType: "tv", Title: "Station Eleven", Status: "approved", ProviderIDs: map[string]string{"tvdb": "1"}}}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceRequestSignals("seerr", []stewardship.RequestSignal{{SourceID: "seerr", ExternalRequestID: "3", MediaType: "movie", Title: "Blade Runner", Status: "pending", ProviderIDs: map[string]string{"tmdb": "78"}}}); err != nil {
		t.Fatal(err)
	}

	got, err := store.ListRequestSignals("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("signals = %#v", got)
	}
	seerr, err := store.ListRequestSignals("seerr")
	if err != nil {
		t.Fatal(err)
	}
	if len(seerr) != 1 || seerr[0].ExternalRequestID != "3" || seerr[0].ProviderIDs["tmdb"] != "78" {
		t.Fatalf("seerr signals = %#v", seerr)
	}
}

func TestStewardshipCollectionPublicationsRoundTrip(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	plan := stewardship.PlanLeavingSoonPublication(stewardship.PublicationInput{
		CampaignID:      "cmp_1",
		ServerID:        "jellyfin",
		CollectionTitle: "Leaving Soon",
		Items:           []stewardship.PublicationCandidate{{ExternalItemID: "jf_1", Title: "Arrival", Verification: "local_verified", EstimatedBytes: 42}},
	})
	saved, err := store.RecordCollectionPublication(plan)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == "" || saved.PublishableItems != 1 || saved.Status != "preview" {
		t.Fatalf("saved publication = %#v", saved)
	}
	published, err := store.MarkCollectionPublicationPublished(saved.ID, "published", "")
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != "published" || published.PublishedAt.IsZero() {
		t.Fatalf("published = %#v", published)
	}
	list, err := store.ListCollectionPublications("cmp_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != saved.ID {
		t.Fatalf("list = %#v", list)
	}
}

func TestStewardshipNotificationsAndProtectionRequests(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	notification, err := store.CreateNotification(stewardship.Notification{Title: "Campaign completed", EventType: "campaign.run"})
	if err != nil {
		t.Fatal(err)
	}
	if notification.ID == "" || notification.Read {
		t.Fatalf("notification = %#v", notification)
	}
	read, err := store.MarkNotificationRead(notification.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !read.Read || read.ReadAt.IsZero() {
		t.Fatalf("read notification = %#v", read)
	}

	request, err := store.CreateProtectionRequest(stewardship.ProtectionRequest{RecommendationID: "rec_1", Title: "Arrival", RequestedBy: "Alex"})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := store.DecideProtectionRequest(request.ID, true, "admin", "family favorite")
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != "approved" || approved.DecisionBy != "admin" {
		t.Fatalf("approved = %#v", approved)
	}
}

func TestStewardshipStorageLedgerFromRecommendations(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.ReplaceRecommendations([]recommendations.Recommendation{
		{ID: "local", Action: recommendations.ActionReviewInactiveMovie, State: recommendations.StateNew, Title: "Local", SpaceSavedBytes: 100, Confidence: 0.9, Source: "rule:test", AffectedPaths: []string{"/media/a.mkv"}, Verification: "local_verified", Evidence: map[string]string{"verifiedSavingsBytes": "100"}},
		{ID: "server", Action: recommendations.ActionReviewInactiveMovie, State: recommendations.StateNew, Title: "Server", SpaceSavedBytes: 80, Confidence: 0.7, Source: "rule:test", AffectedPaths: []string{"/server/b.mkv"}, Verification: "server_reported"},
		{ID: "protected", Action: recommendations.ActionReviewInactiveMovie, State: recommendations.StateProtected, Title: "Protected", SpaceSavedBytes: 30, Confidence: 0.8, Source: "rule:test", AffectedPaths: []string{"/media/c.mkv"}, Verification: "local_verified", Evidence: map[string]string{"verifiedSavingsBytes": "30"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceRequestSignals("seerr", []stewardship.RequestSignal{{SourceID: "seerr", ExternalRequestID: "1", MediaType: "movie", Title: "Requested", EstimatedBytes: 25, ProviderIDs: map[string]string{"tmdb": "1"}}}); err != nil {
		t.Fatal(err)
	}

	ledger, err := store.StorageLedger()
	if err != nil {
		t.Fatal(err)
	}
	if ledger.LocallyVerifiedBytes != 100 || ledger.ServerReportedBytes != 80 || ledger.ProtectedBytes != 30 || ledger.RequestedMediaBytes != 25 {
		t.Fatalf("ledger = %#v", ledger)
	}
}

func TestStewardshipTautulliSyncJobRoundTrip(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	completed := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	job := stewardship.TautulliSyncJob{ID: "tt_1", Status: "completed", ItemsImported: 10, Cursor: "1776700000", StartedAt: completed.Add(-time.Minute), CompletedAt: completed}
	if err := store.RecordTautulliSyncJob(job); err != nil {
		t.Fatal(err)
	}
	got, err := store.LatestTautulliSyncJob()
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "tt_1" || got.ItemsImported != 10 || got.Cursor != "1776700000" || !got.CompletedAt.Equal(completed) {
		t.Fatalf("job = %#v", got)
	}
}
