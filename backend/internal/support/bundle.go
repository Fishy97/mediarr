package support

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/acceptance"
	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/recommendations"
)

type Config struct {
	Store     *database.Store
	OutputDir string
	Service   string
	Version   string
}

type Result struct {
	Path      string    `json:"path"`
	SizeBytes int64     `json:"sizeBytes"`
	Files     []string  `json:"files"`
	CreatedAt time.Time `json:"createdAt"`
}

type Manifest struct {
	Service     string    `json:"service"`
	Version     string    `json:"version"`
	GeneratedAt time.Time `json:"generatedAt"`
	Contents    []string  `json:"contents"`
	Safety      []string  `json:"safety"`
}

func CreateBundle(config Config) (Result, error) {
	if config.Store == nil {
		return Result{}, errors.New("support bundle requires a database store")
	}
	outputDir := strings.TrimSpace(config.OutputDir)
	if outputDir == "" {
		return Result{}, errors.New("support bundle output directory is required")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Result{}, err
	}
	generatedAt := time.Now().UTC()
	path := filepath.Join(outputDir, "mediarr-support-"+generatedAt.Format("20060102T150405.000000000Z")+".zip")
	redactor := newRedactor(config.Store)
	files, err := buildEntries(config, generatedAt)
	if err != nil {
		return Result{}, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return Result{}, err
	}
	archive := zip.NewWriter(file)
	names := make([]string, 0, len(files)+1)
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	manifest := Manifest{
		Service:     firstNonEmpty(config.Service, "mediarr"),
		Version:     firstNonEmpty(config.Version, "unknown"),
		GeneratedAt: generatedAt,
		Contents:    append([]string(nil), names...),
		Safety: []string{
			"no raw database",
			"no media files",
			"provider and media-server API keys redacted",
			"raw provider payloads excluded",
		},
	}
	if err := writeJSON(archive, redactor, "manifest.json", manifest); err != nil {
		_ = archive.Close()
		_ = file.Close()
		return Result{}, err
	}
	names = append([]string{"manifest.json"}, names...)
	for _, name := range names[1:] {
		if err := writeJSON(archive, redactor, name, files[name]); err != nil {
			_ = archive.Close()
			_ = file.Close()
			return Result{}, err
		}
	}
	if err := archive.Close(); err != nil {
		_ = file.Close()
		return Result{}, err
	}
	if err := file.Close(); err != nil {
		return Result{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Result{}, err
	}
	return Result{Path: path, SizeBytes: info.Size(), Files: names, CreatedAt: generatedAt}, nil
}

func buildEntries(config Config, generatedAt time.Time) (map[string]any, error) {
	providers, err := config.Store.ListProviderSettings()
	if err != nil {
		return nil, err
	}
	integrations, err := config.Store.ListIntegrationSettings()
	if err != nil {
		return nil, err
	}
	pathMappings, err := config.Store.ListPathMappings()
	if err != nil {
		return nil, err
	}
	jobs, err := config.Store.ListJobs(database.JobFilter{Limit: 100})
	if err != nil {
		return nil, err
	}
	recs, err := config.Store.ListRecommendations()
	if err != nil {
		return nil, err
	}
	entries := map[string]any{
		"settings/providers.json":    providers,
		"settings/integrations.json": integrations,
		"path-mappings.json":         pathMappings,
		"jobs.json":                  jobs,
		"recommendations.json":       recs,
		"safety.json": map[string]any{
			"generatedAt":            generatedAt,
			"suggestOnly":            true,
			"permanentDeleteEnabled": false,
			"mediaFilesIncluded":     false,
			"rawDatabaseIncluded":    false,
			"secretsIncluded":        false,
		},
	}
	for _, integration := range []string{"jellyfin", "plex", "emby"} {
		snapshot, err := config.Store.GetMediaServerSnapshot(integration)
		if err != nil {
			continue
		}
		filtered := make([]recommendations.Recommendation, 0, len(recs))
		for _, rec := range recs {
			if rec.ServerID == integration {
				filtered = append(filtered, rec)
			}
		}
		entries["diagnostics/"+integration+".json"] = acceptance.BuildReport(snapshot, filtered, nil, acceptance.ReportOptions{
			TargetID:                 integration,
			GeneratedAt:              generatedAt,
			RequireLocalVerification: true,
		})
	}
	return entries, nil
}

type redactor struct {
	values []string
}

func newRedactor(store *database.Store) redactor {
	values := []string{}
	if providers, err := store.ListProviderSettingSecrets(); err == nil {
		for _, provider := range providers {
			if strings.TrimSpace(provider.APIKey) != "" {
				values = append(values, provider.APIKey)
			}
		}
	}
	if integrations, err := store.ListIntegrationSettingSecrets(); err == nil {
		for _, integration := range integrations {
			if strings.TrimSpace(integration.APIKey) != "" {
				values = append(values, integration.APIKey)
			}
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return len(values[i]) > len(values[j])
	})
	return redactor{values: values}
}

func (redactor redactor) apply(input []byte) []byte {
	output := string(input)
	for _, value := range redactor.values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		output = strings.ReplaceAll(output, value, "[redacted]")
	}
	return []byte(output)
}

func writeJSON(archive *zip.Writer, redactor redactor, name string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	body = append(redactor.apply(body), '\n')
	writer, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = writer.Write(body)
	return err
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
