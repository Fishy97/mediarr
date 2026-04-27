package campaigns

import "time"

type Field string
type Operator string

const (
	FieldKind                  Field = "kind"
	FieldLibraryName           Field = "libraryName"
	FieldVerification          Field = "verification"
	FieldStorageBytes          Field = "storageBytes"
	FieldEstimatedSavingsBytes Field = "estimatedSavingsBytes"
	FieldVerifiedSavingsBytes  Field = "verifiedSavingsBytes"
	FieldLastPlayedDays        Field = "lastPlayedDays"
	FieldAddedDays             Field = "addedDays"
	FieldPlayCount             Field = "playCount"
	FieldUniqueUsers           Field = "uniqueUsers"
	FieldFavoriteCount         Field = "favoriteCount"
	FieldConfidence            Field = "confidence"
)

const (
	OperatorEquals         Operator = "equals"
	OperatorNotEquals      Operator = "not_equals"
	OperatorIn             Operator = "in"
	OperatorNotIn          Operator = "not_in"
	OperatorGreaterThan    Operator = "greater_than"
	OperatorGreaterOrEqual Operator = "greater_or_equal"
	OperatorLessThan       Operator = "less_than"
	OperatorLessOrEqual    Operator = "less_or_equal"
	OperatorIsEmpty        Operator = "is_empty"
	OperatorIsNotEmpty     Operator = "is_not_empty"
)

type Campaign struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	Enabled             bool      `json:"enabled"`
	TargetKinds         []string  `json:"targetKinds"`
	TargetLibraryNames  []string  `json:"targetLibraryNames"`
	Rules               []Rule    `json:"rules"`
	RequireAllRules     bool      `json:"requireAllRules"`
	MinimumConfidence   float64   `json:"minimumConfidence"`
	MinimumStorageBytes int64     `json:"minimumStorageBytes"`
	CreatedAt           time.Time `json:"createdAt,omitempty"`
	UpdatedAt           time.Time `json:"updatedAt,omitempty"`
	LastRunAt           time.Time `json:"lastRunAt,omitempty"`
}

type Rule struct {
	Field    Field    `json:"field"`
	Operator Operator `json:"operator"`
	Value    string   `json:"value,omitempty"`
	Values   []string `json:"values,omitempty"`
}

type Candidate struct {
	Key                   string            `json:"key"`
	ServerID              string            `json:"serverId,omitempty"`
	ExternalItemID        string            `json:"externalItemId,omitempty"`
	Title                 string            `json:"title"`
	Kind                  string            `json:"kind"`
	LibraryName           string            `json:"libraryName,omitempty"`
	Verification          string            `json:"verification,omitempty"`
	EstimatedSavingsBytes int64             `json:"estimatedSavingsBytes"`
	VerifiedSavingsBytes  int64             `json:"verifiedSavingsBytes"`
	Confidence            float64           `json:"confidence"`
	AddedAt               time.Time         `json:"addedAt,omitempty"`
	LastPlayedAt          time.Time         `json:"lastPlayedAt,omitempty"`
	PlayCount             int               `json:"playCount"`
	UniqueUsers           int               `json:"uniqueUsers"`
	FavoriteCount         int               `json:"favoriteCount"`
	AffectedPaths         []string          `json:"affectedPaths"`
	Evidence              map[string]string `json:"evidence,omitempty"`
}

type RuleResult struct {
	Rule    Rule   `json:"rule"`
	Matched bool   `json:"matched"`
	Reason  string `json:"reason"`
}

type ResultItem struct {
	Candidate          Candidate    `json:"candidate"`
	MatchedRules       []RuleResult `json:"matchedRules"`
	SuppressionReasons []string     `json:"suppressionReasons"`
	Suppressed         bool         `json:"suppressed"`
}

type Result struct {
	CampaignID                 string       `json:"campaignId"`
	Enabled                    bool         `json:"enabled"`
	Matched                    int          `json:"matched"`
	Suppressed                 int          `json:"suppressed"`
	TotalEstimatedSavingsBytes int64        `json:"totalEstimatedSavingsBytes"`
	TotalVerifiedSavingsBytes  int64        `json:"totalVerifiedSavingsBytes"`
	ConfidenceMin              float64      `json:"confidenceMin"`
	ConfidenceAverage          float64      `json:"confidenceAverage"`
	ConfidenceMax              float64      `json:"confidenceMax"`
	Items                      []ResultItem `json:"items"`
}

type Run struct {
	ID                    string    `json:"id"`
	CampaignID            string    `json:"campaignId"`
	Status                string    `json:"status"`
	Matched               int       `json:"matched"`
	Suppressed            int       `json:"suppressed"`
	EstimatedSavingsBytes int64     `json:"estimatedSavingsBytes"`
	VerifiedSavingsBytes  int64     `json:"verifiedSavingsBytes"`
	Error                 string    `json:"error,omitempty"`
	StartedAt             time.Time `json:"startedAt"`
	CompletedAt           time.Time `json:"completedAt,omitempty"`
}
