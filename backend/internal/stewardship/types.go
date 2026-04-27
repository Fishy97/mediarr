package stewardship

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

type RequestSource struct {
	ID               string    `json:"id"`
	Kind             string    `json:"kind"`
	Name             string    `json:"name"`
	BaseURL          string    `json:"baseUrl,omitempty"`
	APIKey           string    `json:"-"`
	APIKeyConfigured bool      `json:"apiKeyConfigured"`
	APIKeyLast4      string    `json:"apiKeyLast4,omitempty"`
	Enabled          bool      `json:"enabled"`
	LastSyncedAt     time.Time `json:"lastSyncedAt,omitempty"`
	UpdatedAt        time.Time `json:"updatedAt,omitempty"`
}

type RequestSourceInput struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	BaseURL     string `json:"baseUrl"`
	APIKey      string `json:"apiKey"`
	ClearAPIKey bool   `json:"clearApiKey"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

type RequestSignal struct {
	SourceID          string            `json:"sourceId"`
	ExternalRequestID string            `json:"externalRequestId"`
	MediaType         string            `json:"mediaType"`
	ExternalMediaID   string            `json:"externalMediaId,omitempty"`
	Title             string            `json:"title,omitempty"`
	Status            string            `json:"status"`
	Availability      string            `json:"availability"`
	RequestedBy       string            `json:"requestedBy,omitempty"`
	ProviderIDs       map[string]string `json:"providerIds"`
	EstimatedBytes    int64             `json:"estimatedBytes,omitempty"`
	RequestedAt       time.Time         `json:"requestedAt,omitempty"`
	ApprovedAt        time.Time         `json:"approvedAt,omitempty"`
	AvailableAt       time.Time         `json:"availableAt,omitempty"`
	UpdatedAt         time.Time         `json:"updatedAt,omitempty"`
}

func (signal RequestSignal) MatchKey() string {
	mediaType := strings.ToLower(strings.TrimSpace(signal.MediaType))
	if mediaType == "" {
		mediaType = "unknown"
	}
	for _, provider := range []string{"tmdb", "tvdb", "imdb"} {
		if value := strings.TrimSpace(signal.ProviderIDs[provider]); value != "" {
			return mediaType + ":" + provider + ":" + value
		}
	}
	if external := strings.TrimSpace(signal.ExternalMediaID); external != "" {
		return mediaType + ":external:" + external
	}
	return mediaType + ":request:" + strings.TrimSpace(signal.ExternalRequestID)
}

type ActivityRollup struct {
	ServerID       string    `json:"serverId"`
	ItemExternalID string    `json:"itemExternalId"`
	PlayCount      int       `json:"playCount"`
	UniqueUsers    int       `json:"uniqueUsers"`
	WatchedUsers   int       `json:"watchedUsers"`
	FavoriteCount  int       `json:"favoriteCount"`
	LastPlayedAt   time.Time `json:"lastPlayedAt,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt,omitempty"`
	EvidenceSource string    `json:"evidenceSource,omitempty"`
}

type TautulliPlay struct {
	RatingKey       string    `json:"ratingKey"`
	User            string    `json:"user,omitempty"`
	PlayedAt        time.Time `json:"playedAt,omitempty"`
	PercentComplete int       `json:"percentComplete,omitempty"`
}

type TautulliSyncJob struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	ItemsImported int       `json:"itemsImported"`
	Cursor        string    `json:"cursor,omitempty"`
	Error         string    `json:"error,omitempty"`
	StartedAt     time.Time `json:"startedAt"`
	CompletedAt   time.Time `json:"completedAt,omitempty"`
}

type LedgerRecommendation struct {
	ID             string `json:"id"`
	State          string `json:"state"`
	EstimatedBytes int64  `json:"estimatedBytes"`
	VerifiedBytes  int64  `json:"verifiedBytes"`
	Verification   string `json:"verification"`
}

type LedgerInput struct {
	Recommendations []LedgerRecommendation `json:"recommendations"`
	RequestSignals  []RequestSignal        `json:"requestSignals"`
}

type StorageLedger struct {
	LocallyVerifiedBytes int64 `json:"locallyVerifiedBytes"`
	MappedEstimateBytes  int64 `json:"mappedEstimateBytes"`
	ServerReportedBytes  int64 `json:"serverReportedBytes"`
	BlockedUnmappedBytes int64 `json:"blockedUnmappedBytes"`
	ProtectedBytes       int64 `json:"protectedBytes"`
	AcceptedManualBytes  int64 `json:"acceptedManualBytes"`
	RequestedMediaBytes  int64 `json:"requestedMediaBytes"`
	TotalEstimatedBytes  int64 `json:"totalEstimatedBytes"`
	TotalVerifiedBytes   int64 `json:"totalVerifiedBytes"`
}

type CampaignSimulation struct {
	CampaignID          string `json:"campaignId"`
	Matched             int    `json:"matched"`
	Suppressed          int    `json:"suppressed"`
	EstimatedBytes      int64  `json:"estimatedBytes"`
	VerifiedBytes       int64  `json:"verifiedBytes"`
	BlockedUnmapped     int    `json:"blockedUnmapped"`
	ProtectionConflicts int    `json:"protectionConflicts"`
	RequestConflicts    int    `json:"requestConflicts"`
}

type WhatIfSimulation struct {
	Campaigns           int   `json:"campaigns"`
	Matched             int   `json:"matched"`
	Suppressed          int   `json:"suppressed"`
	EstimatedBytes      int64 `json:"estimatedBytes"`
	VerifiedBytes       int64 `json:"verifiedBytes"`
	BlockedUnmapped     int   `json:"blockedUnmapped"`
	ProtectionConflicts int   `json:"protectionConflicts"`
	RequestConflicts    int   `json:"requestConflicts"`
}

type PublicationInput struct {
	CampaignID          string                 `json:"campaignId"`
	ServerID            string                 `json:"serverId"`
	CollectionTitle     string                 `json:"collectionTitle"`
	MinimumVerification string                 `json:"minimumVerification"`
	Items               []PublicationCandidate `json:"items"`
	ConfirmPublish      bool                   `json:"confirmPublish,omitempty"`
}

type PublicationCandidate struct {
	ExternalItemID string `json:"externalItemId,omitempty"`
	Title          string `json:"title"`
	Verification   string `json:"verification"`
	EstimatedBytes int64  `json:"estimatedBytes"`
}

type PublicationPlan struct {
	ID                        string                `json:"id,omitempty"`
	CampaignID                string                `json:"campaignId"`
	ServerID                  string                `json:"serverId"`
	CollectionTitle           string                `json:"collectionTitle"`
	DryRun                    bool                  `json:"dryRun"`
	Status                    string                `json:"status"`
	PublishableItems          int                   `json:"publishableItems"`
	BlockedItems              int                   `json:"blockedItems"`
	PublishableEstimatedBytes int64                 `json:"publishableEstimatedBytes"`
	BlockedEstimatedBytes     int64                 `json:"blockedEstimatedBytes"`
	Items                     []PublicationPlanItem `json:"items"`
	CreatedAt                 time.Time             `json:"createdAt,omitempty"`
	PublishedAt               time.Time             `json:"publishedAt,omitempty"`
	Error                     string                `json:"error,omitempty"`
}

type PublicationPlanItem struct {
	ExternalItemID string `json:"externalItemId,omitempty"`
	Title          string `json:"title"`
	Verification   string `json:"verification"`
	EstimatedBytes int64  `json:"estimatedBytes"`
	Publishable    bool   `json:"publishable"`
	BlockedReason  string `json:"blockedReason,omitempty"`
}

type Notification struct {
	ID        string            `json:"id"`
	Level     string            `json:"level"`
	Title     string            `json:"title"`
	Body      string            `json:"body,omitempty"`
	EventType string            `json:"eventType,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
	Read      bool              `json:"read"`
	CreatedAt time.Time         `json:"createdAt"`
	ReadAt    time.Time         `json:"readAt,omitempty"`
}

func (notification Notification) WithDefaults() Notification {
	notification.ID = strings.TrimSpace(notification.ID)
	if notification.ID == "" {
		notification.ID = randomID("ntf")
	}
	notification.Level = strings.ToLower(strings.TrimSpace(notification.Level))
	if notification.Level == "" {
		notification.Level = "info"
	}
	if notification.Fields == nil {
		notification.Fields = map[string]string{}
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now().UTC()
	}
	return notification
}

type WebhookSetting struct {
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

type ProtectionRequest struct {
	ID               string    `json:"id"`
	RecommendationID string    `json:"recommendationId,omitempty"`
	ServerID         string    `json:"serverId,omitempty"`
	ExternalItemID   string    `json:"externalItemId,omitempty"`
	Title            string    `json:"title"`
	Path             string    `json:"path,omitempty"`
	Reason           string    `json:"reason,omitempty"`
	RequestedBy      string    `json:"requestedBy"`
	Status           string    `json:"status"`
	DecisionBy       string    `json:"decisionBy,omitempty"`
	DecisionNote     string    `json:"decisionNote,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	DecidedAt        time.Time `json:"decidedAt,omitempty"`
}

func (request ProtectionRequest) WithDefaults() ProtectionRequest {
	request.ID = strings.TrimSpace(request.ID)
	if request.ID == "" {
		request.ID = randomID("prt")
	}
	request.Status = strings.ToLower(strings.TrimSpace(request.Status))
	if request.Status == "" {
		request.Status = "pending"
	}
	if request.CreatedAt.IsZero() {
		request.CreatedAt = time.Now().UTC()
	}
	return request
}

func (request ProtectionRequest) Approve(decisionBy string, note string) (ProtectionRequest, error) {
	return request.decide("approved", decisionBy, note)
}

func (request ProtectionRequest) Decline(decisionBy string, note string) (ProtectionRequest, error) {
	return request.decide("declined", decisionBy, note)
}

func (request ProtectionRequest) decide(status string, decisionBy string, note string) (ProtectionRequest, error) {
	request = request.WithDefaults()
	if request.Status != "pending" {
		return ProtectionRequest{}, errors.New("protection request is already decided")
	}
	decisionBy = strings.TrimSpace(decisionBy)
	if decisionBy == "" {
		return ProtectionRequest{}, errors.New("decision actor is required")
	}
	request.Status = status
	request.DecisionBy = decisionBy
	request.DecisionNote = strings.TrimSpace(note)
	request.DecidedAt = time.Now().UTC()
	return request, nil
}

type CampaignTemplate struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Campaign    TemplateCampaignRef `json:"campaign"`
}

type TemplateCampaignRef struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	Enabled             bool           `json:"enabled"`
	TargetKinds         []string       `json:"targetKinds"`
	TargetLibraryNames  []string       `json:"targetLibraryNames,omitempty"`
	Rules               []TemplateRule `json:"rules"`
	RequireAllRules     bool           `json:"requireAllRules"`
	MinimumConfidence   float64        `json:"minimumConfidence"`
	MinimumStorageBytes int64          `json:"minimumStorageBytes"`
}

type TemplateRule struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Value    string   `json:"value,omitempty"`
	Values   []string `json:"values,omitempty"`
}

func randomID(prefix string) string {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}
