package acceptance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/integrations"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

type JellyfinConfig struct {
	BaseURL                  string
	APIKey                   string
	PathMappings             []database.PathMapping
	StoreDir                 string
	ReportDir                string
	RedactTitles             bool
	RequireLocalVerification bool
	Timeout                  time.Duration
}

type ReportOptions struct {
	TargetID                 string
	GeneratedAt              time.Time
	RedactTitles             bool
	RequireLocalVerification bool
}

type Report struct {
	TargetID           string                  `json:"targetId"`
	GeneratedAt        time.Time               `json:"generatedAt"`
	Server             ReportServer            `json:"server"`
	Summary            ReportSummary           `json:"summary"`
	Warnings           []string                `json:"warnings"`
	ProgressSamples    []ProgressSample        `json:"progressSamples"`
	TopRecommendations []RecommendationSummary `json:"topRecommendations"`
}

type ReportServer struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

type ReportSummary struct {
	Libraries                      int   `json:"libraries"`
	Users                          int   `json:"users"`
	Movies                         int   `json:"movies"`
	Series                         int   `json:"series"`
	Episodes                       int   `json:"episodes"`
	Videos                         int   `json:"videos"`
	AnimeItems                     int   `json:"animeItems"`
	Files                          int   `json:"files"`
	ActivityRollups                int   `json:"activityRollups"`
	Recommendations                int   `json:"recommendations"`
	DestructiveRecommendations     int   `json:"destructiveRecommendations"`
	ServerReportedBytes            int64 `json:"serverReportedBytes"`
	LocallyVerifiedBytes           int64 `json:"locallyVerifiedBytes"`
	RecommendationBytes            int64 `json:"recommendationBytes"`
	UnmappedFiles                  int   `json:"unmappedFiles"`
	FilesMissingSize               int   `json:"filesMissingSize"`
	AcceptedForRecommendationBytes int64 `json:"acceptedForRecommendationBytes"`
}

type ProgressSample struct {
	Phase        string `json:"phase"`
	Message      string `json:"message"`
	CurrentLabel string `json:"currentLabel,omitempty"`
	Processed    int    `json:"processed"`
	Total        int    `json:"total"`
}

type RecommendationSummary struct {
	ID              string   `json:"id"`
	Action          string   `json:"action"`
	Title           string   `json:"title"`
	SpaceSavedBytes int64    `json:"spaceSavedBytes"`
	Confidence      float64  `json:"confidence"`
	Source          string   `json:"source"`
	Verification    string   `json:"verification"`
	AffectedPaths   []string `json:"affectedPaths,omitempty"`
}

type ReportFiles struct {
	JSONPath     string `json:"jsonPath"`
	MarkdownPath string `json:"markdownPath"`
}

func RunJellyfin(ctx context.Context, cfg JellyfinConfig) (Report, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	apiKey := strings.TrimSpace(cfg.APIKey)
	if baseURL == "" {
		return Report{}, errors.New("jellyfin acceptance requires MEDIARR_ACCEPTANCE_JELLYFIN_URL")
	}
	if apiKey == "" {
		return Report{}, errors.New("jellyfin acceptance requires MEDIARR_ACCEPTANCE_JELLYFIN_API_KEY")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 6 * time.Hour
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	storeDir := strings.TrimSpace(cfg.StoreDir)
	if storeDir == "" {
		var err error
		storeDir, err = os.MkdirTemp("", "mediarr-acceptance-*")
		if err != nil {
			return Report{}, err
		}
	}
	store, err := database.Open(storeDir)
	if err != nil {
		return Report{}, err
	}
	defer store.Close()

	progress := []integrations.Progress{}
	snapshot, err := integrations.SyncJellyfin(ctx, integrations.Options{
		JellyfinURL: baseURL,
		JellyfinKey: apiKey,
		Progress: func(update integrations.Progress) {
			progress = append(progress, update)
		},
	}, cfg.PathMappings)
	if err != nil {
		return Report{}, err
	}
	if snapshot.Job.ItemsImported == 0 || len(snapshot.Items) == 0 {
		return Report{}, errors.New("jellyfin acceptance imported zero items")
	}
	if err := store.ReplaceMediaServerSnapshot(snapshot); err != nil {
		return Report{}, err
	}
	media, err := store.ListActivityRecommendationMedia()
	if err != nil {
		return Report{}, err
	}
	recs := recommendations.Engine{}.GenerateActivity(media, time.Now().UTC())
	for _, rec := range recs {
		if rec.Destructive {
			return Report{}, fmt.Errorf("destructive recommendation generated: %s", rec.ID)
		}
	}
	return BuildReport(snapshot, recs, progress, ReportOptions{
		TargetID:                 "jellyfin",
		GeneratedAt:              time.Now().UTC(),
		RedactTitles:             cfg.RedactTitles,
		RequireLocalVerification: cfg.RequireLocalVerification,
	}), nil
}

func ParsePathMappings(raw string, serverID string) ([]database.PathMapping, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		serverID = "jellyfin"
	}
	entries := strings.Split(raw, ";")
	mappings := make([]database.PathMapping, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, "=")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("invalid path mapping %q, expected /server/path=/local/path", entry)
		}
		mappings = append(mappings, database.PathMapping{
			ServerID:         serverID,
			ServerPathPrefix: strings.TrimSpace(parts[0]),
			LocalPathPrefix:  strings.TrimSpace(parts[1]),
		})
	}
	return mappings, nil
}

func BuildReport(snapshot database.MediaServerSnapshot, recs []recommendations.Recommendation, progress []integrations.Progress, options ReportOptions) Report {
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	report := Report{
		TargetID:    firstReportValue(options.TargetID, snapshot.Server.ID, "jellyfin"),
		GeneratedAt: generatedAt,
		Server: ReportServer{
			Name:   redactValue(snapshot.Server.Name, options.RedactTitles),
			Kind:   snapshot.Server.Kind,
			Status: snapshot.Server.Status,
		},
		Summary: ReportSummary{
			Libraries:       len(snapshot.Libraries),
			Users:           len(snapshot.Users),
			Files:           len(snapshot.Files),
			ActivityRollups: len(snapshot.Rollups),
		},
	}

	librariesByID := map[string]database.MediaServerLibrary{}
	for _, library := range snapshot.Libraries {
		librariesByID[library.ExternalID] = library
	}
	for _, item := range snapshot.Items {
		library := librariesByID[item.LibraryExternalID]
		switch strings.ToLower(strings.TrimSpace(item.Kind)) {
		case "movie":
			report.Summary.Movies++
		case "series":
			report.Summary.Series++
		case "episode":
			report.Summary.Episodes++
		case "video":
			report.Summary.Videos++
		}
		if looksAnime(library.Name, item.Path, item.Title) {
			report.Summary.AnimeItems++
		}
	}
	for _, file := range snapshot.Files {
		if file.SizeBytes <= 0 {
			report.Summary.FilesMissingSize++
			continue
		}
		report.Summary.ServerReportedBytes += file.SizeBytes
		if file.Verification == "local_verified" {
			report.Summary.LocallyVerifiedBytes += file.SizeBytes
		}
		if strings.TrimSpace(file.LocalPath) == "" || file.Verification == "unmapped" {
			report.Summary.UnmappedFiles++
		}
	}

	sort.Slice(recs, func(i, j int) bool {
		if recs[i].SpaceSavedBytes == recs[j].SpaceSavedBytes {
			return recs[i].ID < recs[j].ID
		}
		return recs[i].SpaceSavedBytes > recs[j].SpaceSavedBytes
	})
	report.Summary.Recommendations = len(recs)
	for _, rec := range recs {
		if rec.Destructive {
			report.Summary.DestructiveRecommendations++
		}
		report.Summary.RecommendationBytes += rec.SpaceSavedBytes
		if rec.Verification == "local_verified" || rec.Verification == "path_mapped" {
			report.Summary.AcceptedForRecommendationBytes += rec.SpaceSavedBytes
		}
		if len(report.TopRecommendations) < 25 {
			report.TopRecommendations = append(report.TopRecommendations, RecommendationSummary{
				ID:              rec.ID,
				Action:          string(rec.Action),
				Title:           redactValue(rec.Title, options.RedactTitles),
				SpaceSavedBytes: rec.SpaceSavedBytes,
				Confidence:      rec.Confidence,
				Source:          rec.Source,
				Verification:    rec.Verification,
				AffectedPaths:   redactPaths(rec.AffectedPaths, options.RedactTitles),
			})
		}
	}
	for _, sample := range progressSamples(progress, options.RedactTitles) {
		report.ProgressSamples = append(report.ProgressSamples, sample)
	}
	report.Warnings = reportWarnings(report, options)
	return report
}

func WriteReportFiles(report Report, dir string) (ReportFiles, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = "acceptance-reports"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ReportFiles{}, err
	}
	stamp := report.GeneratedAt.UTC().Format("20060102-150405")
	base := "jellyfin-live-" + stamp
	jsonPath := filepath.Join(dir, base+".json")
	markdownPath := filepath.Join(dir, base+".md")
	jsonBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return ReportFiles{}, err
	}
	jsonBytes = append(jsonBytes, '\n')
	if err := os.WriteFile(jsonPath, jsonBytes, 0o600); err != nil {
		return ReportFiles{}, err
	}
	if err := os.WriteFile(markdownPath, []byte(markdownReport(report)), 0o600); err != nil {
		return ReportFiles{}, err
	}
	return ReportFiles{JSONPath: jsonPath, MarkdownPath: markdownPath}, nil
}

func reportWarnings(report Report, options ReportOptions) []string {
	warnings := []string{}
	if report.Summary.Movies == 0 && report.Summary.Episodes == 0 && report.Summary.Series == 0 {
		warnings = append(warnings, "No movies, series, or episodes were imported.")
	}
	if report.Summary.ActivityRollups == 0 {
		warnings = append(warnings, "No activity rollups were imported, so last-used recommendations cannot be trusted yet.")
	}
	if report.Summary.FilesMissingSize > 0 {
		warnings = append(warnings, fmt.Sprintf("%d imported files are missing size data.", report.Summary.FilesMissingSize))
	}
	if options.RequireLocalVerification && report.Summary.LocallyVerifiedBytes < report.Summary.ServerReportedBytes {
		warnings = append(warnings, "local verification is incomplete; configure path mappings or read-only media mounts before trusting reclaimable space.")
	}
	if report.Summary.UnmappedFiles > 0 {
		warnings = append(warnings, fmt.Sprintf("%d files are unmapped and should remain blocked from cleanup recommendations.", report.Summary.UnmappedFiles))
	}
	if report.Summary.DestructiveRecommendations > 0 {
		warnings = append(warnings, "Destructive recommendations were generated; this violates Mediarr safety policy.")
	}
	if report.Summary.Recommendations == 0 {
		warnings = append(warnings, "No recommendations were generated from the imported activity data.")
	}
	return warnings
}

func progressSamples(progress []integrations.Progress, redact bool) []ProgressSample {
	if len(progress) == 0 {
		return nil
	}
	const maxSamples = 200
	step := 1
	if len(progress) > maxSamples {
		step = len(progress) / maxSamples
		if step < 1 {
			step = 1
		}
	}
	samples := []ProgressSample{}
	for index, update := range progress {
		if index%step != 0 && index != len(progress)-1 {
			continue
		}
		if len(samples) >= maxSamples && index != len(progress)-1 {
			continue
		}
		samples = append(samples, ProgressSample{
			Phase:        update.Phase,
			Message:      redactProgressMessage(update, redact),
			CurrentLabel: redactValue(update.CurrentLabel, redact),
			Processed:    update.Processed,
			Total:        update.Total,
		})
	}
	return samples
}

func markdownReport(report Report) string {
	var builder strings.Builder
	builder.WriteString("# Mediarr Live Jellyfin Acceptance Report\n\n")
	builder.WriteString("- Generated: " + report.GeneratedAt.Format(time.RFC3339) + "\n")
	builder.WriteString("- Server: " + report.Server.Name + "\n")
	builder.WriteString("- Status: " + report.Server.Status + "\n\n")
	builder.WriteString("## Summary\n\n")
	builder.WriteString(fmt.Sprintf("- Movies: %d\n", report.Summary.Movies))
	builder.WriteString(fmt.Sprintf("- Series: %d\n", report.Summary.Series))
	builder.WriteString(fmt.Sprintf("- Episodes: %d\n", report.Summary.Episodes))
	builder.WriteString(fmt.Sprintf("- Anime/library-matched items: %d\n", report.Summary.AnimeItems))
	builder.WriteString(fmt.Sprintf("- Files: %d\n", report.Summary.Files))
	builder.WriteString(fmt.Sprintf("- Activity rollups: %d\n", report.Summary.ActivityRollups))
	builder.WriteString(fmt.Sprintf("- Server-reported size: %s\n", formatBytes(report.Summary.ServerReportedBytes)))
	builder.WriteString(fmt.Sprintf("- Locally verified size: %s\n", formatBytes(report.Summary.LocallyVerifiedBytes)))
	builder.WriteString(fmt.Sprintf("- Recommendations: %d\n", report.Summary.Recommendations))
	builder.WriteString(fmt.Sprintf("- Potential reclaimable space: %s\n\n", formatBytes(report.Summary.RecommendationBytes)))
	if len(report.Warnings) > 0 {
		builder.WriteString("## Warnings\n\n")
		for _, warning := range report.Warnings {
			builder.WriteString("- " + warning + "\n")
		}
		builder.WriteString("\n")
	}
	if len(report.TopRecommendations) > 0 {
		builder.WriteString("## Top Recommendations\n\n")
		for _, rec := range report.TopRecommendations {
			builder.WriteString(fmt.Sprintf("- %s: %s, confidence %.2f, %s\n", rec.Title, formatBytes(rec.SpaceSavedBytes), rec.Confidence, rec.Verification))
		}
	}
	return builder.String()
}

func redactProgressMessage(update integrations.Progress, redact bool) string {
	if !redact {
		return update.Message
	}
	if strings.TrimSpace(update.CurrentLabel) == "" {
		return update.Message
	}
	return strings.ReplaceAll(update.Message, update.CurrentLabel, "[redacted]")
}

func redactValue(value string, redact bool) string {
	if !redact || strings.TrimSpace(value) == "" {
		return value
	}
	return "[redacted]"
}

func redactPaths(paths []string, redact bool) []string {
	if !redact {
		return append([]string(nil), paths...)
	}
	redacted := make([]string, 0, len(paths))
	for range paths {
		redacted = append(redacted, "[redacted]")
	}
	return redacted
}

func looksAnime(values ...string) bool {
	return strings.Contains(strings.ToLower(strings.Join(values, " ")), "anime")
}

func firstReportValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func formatBytes(bytes int64) string {
	if bytes < 0 {
		bytes = 0
	}
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	for _, suffix := range units {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.2f %s", value, suffix)
		}
	}
	return fmt.Sprintf("%.2f PB", value)
}
