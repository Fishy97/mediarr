package stewardship

import "errors"

func BuiltInCampaignTemplates() []CampaignTemplate {
	return []CampaignTemplate{
		template("cold-movies", "Cold Movies", "Movies not watched for a long time with trustworthy storage evidence.", []string{"movie"}, 0.72, []TemplateRule{
			{Field: "lastPlayedDays", Operator: "greater_or_equal", Value: "365"},
			{Field: "playCount", Operator: "greater_or_equal", Value: "1"},
		}),
		template("abandoned-series", "Abandoned Series", "Series with no recent household activity.", []string{"series"}, 0.72, []TemplateRule{
			{Field: "lastPlayedDays", Operator: "greater_or_equal", Value: "270"},
			{Field: "playCount", Operator: "greater_or_equal", Value: "1"},
		}),
		template("anime-backlog", "Anime Backlog", "Anime libraries with no viewing activity and meaningful storage impact.", []string{"anime"}, 0.68, []TemplateRule{
			{Field: "playCount", Operator: "less_or_equal", Value: "0"},
			{Field: "estimatedSavingsBytes", Operator: "greater_or_equal", Value: "10000000000"},
		}),
		template("verified-high-storage", "Verified High Storage", "Large locally verified candidates worth reviewing first.", []string{"movie", "series", "anime"}, 0.82, []TemplateRule{
			{Field: "verification", Operator: "equals", Value: "local_verified"},
			{Field: "verifiedSavingsBytes", Operator: "greater_or_equal", Value: "50000000000"},
		}),
		template("never-watched-large", "Never Watched Large Files", "Large files nobody has played yet.", []string{"movie", "series", "anime"}, 0.68, []TemplateRule{
			{Field: "playCount", Operator: "less_or_equal", Value: "0"},
			{Field: "estimatedSavingsBytes", Operator: "greater_or_equal", Value: "25000000000"},
		}),
		template("requested-never-watched", "Requested But Never Watched", "Requested media that became available but has not been watched.", []string{"movie", "series", "anime"}, 0.62, []TemplateRule{
			{Field: "playCount", Operator: "less_or_equal", Value: "0"},
			{Field: "addedDays", Operator: "greater_or_equal", Value: "90"},
		}),
	}
}

func CampaignFromTemplate(id string) (TemplateCampaignRef, error) {
	for _, candidate := range BuiltInCampaignTemplates() {
		if candidate.ID == id {
			return candidate.Campaign, nil
		}
	}
	return TemplateCampaignRef{}, errors.New("campaign template not found")
}

func template(id string, name string, description string, kinds []string, confidence float64, rules []TemplateRule) CampaignTemplate {
	return CampaignTemplate{
		ID:          id,
		Name:        name,
		Description: description,
		Campaign: TemplateCampaignRef{
			Name:                name,
			Description:         description,
			Enabled:             true,
			TargetKinds:         kinds,
			Rules:               rules,
			RequireAllRules:     true,
			MinimumConfidence:   confidence,
			MinimumStorageBytes: 0,
		},
	}
}
