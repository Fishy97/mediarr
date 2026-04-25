package recommendations

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
)

type Action string

const (
	ActionReviewDuplicate Action = "review_duplicate"
	ActionReviewOversized Action = "review_oversized"
)

type MediaFile struct {
	ID           string `json:"id"`
	CanonicalKey string `json:"canonicalKey"`
	Path         string `json:"path"`
	SizeBytes    int64  `json:"sizeBytes"`
	Quality      string `json:"quality,omitempty"`
}

type Recommendation struct {
	ID              string   `json:"id"`
	Action          Action   `json:"action"`
	Title           string   `json:"title"`
	Explanation     string   `json:"explanation"`
	SpaceSavedBytes int64    `json:"spaceSavedBytes"`
	Confidence      float64  `json:"confidence"`
	Source          string   `json:"source"`
	AffectedPaths   []string `json:"affectedPaths"`
	Destructive     bool     `json:"destructive"`
}

type Engine struct {
	OversizedThresholdBytes int64
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

func stableID(value string) string {
	sum := sha1.Sum([]byte(value))
	return "rec_" + hex.EncodeToString(sum[:8])
}
