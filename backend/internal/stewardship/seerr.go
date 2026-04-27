package stewardship

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type seerrRequestsEnvelope struct {
	Results []seerrRequest `json:"results"`
}

type seerrRequest struct {
	ID          int        `json:"id"`
	Status      int        `json:"status"`
	CreatedAt   string     `json:"createdAt"`
	UpdatedAt   string     `json:"updatedAt"`
	RequestedBy seerrUser  `json:"requestedBy"`
	Media       seerrMedia `json:"media"`
}

type seerrUser struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PlexUsername string `json:"plexUsername"`
	DisplayName  string `json:"displayName"`
}

type seerrMedia struct {
	ID        int    `json:"id"`
	TMDBID    int    `json:"tmdbId"`
	TVDBID    int    `json:"tvdbId"`
	IMDBID    string `json:"imdbId"`
	Status    int    `json:"status"`
	MediaType string `json:"mediaType"`
	Title     string `json:"title"`
	Name      string `json:"name"`
}

func NormalizeSeerrRequests(sourceID string, payload []byte) ([]RequestSignal, error) {
	var envelope seerrRequestsEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}
	sourceID = strings.TrimSpace(sourceID)
	signals := make([]RequestSignal, 0, len(envelope.Results))
	for _, request := range envelope.Results {
		providers := map[string]string{}
		if request.Media.TMDBID > 0 {
			providers["tmdb"] = strconv.Itoa(request.Media.TMDBID)
		}
		if request.Media.TVDBID > 0 {
			providers["tvdb"] = strconv.Itoa(request.Media.TVDBID)
		}
		if strings.TrimSpace(request.Media.IMDBID) != "" {
			providers["imdb"] = strings.TrimSpace(request.Media.IMDBID)
		}
		mediaType := strings.ToLower(strings.TrimSpace(request.Media.MediaType))
		if mediaType == "" {
			mediaType = "unknown"
		}
		updatedAt := parseTime(request.UpdatedAt)
		signal := RequestSignal{
			SourceID:          sourceID,
			ExternalRequestID: strconv.Itoa(request.ID),
			MediaType:         mediaType,
			ExternalMediaID:   strconv.Itoa(request.Media.ID),
			Title:             firstNonEmpty(request.Media.Title, request.Media.Name),
			Status:            normalizeRequestStatus(request.Status),
			Availability:      normalizeAvailabilityStatus(request.Media.Status),
			RequestedBy:       firstNonEmpty(request.RequestedBy.Username, request.RequestedBy.PlexUsername, request.RequestedBy.DisplayName),
			ProviderIDs:       providers,
			RequestedAt:       parseTime(request.CreatedAt),
			UpdatedAt:         updatedAt,
		}
		if signal.Status == "approved" {
			signal.ApprovedAt = updatedAt
		}
		if signal.Availability == "available" {
			signal.AvailableAt = updatedAt
		}
		signals = append(signals, signal)
	}
	return signals, nil
}

func normalizeRequestStatus(status int) string {
	switch status {
	case 1:
		return "pending"
	case 2:
		return "approved"
	case 3:
		return "declined"
	default:
		return "unknown"
	}
}

func normalizeAvailabilityStatus(status int) string {
	switch status {
	case 2:
		return "pending"
	case 3:
		return "processing"
	case 4:
		return "partially_available"
	case 5:
		return "available"
	case 6:
		return "deleted"
	default:
		return "unknown"
	}
}

func parseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
