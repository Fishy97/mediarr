package stewardship

import (
	"strings"
	"time"
)

func PlanLeavingSoonPublication(input PublicationInput) PublicationPlan {
	title := strings.TrimSpace(input.CollectionTitle)
	if title == "" {
		title = "Leaving Soon"
	}
	threshold := strings.TrimSpace(input.MinimumVerification)
	if threshold == "" {
		threshold = "path_mapped"
	}
	plan := PublicationPlan{
		CampaignID:      strings.TrimSpace(input.CampaignID),
		ServerID:        strings.TrimSpace(input.ServerID),
		CollectionTitle: title,
		DryRun:          true,
		Status:          "preview",
		Items:           make([]PublicationPlanItem, 0, len(input.Items)),
		CreatedAt:       nowUTC(),
	}
	for _, candidate := range input.Items {
		item := PublicationPlanItem{
			ExternalItemID: strings.TrimSpace(candidate.ExternalItemID),
			Title:          strings.TrimSpace(candidate.Title),
			Verification:   strings.TrimSpace(candidate.Verification),
			EstimatedBytes: maxInt64(candidate.EstimatedBytes, 0),
		}
		switch {
		case item.ExternalItemID == "":
			item.BlockedReason = "missing_external_item_id"
		case verificationRank(item.Verification) < verificationRank(threshold):
			item.BlockedReason = "below_verification_threshold"
		default:
			item.Publishable = true
		}
		if item.Publishable {
			plan.PublishableItems++
			plan.PublishableEstimatedBytes += item.EstimatedBytes
		} else {
			plan.BlockedItems++
			plan.BlockedEstimatedBytes += item.EstimatedBytes
		}
		plan.Items = append(plan.Items, item)
	}
	return plan
}

func verificationRank(value string) int {
	switch strings.TrimSpace(value) {
	case "local_verified":
		return 4
	case "path_mapped":
		return 3
	case "server_reported":
		return 2
	default:
		return 1
	}
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
