package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/acceptance"
)

func main() {
	log.SetFlags(0)
	mappings, err := acceptance.ParsePathMappings(os.Getenv("MEDIARR_ACCEPTANCE_PATH_MAPS"), "jellyfin")
	if err != nil {
		log.Fatalf("parse path mappings: %v", err)
	}
	timeout, err := parseDurationEnv("MEDIARR_ACCEPTANCE_TIMEOUT", 6*time.Hour)
	if err != nil {
		log.Fatalf("parse timeout: %v", err)
	}
	reportDir := os.Getenv("MEDIARR_ACCEPTANCE_REPORT_DIR")
	storeDir := os.Getenv("MEDIARR_ACCEPTANCE_STORE_DIR")
	if storeDir == "" && reportDir != "" {
		storeDir = filepath.Join(reportDir, "scratch-db-"+time.Now().UTC().Format("20060102-150405"))
	}
	report, err := acceptance.RunJellyfin(context.Background(), acceptance.JellyfinConfig{
		BaseURL:                  os.Getenv("MEDIARR_ACCEPTANCE_JELLYFIN_URL"),
		APIKey:                   os.Getenv("MEDIARR_ACCEPTANCE_JELLYFIN_API_KEY"),
		PathMappings:             mappings,
		StoreDir:                 storeDir,
		RedactTitles:             parseBoolEnv("MEDIARR_ACCEPTANCE_REDACT_TITLES", false),
		RequireLocalVerification: parseBoolEnv("MEDIARR_ACCEPTANCE_REQUIRE_LOCAL_VERIFY", false),
		Timeout:                  timeout,
	})
	if err != nil {
		log.Fatalf("run jellyfin acceptance: %v", err)
	}
	files, err := acceptance.WriteReportFiles(report, reportDir)
	if err != nil {
		log.Fatalf("write acceptance reports: %v", err)
	}
	fmt.Printf("Mediarr live Jellyfin acceptance completed\n")
	fmt.Printf("JSON report: %s\n", files.JSONPath)
	fmt.Printf("Markdown report: %s\n", files.MarkdownPath)
	fmt.Printf("Imported: %d movies, %d series, %d episodes, %d files\n", report.Summary.Movies, report.Summary.Series, report.Summary.Episodes, report.Summary.Files)
	fmt.Printf("Recommendations: %d (%s reclaimable, server reported)\n", report.Summary.Recommendations, formatBytes(report.Summary.RecommendationBytes))
	if len(report.Warnings) > 0 {
		fmt.Printf("Warnings: %d\n", len(report.Warnings))
	}
}

func parseBoolEnv(name string, fallback bool) bool {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func parseDurationEnv(name string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(raw)
	if err == nil {
		return duration, nil
	}
	minutes, parseErr := strconv.Atoi(raw)
	if parseErr != nil {
		return 0, err
	}
	return time.Duration(minutes) * time.Minute, nil
}

func formatBytes(bytes int64) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	for _, suffix := range []string{"KB", "MB", "GB", "TB", "PB"} {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.2f %s", value, suffix)
		}
	}
	return fmt.Sprintf("%.2f PB", value)
}
