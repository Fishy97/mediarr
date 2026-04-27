package stewardship

import "testing"

func TestPlanLeavingSoonPublicationRequiresVerificationAndExternalIDs(t *testing.T) {
	plan := PlanLeavingSoonPublication(PublicationInput{
		CampaignID:          "cmp_1",
		ServerID:            "jellyfin",
		CollectionTitle:     "Leaving Soon",
		MinimumVerification: "path_mapped",
		Items: []PublicationCandidate{
			{ExternalItemID: "ok", Title: "Verified", Verification: "local_verified", EstimatedBytes: 100},
			{ExternalItemID: "mapped", Title: "Mapped", Verification: "path_mapped", EstimatedBytes: 90},
			{ExternalItemID: "server", Title: "Server Only", Verification: "server_reported", EstimatedBytes: 80},
			{Title: "Missing ID", Verification: "local_verified", EstimatedBytes: 70},
		},
	})

	if !plan.DryRun || plan.PublishableItems != 2 || plan.BlockedItems != 2 {
		t.Fatalf("unexpected plan counts: %#v", plan)
	}
	if plan.PublishableEstimatedBytes != 190 || plan.BlockedEstimatedBytes != 150 {
		t.Fatalf("unexpected plan bytes: %#v", plan)
	}
	if len(plan.Items) != 4 || plan.Items[2].BlockedReason != "below_verification_threshold" || plan.Items[3].BlockedReason != "missing_external_item_id" {
		t.Fatalf("unexpected item blockers: %#v", plan.Items)
	}
}
