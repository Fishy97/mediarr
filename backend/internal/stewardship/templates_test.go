package stewardship

import "testing"

func TestBuiltInCampaignTemplatesAreEditableAndSafe(t *testing.T) {
	templates := BuiltInCampaignTemplates()
	if len(templates) < 6 {
		t.Fatalf("templates = %d, want at least 6", len(templates))
	}
	for _, template := range templates {
		if template.ID == "" || template.Name == "" || len(template.Campaign.Rules) == 0 {
			t.Fatalf("invalid template: %#v", template)
		}
		if template.Campaign.ID != "" {
			t.Fatalf("template campaign should not have a persisted id: %#v", template.Campaign)
		}
		if template.Campaign.MinimumConfidence < 0.55 {
			t.Fatalf("template %s has weak minimum confidence: %.2f", template.ID, template.Campaign.MinimumConfidence)
		}
	}
}

func TestCampaignFromTemplateAssignsNameAndPreservesRules(t *testing.T) {
	template := BuiltInCampaignTemplates()[0]
	campaign, err := CampaignFromTemplate(template.ID)
	if err != nil {
		t.Fatalf("CampaignFromTemplate returned error: %v", err)
	}
	if campaign.Name != template.Campaign.Name || len(campaign.Rules) != len(template.Campaign.Rules) {
		t.Fatalf("campaign not created from template: %#v", campaign)
	}
}
