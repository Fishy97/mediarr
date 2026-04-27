package stewardship

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type tautulliHistoryEnvelope struct {
	Response struct {
		Result string          `json:"result"`
		Data   json.RawMessage `json:"data"`
	} `json:"response"`
}

type tautulliHistoryData struct {
	Data []tautulliHistoryRow `json:"data"`
}

type tautulliHistoryRow struct {
	RatingKey       any    `json:"rating_key"`
	User            string `json:"user"`
	Date            any    `json:"date"`
	PercentComplete int    `json:"percent_complete"`
}

func NormalizeTautulliHistory(payload []byte) ([]TautulliPlay, error) {
	var envelope tautulliHistoryEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(envelope.Response.Data)) == "" || string(envelope.Response.Data) == "null" {
		return []TautulliPlay{}, nil
	}
	var table tautulliHistoryData
	if err := json.Unmarshal(envelope.Response.Data, &table); err != nil {
		var rows []tautulliHistoryRow
		if err := json.Unmarshal(envelope.Response.Data, &rows); err != nil {
			return nil, err
		}
		table.Data = rows
	}
	plays := make([]TautulliPlay, 0, len(table.Data))
	for _, row := range table.Data {
		ratingKey := anyString(row.RatingKey)
		if ratingKey == "" {
			continue
		}
		plays = append(plays, TautulliPlay{
			RatingKey:       ratingKey,
			User:            strings.TrimSpace(row.User),
			PlayedAt:        unixAnyTime(row.Date),
			PercentComplete: row.PercentComplete,
		})
	}
	return plays, nil
}

func ApplyTautulliHistory(existing []ActivityRollup, plays []TautulliPlay, watchedThreshold int) []ActivityRollup {
	if watchedThreshold <= 0 {
		watchedThreshold = 80
	}
	byKey := map[string]*ActivityRollup{}
	userSeen := map[string]map[string]bool{}
	for index := range existing {
		rollup := existing[index]
		rollup.ServerID = firstNonEmpty(rollup.ServerID, "plex")
		byKey[rollup.ItemExternalID] = &rollup
	}
	for _, play := range plays {
		if play.PercentComplete > 0 && play.PercentComplete < watchedThreshold {
			continue
		}
		rollup := byKey[play.RatingKey]
		if rollup == nil {
			continue
		}
		rollup.PlayCount++
		if !play.PlayedAt.IsZero() && (rollup.LastPlayedAt.IsZero() || play.PlayedAt.After(rollup.LastPlayedAt)) {
			rollup.LastPlayedAt = play.PlayedAt
		}
		user := strings.TrimSpace(play.User)
		if user != "" {
			if userSeen[play.RatingKey] == nil {
				userSeen[play.RatingKey] = map[string]bool{}
			}
			if !userSeen[play.RatingKey][user] {
				userSeen[play.RatingKey][user] = true
				rollup.UniqueUsers++
				rollup.WatchedUsers++
			}
		}
		rollup.EvidenceSource = "tautulli"
		rollup.UpdatedAt = time.Now().UTC()
		byKey[play.RatingKey] = rollup
	}
	enriched := make([]ActivityRollup, 0, len(existing))
	for _, original := range existing {
		if rollup := byKey[original.ItemExternalID]; rollup != nil {
			enriched = append(enriched, *rollup)
		}
	}
	return enriched
}

func anyString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	default:
		return ""
	}
}

func unixAnyTime(value any) time.Time {
	raw := anyString(value)
	if raw == "" {
		return time.Time{}
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seconds <= 0 {
		return parseTime(raw)
	}
	return time.Unix(seconds, 0).UTC()
}
