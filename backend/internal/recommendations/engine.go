package recommendations

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strconv"
	"time"
)

type Action string

const (
	ActionReviewDuplicate          Action = "review_duplicate"
	ActionReviewOversized          Action = "review_oversized"
	ActionReviewMissingSubtitles   Action = "review_missing_subtitles"
	ActionReviewInactiveMovie      Action = "review_inactive_movie"
	ActionReviewNeverWatchedMovie  Action = "review_never_watched_movie"
	ActionReviewInactiveSeries     Action = "review_inactive_series"
	ActionReviewAbandonedSeries    Action = "review_abandoned_series"
	ActionReviewUnwatchedDuplicate Action = "review_unwatched_duplicate"
)

type MediaFile struct {
	ID             string `json:"id"`
	CanonicalKey   string `json:"canonicalKey"`
	Path           string `json:"path"`
	SizeBytes      int64  `json:"sizeBytes"`
	Quality        string `json:"quality,omitempty"`
	HasSubtitles   bool   `json:"hasSubtitles"`
	WantsSubtitles bool   `json:"wantsSubtitles"`
}

type ActivityMedia struct {
	ServerID        string
	ExternalItemID  string
	Kind            string
	Title           string
	Path            string
	SizeBytes       int64
	AddedAt         time.Time
	LastPlayedAt    time.Time
	PlayCount       int
	UniqueUsers     int
	FavoriteCount   int
	Verification    string
	MatchConfidence float64
}

type Recommendation struct {
	ID              string            `json:"id"`
	Action          Action            `json:"action"`
	Title           string            `json:"title"`
	Explanation     string            `json:"explanation"`
	SpaceSavedBytes int64             `json:"spaceSavedBytes"`
	Confidence      float64           `json:"confidence"`
	Source          string            `json:"source"`
	AffectedPaths   []string          `json:"affectedPaths"`
	Destructive     bool              `json:"destructive"`
	AIRationale     string            `json:"aiRationale,omitempty"`
	AITags          []string          `json:"aiTags,omitempty"`
	AIConfidence    float64           `json:"aiConfidence,omitempty"`
	AISource        string            `json:"aiSource,omitempty"`
	ServerID        string            `json:"serverId,omitempty"`
	ExternalItemID  string            `json:"externalItemId,omitempty"`
	LastPlayedAt    time.Time         `json:"lastPlayedAt,omitempty"`
	PlayCount       int               `json:"playCount,omitempty"`
	UniqueUsers     int               `json:"uniqueUsers,omitempty"`
	FavoriteCount   int               `json:"favoriteCount,omitempty"`
	Verification    string            `json:"verification,omitempty"`
	Evidence        map[string]string `json:"evidence,omitempty"`
}

type Engine struct {
	OversizedThresholdBytes int64
	NeverWatchedDays        int
	InactiveDays            int
}

func (engine Engine) Generate(files []MediaFile) []Recommendation {
	threshold := engine.OversizedThresholdBytes
	if threshold == 0 {
		threshold = 60_000_000_000
	}

	byKey := map[string][]MediaFile{}
	for _, file := range files {
		if file.CanonicalKey == "" {
			continue
		}
		byKey[file.CanonicalKey] = append(byKey[file.CanonicalKey], file)
	}

	var recs []Recommendation
	for key, group := range byKey {
		sort.Slice(group, func(i, j int) bool {
			if group[i].SizeBytes == group[j].SizeBytes {
				return group[i].Path < group[j].Path
			}
			return group[i].SizeBytes > group[j].SizeBytes
		})
		if len(group) > 1 {
			var total int64
			var paths []string
			for _, file := range group {
				total += file.SizeBytes
				paths = append(paths, file.Path)
			}
			recs = append(recs, Recommendation{
				ID:              stableID("duplicate:" + key),
				Action:          ActionReviewDuplicate,
				Title:           "Review duplicate media",
				Explanation:     "Multiple files resolve to the same catalog item. Keep the intended edition or highest-value copy and remove only after manual review.",
				SpaceSavedBytes: total - group[0].SizeBytes,
				Confidence:      0.92,
				Source:          "rule:duplicate-canonical-key",
				AffectedPaths:   paths,
				Destructive:     false,
			})
			continue
		}
		file := group[0]
		if file.WantsSubtitles && !file.HasSubtitles {
			recs = append(recs, Recommendation{
				ID:              stableID("missing-subtitles:" + file.Path),
				Action:          ActionReviewMissingSubtitles,
				Title:           "Review missing subtitles",
				Explanation:     "No sidecar subtitles were detected for this file. Confirm whether embedded subtitles exist or add a subtitle file if needed.",
				SpaceSavedBytes: 0,
				Confidence:      0.74,
				Source:          "rule:missing-sidecar-subtitles",
				AffectedPaths:   []string{file.Path},
				Destructive:     false,
			})
		}
		if file.SizeBytes >= threshold {
			recs = append(recs, Recommendation{
				ID:              stableID("oversized:" + file.Path),
				Action:          ActionReviewOversized,
				Title:           "Review unusually large file",
				Explanation:     "This file is larger than the configured review threshold. Consider whether the edition, quality, or encode is worth the storage cost.",
				SpaceSavedBytes: file.SizeBytes,
				Confidence:      0.67,
				Source:          "rule:oversized-file",
				AffectedPaths:   []string{file.Path},
				Destructive:     false,
			})
		}
	}

	sort.Slice(recs, func(i, j int) bool {
		if recs[i].SpaceSavedBytes == recs[j].SpaceSavedBytes {
			return recs[i].ID < recs[j].ID
		}
		return recs[i].SpaceSavedBytes > recs[j].SpaceSavedBytes
	})
	return recs
}

func (engine Engine) GenerateActivity(items []ActivityMedia, now time.Time) []Recommendation {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	neverWatchedDays := engine.NeverWatchedDays
	if neverWatchedDays == 0 {
		neverWatchedDays = 180
	}
	inactiveDays := engine.InactiveDays
	if inactiveDays == 0 {
		inactiveDays = 540
	}

	var recs []Recommendation
	for _, item := range items {
		if item.Kind != "movie" || item.SizeBytes <= 0 || item.Path == "" || item.Title == "" {
			continue
		}
		if item.FavoriteCount > 0 {
			continue
		}
		confidence := activityConfidence(item.MatchConfidence, item.Verification)
		if confidence < 0.65 {
			continue
		}
		if item.PlayCount == 0 && !item.AddedAt.IsZero() {
			ageDays := int(now.Sub(item.AddedAt).Hours() / 24)
			if ageDays >= neverWatchedDays {
				recs = append(recs, Recommendation{
					ID:              stableID("activity-never-watched:" + item.ServerID + ":" + item.ExternalItemID + ":" + item.Path),
					Action:          ActionReviewNeverWatchedMovie,
					Title:           "Review never-watched movie",
					Explanation:     item.Title + " has not been watched by any imported media-server user since it was added. Review whether it is still worth keeping before reclaiming storage.",
					SpaceSavedBytes: item.SizeBytes,
					Confidence:      confidence,
					Source:          "rule:activity-never-watched",
					AffectedPaths:   []string{item.Path},
					Destructive:     false,
					ServerID:        item.ServerID,
					ExternalItemID:  item.ExternalItemID,
					PlayCount:       item.PlayCount,
					UniqueUsers:     item.UniqueUsers,
					FavoriteCount:   item.FavoriteCount,
					Verification:    item.Verification,
					Evidence: map[string]string{
						"ageDays":       strconv.Itoa(ageDays),
						"thresholdDays": strconv.Itoa(neverWatchedDays),
					},
				})
			}
			continue
		}
		if item.PlayCount > 0 && !item.LastPlayedAt.IsZero() {
			inactiveForDays := int(now.Sub(item.LastPlayedAt).Hours() / 24)
			if inactiveForDays >= inactiveDays {
				recs = append(recs, Recommendation{
					ID:              stableID("activity-inactive:" + item.ServerID + ":" + item.ExternalItemID + ":" + item.Path),
					Action:          ActionReviewInactiveMovie,
					Title:           "Review inactive movie",
					Explanation:     item.Title + " has not been watched recently by any imported media-server user. Review activity and quality before reclaiming storage.",
					SpaceSavedBytes: item.SizeBytes,
					Confidence:      confidence,
					Source:          "rule:activity-inactive-movie",
					AffectedPaths:   []string{item.Path},
					Destructive:     false,
					ServerID:        item.ServerID,
					ExternalItemID:  item.ExternalItemID,
					LastPlayedAt:    item.LastPlayedAt,
					PlayCount:       item.PlayCount,
					UniqueUsers:     item.UniqueUsers,
					FavoriteCount:   item.FavoriteCount,
					Verification:    item.Verification,
					Evidence: map[string]string{
						"inactiveDays":  strconv.Itoa(inactiveForDays),
						"thresholdDays": strconv.Itoa(inactiveDays),
					},
				})
			}
		}
	}
	sort.Slice(recs, func(i, j int) bool {
		if recs[i].SpaceSavedBytes == recs[j].SpaceSavedBytes {
			return recs[i].ID < recs[j].ID
		}
		return recs[i].SpaceSavedBytes > recs[j].SpaceSavedBytes
	})
	return recs
}

func activityConfidence(matchConfidence float64, verification string) float64 {
	if matchConfidence <= 0 {
		matchConfidence = 0.7
	}
	verificationConfidence := 0.72
	switch verification {
	case "local_verified":
		verificationConfidence = 0.92
	case "path_mapped":
		verificationConfidence = 0.86
	case "server_reported":
		verificationConfidence = 0.72
	}
	if matchConfidence < verificationConfidence {
		return matchConfidence
	}
	return verificationConfidence
}

func stableID(value string) string {
	sum := sha1.Sum([]byte(value))
	return "rec_" + hex.EncodeToString(sum[:8])
}
