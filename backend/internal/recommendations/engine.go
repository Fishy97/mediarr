package recommendations

import (
	"crypto/sha1"
	"encoding/hex"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Action string
type State string

const (
	ActionReviewDuplicate          Action = "review_duplicate"
	ActionReviewOversized          Action = "review_oversized"
	ActionReviewMissingSubtitles   Action = "review_missing_subtitles"
	ActionReviewInactiveMovie      Action = "review_inactive_movie"
	ActionReviewNeverWatchedMovie  Action = "review_never_watched_movie"
	ActionReviewInactiveSeries     Action = "review_inactive_series"
	ActionReviewAbandonedSeries    Action = "review_abandoned_series"
	ActionReviewUnwatchedDuplicate Action = "review_unwatched_duplicate"
	ActionReviewCampaignMatch      Action = "review_campaign_match"
)

const (
	StateNew                     State = "new"
	StateReviewing               State = "reviewing"
	StateIgnored                 State = "ignored"
	StateProtected               State = "protected"
	StateAcceptedForManualAction State = "accepted_for_manual_action"
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
	ServerID             string
	ExternalItemID       string
	ParentExternalItemID string
	ParentTitle          string
	LibraryName          string
	Kind                 string
	Title                string
	Path                 string
	SizeBytes            int64
	AddedAt              time.Time
	LastPlayedAt         time.Time
	PlayCount            int
	UniqueUsers          int
	FavoriteCount        int
	Verification         string
	MatchConfidence      float64
}

type Recommendation struct {
	ID              string            `json:"id"`
	Action          Action            `json:"action"`
	State           State             `json:"state"`
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

type Evidence struct {
	RecommendationID   string            `json:"recommendationId"`
	State              State             `json:"state"`
	Title              string            `json:"title"`
	Explanation        string            `json:"explanation"`
	Confidence         float64           `json:"confidence"`
	Destructive        bool              `json:"destructive"`
	AffectedPaths      []string          `json:"affectedPaths"`
	SuppressionReasons []string          `json:"suppressionReasons"`
	Raw                map[string]string `json:"raw"`
	Storage            StorageEvidence   `json:"storage"`
	Activity           ActivityEvidence  `json:"activity"`
	Source             SourceEvidence    `json:"source"`
	Proof              []ProofPoint      `json:"proof"`
}

type StorageEvidence struct {
	SpaceSavedBytes       int64  `json:"spaceSavedBytes"`
	EstimatedSavingsBytes int64  `json:"estimatedSavingsBytes"`
	VerifiedSavingsBytes  int64  `json:"verifiedSavingsBytes"`
	Verification          string `json:"verification"`
	Certainty             string `json:"certainty"`
	Basis                 string `json:"basis"`
	Risk                  string `json:"risk"`
}

type ActivityEvidence struct {
	ServerID       string    `json:"serverId,omitempty"`
	ExternalItemID string    `json:"externalItemId,omitempty"`
	LastPlayedAt   time.Time `json:"lastPlayedAt,omitempty"`
	PlayCount      int       `json:"playCount"`
	UniqueUsers    int       `json:"uniqueUsers"`
	FavoriteCount  int       `json:"favoriteCount"`
}

type SourceEvidence struct {
	Rule         string  `json:"rule"`
	AI           string  `json:"ai,omitempty"`
	AIConfidence float64 `json:"aiConfidence,omitempty"`
}

type ProofPoint struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Status string `json:"status"`
}

func (rec Recommendation) WithDefaults() Recommendation {
	if rec.State == "" {
		rec.State = StateNew
	}
	if rec.Evidence == nil {
		rec.Evidence = map[string]string{}
	}
	if rec.AffectedPaths == nil {
		rec.AffectedPaths = []string{}
	}
	if rec.AITags == nil {
		rec.AITags = []string{}
	}
	return rec
}

func BuildEvidence(rec Recommendation) Evidence {
	rec = rec.WithDefaults()
	raw := map[string]string{}
	for key, value := range rec.Evidence {
		raw[key] = value
	}
	storageRisk := storageRisk(rec.Verification, rec.Confidence, rec.Destructive)
	storageBasis := firstActivityValue(raw["storageBasis"], rec.Verification)
	storageCertainty := firstActivityValue(raw["storageCertainty"], storageCertaintyLabel(rec.Verification))
	estimatedSavingsBytes := parseEvidenceInt64(raw["estimatedSavingsBytes"], rec.SpaceSavedBytes)
	verifiedSavingsBytes := parseEvidenceInt64(raw["verifiedSavingsBytes"], defaultVerifiedSavingsBytes(rec.Verification, rec.SpaceSavedBytes))
	evidence := Evidence{
		RecommendationID:   rec.ID,
		State:              rec.State,
		Title:              rec.Title,
		Explanation:        rec.Explanation,
		Confidence:         rec.Confidence,
		Destructive:        rec.Destructive,
		AffectedPaths:      append([]string(nil), rec.AffectedPaths...),
		SuppressionReasons: suppressionReasons(rec),
		Raw:                raw,
		Storage: StorageEvidence{
			SpaceSavedBytes:       rec.SpaceSavedBytes,
			EstimatedSavingsBytes: estimatedSavingsBytes,
			VerifiedSavingsBytes:  verifiedSavingsBytes,
			Verification:          rec.Verification,
			Certainty:             storageCertainty,
			Basis:                 storageBasis,
			Risk:                  storageRisk,
		},
		Activity: ActivityEvidence{
			ServerID:       rec.ServerID,
			ExternalItemID: rec.ExternalItemID,
			LastPlayedAt:   rec.LastPlayedAt,
			PlayCount:      rec.PlayCount,
			UniqueUsers:    rec.UniqueUsers,
			FavoriteCount:  rec.FavoriteCount,
		},
		Source: SourceEvidence{
			Rule:         rec.Source,
			AI:           rec.AISource,
			AIConfidence: rec.AIConfidence,
		},
	}
	evidence.Proof = []ProofPoint{
		{Label: "Deletion mode", Value: "Suggest-only", Status: "safe"},
		{Label: "Storage proof", Value: verificationLabel(rec.Verification), Status: storageRisk},
		{Label: "Rule source", Value: rec.Source, Status: "info"},
	}
	if rec.ServerID != "" {
		evidence.Proof = append(evidence.Proof,
			ProofPoint{Label: "Media server", Value: rec.ServerID, Status: "info"},
			ProofPoint{Label: "Activity", Value: activityLabel(rec), Status: "info"},
		)
	}
	return evidence
}

type Engine struct {
	OversizedThresholdBytes int64
	NeverWatchedDays        int
	InactiveDays            int
	ActiveSeriesGraceDays   int
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
	activeSeriesGraceDays := engine.ActiveSeriesGraceDays
	if activeSeriesGraceDays == 0 {
		activeSeriesGraceDays = 90
	}

	var recs []Recommendation
	for _, item := range items {
		if item.Kind != "movie" || item.SizeBytes <= 0 || item.Path == "" || item.Title == "" {
			continue
		}
		if item.FavoriteCount > 0 {
			continue
		}
		if item.PlayCount == 0 && !item.AddedAt.IsZero() {
			ageDays := int(now.Sub(item.AddedAt).Hours() / 24)
			if ageDays >= neverWatchedDays {
				confidenceFactors := activityConfidenceFactors{
					MatchConfidence: item.MatchConfidence,
					Verification:    item.Verification,
					Action:          ActionReviewNeverWatchedMovie,
					EvidenceDays:    ageDays,
					ThresholdDays:   neverWatchedDays,
					ItemCount:       1,
					PlayCount:       item.PlayCount,
					UniqueUsers:     item.UniqueUsers,
				}
				confidence := activityRecommendationConfidence(confidenceFactors)
				if confidence < 0.65 {
					continue
				}
				recs = append(recs, Recommendation{
					ID:              stableID("activity-never-watched:" + item.ServerID + ":" + item.ExternalItemID + ":" + item.Path),
					Action:          ActionReviewNeverWatchedMovie,
					Title:           item.Title,
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
					Evidence: activityEvidenceMap(item.Verification, item.SizeBytes, confidenceFactors, map[string]string{
						"ageDays":       strconv.Itoa(ageDays),
						"thresholdDays": strconv.Itoa(neverWatchedDays),
						"itemCount":     "1",
						"subjectKind":   "movie",
						"subjectTitle":  item.Title,
					}),
				})
			}
			continue
		}
		if item.PlayCount > 0 && !item.LastPlayedAt.IsZero() {
			inactiveForDays := int(now.Sub(item.LastPlayedAt).Hours() / 24)
			if inactiveForDays >= inactiveDays {
				confidenceFactors := activityConfidenceFactors{
					MatchConfidence: item.MatchConfidence,
					Verification:    item.Verification,
					Action:          ActionReviewInactiveMovie,
					EvidenceDays:    inactiveForDays,
					ThresholdDays:   inactiveDays,
					ItemCount:       1,
					PlayCount:       item.PlayCount,
					UniqueUsers:     item.UniqueUsers,
				}
				confidence := activityRecommendationConfidence(confidenceFactors)
				if confidence < 0.65 {
					continue
				}
				recs = append(recs, Recommendation{
					ID:              stableID("activity-inactive:" + item.ServerID + ":" + item.ExternalItemID + ":" + item.Path),
					Action:          ActionReviewInactiveMovie,
					Title:           item.Title,
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
					Evidence: activityEvidenceMap(item.Verification, item.SizeBytes, confidenceFactors, map[string]string{
						"inactiveDays":  strconv.Itoa(inactiveForDays),
						"thresholdDays": strconv.Itoa(inactiveDays),
						"itemCount":     "1",
						"subjectKind":   "movie",
						"subjectTitle":  item.Title,
					}),
				})
			}
		}
	}
	recs = append(recs, engine.generateSeriesActivity(items, now, neverWatchedDays, inactiveDays, activeSeriesGraceDays)...)
	sort.Slice(recs, func(i, j int) bool {
		if recs[i].SpaceSavedBytes == recs[j].SpaceSavedBytes {
			return recs[i].ID < recs[j].ID
		}
		return recs[i].SpaceSavedBytes > recs[j].SpaceSavedBytes
	})
	return recs
}

type seriesActivityGroup struct {
	serverID        string
	externalID      string
	title           string
	libraryName     string
	sizeBytes       int64
	paths           []string
	latestAddedAt   time.Time
	lastPlayedAt    time.Time
	playCount       int
	uniqueUsers     int
	favoriteCount   int
	verification    string
	matchConfidence float64
	matchCount      int
	isAnime         bool
}

func (engine Engine) generateSeriesActivity(items []ActivityMedia, now time.Time, neverWatchedDays int, inactiveDays int, activeSeriesGraceDays int) []Recommendation {
	groups := map[string]*seriesActivityGroup{}
	for _, item := range items {
		if !isSeriesActivityItem(item) || item.SizeBytes <= 0 || strings.TrimSpace(item.Path) == "" {
			continue
		}
		externalID := firstActivityValue(item.ParentExternalItemID, item.ExternalItemID)
		if externalID == "" {
			continue
		}
		key := item.ServerID + ":" + externalID
		group := groups[key]
		if group == nil {
			group = &seriesActivityGroup{
				serverID:        item.ServerID,
				externalID:      externalID,
				title:           firstActivityValue(item.ParentTitle, item.Title),
				libraryName:     item.LibraryName,
				verification:    strings.TrimSpace(item.Verification),
				matchConfidence: 1,
				isAnime:         activityLooksAnime(item),
			}
			groups[key] = group
		}
		if group.title == "" {
			group.title = firstActivityValue(item.ParentTitle, item.Title)
		}
		if group.libraryName == "" {
			group.libraryName = item.LibraryName
		}
		group.isAnime = group.isAnime || activityLooksAnime(item)
		group.sizeBytes += item.SizeBytes
		group.paths = append(group.paths, item.Path)
		if !item.AddedAt.IsZero() && (group.latestAddedAt.IsZero() || item.AddedAt.After(group.latestAddedAt)) {
			group.latestAddedAt = item.AddedAt
		}
		if !item.LastPlayedAt.IsZero() && (group.lastPlayedAt.IsZero() || item.LastPlayedAt.After(group.lastPlayedAt)) {
			group.lastPlayedAt = item.LastPlayedAt
		}
		group.playCount += item.PlayCount
		if item.UniqueUsers > group.uniqueUsers {
			group.uniqueUsers = item.UniqueUsers
		}
		group.favoriteCount += item.FavoriteCount
		group.verification = weakerVerification(group.verification, item.Verification)
		if item.MatchConfidence > 0 {
			if group.matchCount == 0 || item.MatchConfidence < group.matchConfidence {
				group.matchConfidence = item.MatchConfidence
			}
			group.matchCount++
		}
	}

	var recs []Recommendation
	for _, group := range groups {
		if group.title == "" || group.sizeBytes <= 0 || len(group.paths) == 0 {
			continue
		}
		if group.favoriteCount > 0 {
			continue
		}
		if group.matchCount == 0 {
			group.matchConfidence = 0
		}
		sort.Strings(group.paths)
		if !group.latestAddedAt.IsZero() {
			activeAgeDays := int(now.Sub(group.latestAddedAt).Hours() / 24)
			if activeAgeDays >= 0 && activeAgeDays < activeSeriesGraceDays {
				continue
			}
		}
		category := "series"
		subjectKind := "series"
		if group.isAnime {
			category = "anime"
			subjectKind = "anime"
		}
		if group.playCount == 0 && !group.latestAddedAt.IsZero() {
			ageDays := int(now.Sub(group.latestAddedAt).Hours() / 24)
			if ageDays >= neverWatchedDays {
				confidenceFactors := activityConfidenceFactors{
					MatchConfidence: group.matchConfidence,
					Verification:    group.verification,
					Action:          ActionReviewAbandonedSeries,
					EvidenceDays:    ageDays,
					ThresholdDays:   neverWatchedDays,
					ItemCount:       len(group.paths),
					PlayCount:       group.playCount,
					UniqueUsers:     group.uniqueUsers,
				}
				confidence := activityRecommendationConfidence(confidenceFactors)
				if confidence < 0.65 {
					continue
				}
				recs = append(recs, Recommendation{
					ID:              stableID("activity-abandoned-series:" + group.serverID + ":" + group.externalID + ":" + strings.Join(group.paths, "|")),
					Action:          ActionReviewAbandonedSeries,
					Title:           group.title,
					Explanation:     group.title + " has no imported play activity and all known files are older than the review threshold. Review the series before reclaiming storage.",
					SpaceSavedBytes: group.sizeBytes,
					Confidence:      confidence,
					Source:          "rule:activity-abandoned-series",
					AffectedPaths:   group.paths,
					Destructive:     false,
					ServerID:        group.serverID,
					ExternalItemID:  group.externalID,
					PlayCount:       group.playCount,
					UniqueUsers:     group.uniqueUsers,
					FavoriteCount:   group.favoriteCount,
					Verification:    group.verification,
					Evidence: activityEvidenceMap(group.verification, group.sizeBytes, confidenceFactors, map[string]string{
						"ageDays":       strconv.Itoa(ageDays),
						"thresholdDays": strconv.Itoa(neverWatchedDays),
						"itemCount":     strconv.Itoa(len(group.paths)),
						"category":      category,
						"seriesTitle":   group.title,
						"subjectKind":   subjectKind,
						"subjectTitle":  group.title,
					}),
				})
			}
			continue
		}
		if group.playCount > 0 && !group.lastPlayedAt.IsZero() {
			inactiveForDays := int(now.Sub(group.lastPlayedAt).Hours() / 24)
			if inactiveForDays >= inactiveDays {
				confidenceFactors := activityConfidenceFactors{
					MatchConfidence: group.matchConfidence,
					Verification:    group.verification,
					Action:          ActionReviewInactiveSeries,
					EvidenceDays:    inactiveForDays,
					ThresholdDays:   inactiveDays,
					ItemCount:       len(group.paths),
					PlayCount:       group.playCount,
					UniqueUsers:     group.uniqueUsers,
				}
				confidence := activityRecommendationConfidence(confidenceFactors)
				if confidence < 0.65 {
					continue
				}
				recs = append(recs, Recommendation{
					ID:              stableID("activity-inactive-series:" + group.serverID + ":" + group.externalID + ":" + strings.Join(group.paths, "|")),
					Action:          ActionReviewInactiveSeries,
					Title:           group.title,
					Explanation:     group.title + " has not been watched recently by any imported media-server user. Review the full series before reclaiming storage.",
					SpaceSavedBytes: group.sizeBytes,
					Confidence:      confidence,
					Source:          "rule:activity-inactive-series",
					AffectedPaths:   group.paths,
					Destructive:     false,
					ServerID:        group.serverID,
					ExternalItemID:  group.externalID,
					LastPlayedAt:    group.lastPlayedAt,
					PlayCount:       group.playCount,
					UniqueUsers:     group.uniqueUsers,
					FavoriteCount:   group.favoriteCount,
					Verification:    group.verification,
					Evidence: activityEvidenceMap(group.verification, group.sizeBytes, confidenceFactors, map[string]string{
						"inactiveDays":  strconv.Itoa(inactiveForDays),
						"thresholdDays": strconv.Itoa(inactiveDays),
						"itemCount":     strconv.Itoa(len(group.paths)),
						"category":      category,
						"seriesTitle":   group.title,
						"subjectKind":   subjectKind,
						"subjectTitle":  group.title,
					}),
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

func firstActivityValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func isSeriesActivityItem(item ActivityMedia) bool {
	kind := strings.ToLower(strings.TrimSpace(item.Kind))
	switch kind {
	case "episode", "season":
		return true
	case "series":
		return strings.TrimSpace(item.Path) != ""
	case "video":
		return strings.TrimSpace(item.ParentExternalItemID) != "" || activityLooksAnime(item)
	default:
		return strings.TrimSpace(item.ParentExternalItemID) != "" && kind != "movie"
	}
}

func activityLooksAnime(item ActivityMedia) bool {
	haystack := strings.ToLower(strings.Join([]string{item.LibraryName, item.Path, item.ParentTitle}, " "))
	return strings.Contains(haystack, "anime")
}

func weakerVerification(current string, next string) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if current == "" {
		return next
	}
	if next == "" {
		return current
	}
	if verificationRank(next) < verificationRank(current) {
		return next
	}
	return current
}

func verificationRank(verification string) int {
	switch strings.TrimSpace(verification) {
	case "local_verified":
		return 4
	case "path_mapped":
		return 3
	case "server_reported":
		return 2
	case "unmapped":
		return 1
	default:
		return 0
	}
}

func suppressionReasons(rec Recommendation) []string {
	reasons := []string{}
	if rec.FavoriteCount > 0 {
		reasons = append(reasons, "favorite")
	}
	if rec.Confidence < 0.65 {
		reasons = append(reasons, "low_confidence")
	}
	if rec.Verification == "" || rec.Verification == "unmapped" {
		reasons = append(reasons, "missing_verified_path")
	}
	if rec.State == StateProtected {
		reasons = append(reasons, "user_protected")
	}
	return reasons
}

type activityConfidenceFactors struct {
	MatchConfidence float64
	Verification    string
	Action          Action
	EvidenceDays    int
	ThresholdDays   int
	ItemCount       int
	PlayCount       int
	UniqueUsers     int
}

func activityEvidenceMap(verification string, estimatedSavingsBytes int64, factors activityConfidenceFactors, values map[string]string) map[string]string {
	values = storageEvidenceMap(verification, estimatedSavingsBytes, values)
	values["confidenceBasis"] = confidenceBasisLabel(factors)
	return values
}

func storageEvidenceMap(verification string, estimatedSavingsBytes int64, values map[string]string) map[string]string {
	if values == nil {
		values = map[string]string{}
	}
	values["storageBasis"] = strings.TrimSpace(verification)
	values["estimatedSavingsBytes"] = strconv.FormatInt(estimatedSavingsBytes, 10)
	values["verifiedSavingsBytes"] = strconv.FormatInt(defaultVerifiedSavingsBytes(verification, estimatedSavingsBytes), 10)
	values["storageCertainty"] = storageCertaintyLabel(verification)
	return values
}

func defaultVerifiedSavingsBytes(verification string, estimatedSavingsBytes int64) int64 {
	if strings.TrimSpace(verification) == "local_verified" {
		return estimatedSavingsBytes
	}
	return 0
}

func storageCertaintyLabel(verification string) string {
	switch strings.TrimSpace(verification) {
	case "local_verified":
		return "verified"
	case "path_mapped":
		return "mapped_estimate"
	case "server_reported":
		return "estimate"
	case "unmapped":
		return "unmapped"
	default:
		return "estimate"
	}
}

func parseEvidenceInt64(value string, fallback int64) int64 {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func storageRisk(verification string, confidence float64, destructive bool) string {
	if destructive {
		return "blocked"
	}
	switch verification {
	case "local_verified":
		if confidence >= 0.85 {
			return "strong"
		}
		return "moderate"
	case "path_mapped":
		return "moderate"
	case "server_reported":
		return "caution"
	default:
		return "unknown"
	}
}

func verificationLabel(verification string) string {
	switch verification {
	case "local_verified":
		return "Local file verified"
	case "path_mapped":
		return "Path mapped"
	case "server_reported":
		return "Server reported"
	case "unmapped":
		return "Unmapped"
	default:
		return "Unknown"
	}
}

func activityLabel(rec Recommendation) string {
	if rec.PlayCount == 0 {
		return "Never watched by imported users"
	}
	return strconv.Itoa(rec.PlayCount) + " plays across " + strconv.Itoa(rec.UniqueUsers) + " users"
}

func activityConfidence(matchConfidence float64, verification string) float64 {
	return activityRecommendationConfidence(activityConfidenceFactors{
		MatchConfidence: matchConfidence,
		Verification:    verification,
		ItemCount:       1,
	})
}

func activityRecommendationConfidence(factors activityConfidenceFactors) float64 {
	matchConfidence := clampFloat(factors.MatchConfidence, 0, 1)
	if matchConfidence <= 0 {
		matchConfidence = 0.7
	}

	score := 0.50 + matchConfidence*0.18
	score += verificationConfidenceAdjustment(factors.Verification)
	score += actionConfidenceAdjustment(factors.Action)

	if factors.ThresholdDays > 0 && factors.EvidenceDays > factors.ThresholdDays {
		ratio := float64(factors.EvidenceDays) / float64(factors.ThresholdDays)
		score += clampFloat((ratio-1)*0.035, 0, 0.12)
	}
	if factors.ItemCount > 1 {
		score += clampFloat(math.Log1p(float64(factors.ItemCount))*0.012, 0, 0.05)
	}
	if factors.PlayCount > 0 {
		score -= clampFloat(math.Log1p(float64(factors.PlayCount))*0.018, 0, 0.09)
	}
	if factors.UniqueUsers > 1 {
		score -= clampFloat(float64(factors.UniqueUsers-1)*0.012, 0, 0.05)
	}

	floor, ceiling := confidenceBounds(factors.Verification)
	return math.Round(clampFloat(score, floor, ceiling)*100) / 100
}

func verificationConfidenceAdjustment(verification string) float64 {
	switch strings.TrimSpace(verification) {
	case "local_verified":
		return 0.18
	case "path_mapped":
		return 0.11
	case "server_reported":
		return 0.03
	case "unmapped":
		return -0.14
	default:
		return -0.02
	}
}

func actionConfidenceAdjustment(action Action) float64 {
	switch action {
	case ActionReviewNeverWatchedMovie:
		return 0.02
	case ActionReviewAbandonedSeries:
		return 0.03
	case ActionReviewInactiveMovie:
		return -0.015
	case ActionReviewInactiveSeries:
		return -0.005
	default:
		return 0
	}
}

func confidenceBounds(verification string) (float64, float64) {
	switch strings.TrimSpace(verification) {
	case "local_verified":
		return 0.70, 0.97
	case "path_mapped":
		return 0.66, 0.90
	case "server_reported":
		return 0.60, 0.82
	case "unmapped":
		return 0, 0.55
	default:
		return 0.50, 0.76
	}
}

func confidenceBasisLabel(factors activityConfidenceFactors) string {
	parts := []string{verificationLabel(factors.Verification)}
	if factors.ThresholdDays > 0 {
		parts = append(parts, strconv.Itoa(factors.EvidenceDays)+" days vs "+strconv.Itoa(factors.ThresholdDays)+" day threshold")
	}
	if factors.ItemCount > 1 {
		parts = append(parts, strconv.Itoa(factors.ItemCount)+" affected files")
	}
	if factors.PlayCount > 0 {
		parts = append(parts, strconv.Itoa(factors.PlayCount)+" prior plays")
	}
	if factors.UniqueUsers > 0 {
		parts = append(parts, strconv.Itoa(factors.UniqueUsers)+" watched users")
	}
	if factors.MatchConfidence > 0 {
		parts = append(parts, "provider match "+strconv.Itoa(int(math.Round(clampFloat(factors.MatchConfidence, 0, 1)*100)))+"%")
	}
	return strings.Join(parts, "; ")
}

func clampFloat(value float64, minimum float64, maximum float64) float64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func stableID(value string) string {
	sum := sha1.Sum([]byte(value))
	return "rec_" + hex.EncodeToString(sum[:8])
}
