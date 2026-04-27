package campaigns

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

const (
	sourcePrefix = "campaign:"
)

func EvaluateRule(rule Rule, candidate Candidate, now time.Time) RuleResult {
	field := valueForField(rule.Field, candidate, now)
	result := RuleResult{Rule: rule}
	switch rule.Operator {
	case OperatorEquals:
		result.Matched = strings.EqualFold(field.text, strings.TrimSpace(rule.Value))
	case OperatorNotEquals:
		result.Matched = !strings.EqualFold(field.text, strings.TrimSpace(rule.Value))
	case OperatorIn:
		result.Matched = stringIn(field.text, rule.Values)
	case OperatorNotIn:
		result.Matched = !stringIn(field.text, rule.Values)
	case OperatorGreaterThan:
		result.Matched = compareNumber(field, rule.Value, func(left, right float64) bool { return left > right })
	case OperatorGreaterOrEqual:
		result.Matched = compareNumber(field, rule.Value, func(left, right float64) bool { return left >= right })
	case OperatorLessThan:
		result.Matched = compareNumber(field, rule.Value, func(left, right float64) bool { return left < right })
	case OperatorLessOrEqual:
		result.Matched = compareNumber(field, rule.Value, func(left, right float64) bool { return left <= right })
	case OperatorIsEmpty:
		result.Matched = field.empty
	case OperatorIsNotEmpty:
		result.Matched = !field.empty
	default:
		result.Matched = false
	}
	result.Reason = ruleReason(rule, field, result.Matched)
	return result
}

func Simulate(campaign Campaign, candidates []Candidate, now time.Time) Result {
	result := Result{
		CampaignID: campaign.ID,
		Enabled:    campaign.Enabled,
		Items:      []ResultItem{},
	}
	var confidenceSum float64
	for _, candidate := range candidates {
		if !campaignTargetsCandidate(campaign, candidate) {
			continue
		}
		item := evaluateCandidate(campaign, candidate, now)
		if len(item.MatchedRules) == 0 {
			continue
		}
		item.SuppressionReasons = append(item.SuppressionReasons, suppressionReasons(campaign, candidate)...)
		item.Suppressed = len(item.SuppressionReasons) > 0
		result.Items = append(result.Items, item)
		if item.Suppressed {
			result.Suppressed++
			continue
		}
		result.Matched++
		result.TotalEstimatedSavingsBytes += candidate.EstimatedSavingsBytes
		result.TotalVerifiedSavingsBytes += candidate.VerifiedSavingsBytes
		if result.ConfidenceMin == 0 || candidate.Confidence < result.ConfidenceMin {
			result.ConfidenceMin = candidate.Confidence
		}
		if candidate.Confidence > result.ConfidenceMax {
			result.ConfidenceMax = candidate.Confidence
		}
		confidenceSum += candidate.Confidence
	}
	if result.Matched > 0 {
		result.ConfidenceAverage = roundConfidence(confidenceSum / float64(result.Matched))
		result.ConfidenceMin = roundConfidence(result.ConfidenceMin)
		result.ConfidenceMax = roundConfidence(result.ConfidenceMax)
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		left := result.Items[i]
		right := result.Items[j]
		if left.Suppressed != right.Suppressed {
			return !left.Suppressed
		}
		if left.Candidate.EstimatedSavingsBytes == right.Candidate.EstimatedSavingsBytes {
			return left.Candidate.Title < right.Candidate.Title
		}
		return left.Candidate.EstimatedSavingsBytes > right.Candidate.EstimatedSavingsBytes
	})
	return result
}

func RecommendationForMatch(campaign Campaign, runID string, item ResultItem) recommendations.Recommendation {
	candidate := item.Candidate
	source := sourcePrefix + campaign.ID
	evidence := map[string]string{
		"campaignId":            campaign.ID,
		"campaignName":          campaign.Name,
		"campaignRunId":         runID,
		"matchedRules":          matchedRulesSummary(item.MatchedRules),
		"suppressionReasons":    strings.Join(item.SuppressionReasons, ", "),
		"subjectKind":           candidate.Kind,
		"subjectTitle":          candidate.Title,
		"itemCount":             strconv.Itoa(len(candidate.AffectedPaths)),
		"estimatedSavingsBytes": strconv.FormatInt(candidate.EstimatedSavingsBytes, 10),
		"verifiedSavingsBytes":  strconv.FormatInt(candidate.VerifiedSavingsBytes, 10),
		"storageBasis":          candidate.Verification,
		"storageCertainty":      storageCertainty(candidate.Verification),
		"confidenceBasis":       "campaign rule match with media-server activity and local path evidence",
	}
	for key, value := range candidate.Evidence {
		if strings.TrimSpace(value) != "" {
			evidence[key] = value
		}
	}
	return recommendations.Recommendation{
		ID:              stableID(source + ":" + candidate.Key),
		Action:          recommendations.ActionReviewCampaignMatch,
		State:           recommendations.StateNew,
		Title:           candidate.Title,
		Explanation:     campaign.Name + " matched this item. Review the campaign evidence and media-server activity before taking any manual cleanup action.",
		SpaceSavedBytes: candidate.EstimatedSavingsBytes,
		Confidence:      roundConfidence(candidate.Confidence),
		Source:          source,
		AffectedPaths:   append([]string(nil), candidate.AffectedPaths...),
		Destructive:     false,
		ServerID:        candidate.ServerID,
		ExternalItemID:  candidate.ExternalItemID,
		LastPlayedAt:    candidate.LastPlayedAt,
		PlayCount:       candidate.PlayCount,
		UniqueUsers:     candidate.UniqueUsers,
		FavoriteCount:   candidate.FavoriteCount,
		Verification:    candidate.Verification,
		Evidence:        evidence,
	}.WithDefaults()
}

func CandidatesFromActivity(items []recommendations.ActivityMedia, now time.Time) []Candidate {
	movies := make([]Candidate, 0, len(items))
	series := map[string]*Candidate{}
	for _, item := range items {
		path := strings.TrimSpace(item.Path)
		if item.SizeBytes <= 0 || path == "" {
			continue
		}
		if isSeriesActivityItem(item) {
			key := item.ServerID + ":" + firstNonEmpty(item.ParentExternalItemID, item.ExternalItemID)
			if key == item.ServerID+":" {
				continue
			}
			candidate := series[key]
			if candidate == nil {
				title := firstNonEmpty(item.ParentTitle, item.Title)
				kind := "series"
				if activityLooksAnime(item) {
					kind = "anime"
				}
				candidate = &Candidate{
					Key:            "activity:" + key,
					ServerID:       item.ServerID,
					ExternalItemID: firstNonEmpty(item.ParentExternalItemID, item.ExternalItemID),
					Title:          title,
					Kind:           kind,
					LibraryName:    item.LibraryName,
					Verification:   strings.TrimSpace(item.Verification),
					Confidence:     normalizedConfidence(item.MatchConfidence),
					AddedAt:        item.AddedAt,
					LastPlayedAt:   item.LastPlayedAt,
					Evidence:       map[string]string{"source": "media-server-activity"},
				}
				series[key] = candidate
			}
			candidate.EstimatedSavingsBytes += item.SizeBytes
			if item.Verification == "local_verified" {
				candidate.VerifiedSavingsBytes += item.SizeBytes
			}
			candidate.AffectedPaths = append(candidate.AffectedPaths, path)
			candidate.PlayCount += item.PlayCount
			if item.UniqueUsers > candidate.UniqueUsers {
				candidate.UniqueUsers = item.UniqueUsers
			}
			candidate.FavoriteCount += item.FavoriteCount
			candidate.Verification = weakerVerification(candidate.Verification, item.Verification)
			candidate.Confidence = weakerConfidence(candidate.Confidence, item.MatchConfidence)
			if !item.AddedAt.IsZero() && (candidate.AddedAt.IsZero() || item.AddedAt.After(candidate.AddedAt)) {
				candidate.AddedAt = item.AddedAt
			}
			if !item.LastPlayedAt.IsZero() && (candidate.LastPlayedAt.IsZero() || item.LastPlayedAt.After(candidate.LastPlayedAt)) {
				candidate.LastPlayedAt = item.LastPlayedAt
			}
			continue
		}
		candidate := candidateFromActivity(item, path)
		if candidate.Kind == "" {
			candidate.Kind = "movie"
		}
		movies = append(movies, candidate)
	}
	candidates := append([]Candidate{}, movies...)
	keys := make([]string, 0, len(series))
	for key := range series {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		candidate := *series[key]
		sort.Strings(candidate.AffectedPaths)
		candidate.Evidence["itemCount"] = strconv.Itoa(len(candidate.AffectedPaths))
		candidates = append(candidates, candidate)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].EstimatedSavingsBytes == candidates[j].EstimatedSavingsBytes {
			return candidates[i].Title < candidates[j].Title
		}
		return candidates[i].EstimatedSavingsBytes > candidates[j].EstimatedSavingsBytes
	})
	return candidates
}

type fieldValue struct {
	text  string
	num   float64
	empty bool
}

func valueForField(field Field, candidate Candidate, now time.Time) fieldValue {
	switch field {
	case FieldKind:
		return textValue(candidate.Kind)
	case FieldLibraryName:
		return textValue(candidate.LibraryName)
	case FieldVerification:
		return textValue(candidate.Verification)
	case FieldStorageBytes, FieldEstimatedSavingsBytes:
		return numberValue(float64(candidate.EstimatedSavingsBytes))
	case FieldVerifiedSavingsBytes:
		return numberValue(float64(candidate.VerifiedSavingsBytes))
	case FieldLastPlayedDays:
		return daysValue(candidate.LastPlayedAt, now)
	case FieldAddedDays:
		return daysValue(candidate.AddedAt, now)
	case FieldPlayCount:
		return numberValue(float64(candidate.PlayCount))
	case FieldUniqueUsers:
		return numberValue(float64(candidate.UniqueUsers))
	case FieldFavoriteCount:
		return numberValue(float64(candidate.FavoriteCount))
	case FieldConfidence:
		return numberValue(candidate.Confidence)
	default:
		return fieldValue{empty: true}
	}
}

func textValue(value string) fieldValue {
	trimmed := strings.TrimSpace(value)
	return fieldValue{text: trimmed, empty: trimmed == ""}
}

func numberValue(value float64) fieldValue {
	return fieldValue{text: strconv.FormatFloat(value, 'f', -1, 64), num: value}
}

func daysValue(value time.Time, now time.Time) fieldValue {
	if value.IsZero() {
		return fieldValue{empty: true}
	}
	days := int(now.Sub(value).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return fieldValue{text: strconv.Itoa(days), num: float64(days)}
}

func compareNumber(field fieldValue, wanted string, compare func(float64, float64) bool) bool {
	if field.empty {
		return false
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(wanted), 64)
	if err != nil {
		return false
	}
	return compare(field.num, value)
}

func stringIn(value string, values []string) bool {
	for _, option := range values {
		if strings.EqualFold(strings.TrimSpace(option), strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

func ruleReason(rule Rule, field fieldValue, matched bool) string {
	state := "did not match"
	if matched {
		state = "matched"
	}
	switch rule.Operator {
	case OperatorIn, OperatorNotIn:
		return fmt.Sprintf("%s %s %s against %s", rule.Field, rule.Operator, state, strings.Join(rule.Values, ", "))
	case OperatorIsEmpty, OperatorIsNotEmpty:
		return fmt.Sprintf("%s %s %s", rule.Field, rule.Operator, state)
	default:
		return fmt.Sprintf("%s value %q %s %s %q", rule.Field, field.text, state, rule.Operator, rule.Value)
	}
}

func evaluateCandidate(campaign Campaign, candidate Candidate, now time.Time) ResultItem {
	item := ResultItem{Candidate: candidate, MatchedRules: []RuleResult{}}
	if len(campaign.Rules) == 0 {
		item.MatchedRules = append(item.MatchedRules, RuleResult{
			Rule:    Rule{Field: FieldKind, Operator: OperatorIsNotEmpty},
			Matched: true,
			Reason:  "campaign has no rules; target matched",
		})
		return item
	}
	requireAll := campaign.RequireAllRules
	matched := 0
	for _, rule := range campaign.Rules {
		result := EvaluateRule(rule, candidate, now)
		if result.Matched {
			matched++
			item.MatchedRules = append(item.MatchedRules, result)
		} else if requireAll {
			return ResultItem{Candidate: candidate, MatchedRules: []RuleResult{}}
		}
	}
	if requireAll && matched != len(campaign.Rules) {
		return ResultItem{Candidate: candidate, MatchedRules: []RuleResult{}}
	}
	if !requireAll && matched == 0 {
		return ResultItem{Candidate: candidate, MatchedRules: []RuleResult{}}
	}
	return item
}

func campaignTargetsCandidate(campaign Campaign, candidate Candidate) bool {
	if len(campaign.TargetKinds) > 0 && !stringIn(candidate.Kind, campaign.TargetKinds) {
		return false
	}
	if len(campaign.TargetLibraryNames) > 0 && !stringIn(candidate.LibraryName, campaign.TargetLibraryNames) {
		return false
	}
	return true
}

func suppressionReasons(campaign Campaign, candidate Candidate) []string {
	var reasons []string
	if candidate.FavoriteCount > 0 {
		reasons = append(reasons, "favorite_or_protected")
	}
	if campaign.MinimumConfidence > 0 && candidate.Confidence < campaign.MinimumConfidence {
		reasons = append(reasons, "below_campaign_confidence")
	}
	if candidate.Confidence < 0.55 {
		reasons = append(reasons, "low_evidence_confidence")
	}
	if len(candidate.AffectedPaths) == 0 {
		reasons = append(reasons, "missing_affected_paths")
	}
	if campaign.MinimumStorageBytes > 0 && candidate.EstimatedSavingsBytes < campaign.MinimumStorageBytes {
		reasons = append(reasons, "below_campaign_storage_threshold")
	}
	if candidate.EstimatedSavingsBytes <= 0 {
		reasons = append(reasons, "missing_storage_estimate")
	}
	return reasons
}

func matchedRulesSummary(results []RuleResult) string {
	parts := make([]string, 0, len(results))
	for _, result := range results {
		parts = append(parts, string(result.Rule.Field)+":"+string(result.Rule.Operator))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func storageCertainty(verification string) string {
	switch strings.TrimSpace(verification) {
	case "local_verified":
		return "local_verified"
	case "path_mapped":
		return "mapped_estimate"
	case "server_reported":
		return "server_reported"
	default:
		return "unmapped"
	}
}

func candidateFromActivity(item recommendations.ActivityMedia, path string) Candidate {
	estimated := item.SizeBytes
	verified := int64(0)
	if item.Verification == "local_verified" {
		verified = item.SizeBytes
	}
	title := firstNonEmpty(item.Title, item.ParentTitle)
	kind := strings.ToLower(strings.TrimSpace(item.Kind))
	return Candidate{
		Key:                   "activity:" + item.ServerID + ":" + item.ExternalItemID + ":" + path,
		ServerID:              item.ServerID,
		ExternalItemID:        item.ExternalItemID,
		Title:                 title,
		Kind:                  kind,
		LibraryName:           item.LibraryName,
		Verification:          strings.TrimSpace(item.Verification),
		EstimatedSavingsBytes: estimated,
		VerifiedSavingsBytes:  verified,
		Confidence:            normalizedConfidence(item.MatchConfidence),
		AddedAt:               item.AddedAt,
		LastPlayedAt:          item.LastPlayedAt,
		PlayCount:             item.PlayCount,
		UniqueUsers:           item.UniqueUsers,
		FavoriteCount:         item.FavoriteCount,
		AffectedPaths:         []string{path},
		Evidence:              map[string]string{"source": "media-server-activity", "itemCount": "1"},
	}
}

func isSeriesActivityItem(item recommendations.ActivityMedia) bool {
	kind := strings.ToLower(strings.TrimSpace(item.Kind))
	switch kind {
	case "episode", "season", "series":
		return true
	case "video":
		return strings.TrimSpace(item.ParentExternalItemID) != "" || activityLooksAnime(item)
	default:
		return strings.TrimSpace(item.ParentExternalItemID) != ""
	}
}

func activityLooksAnime(item recommendations.ActivityMedia) bool {
	value := strings.ToLower(item.LibraryName + " " + item.ParentTitle + " " + item.Title)
	return strings.Contains(value, "anime")
}

func weakerVerification(left string, right string) string {
	rank := map[string]int{
		"local_verified":  4,
		"path_mapped":     3,
		"server_reported": 2,
		"unmapped":        1,
		"":                0,
	}
	if rank[strings.TrimSpace(left)] <= rank[strings.TrimSpace(right)] {
		if strings.TrimSpace(left) == "" {
			return strings.TrimSpace(right)
		}
		return strings.TrimSpace(left)
	}
	return strings.TrimSpace(right)
}

func weakerConfidence(current float64, next float64) float64 {
	next = normalizedConfidence(next)
	if current <= 0 {
		return next
	}
	if next <= 0 {
		return current
	}
	if next < current {
		return next
	}
	return current
}

func normalizedConfidence(value float64) float64 {
	if value <= 0 {
		return 0.6
	}
	if value > 1 {
		return 1
	}
	return roundConfidence(value)
}

func roundConfidence(value float64) float64 {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	return math.Round(value*1000) / 1000
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stableID(value string) string {
	sum := sha1.Sum([]byte(value))
	return "rec_" + hex.EncodeToString(sum[:8])
}
