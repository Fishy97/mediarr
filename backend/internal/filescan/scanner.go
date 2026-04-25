package filescan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/catalog"
	"github.com/Fishy97/mediarr/backend/internal/probe"
)

type Library struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Root string `json:"root"`
}

type Item struct {
	ID          string              `json:"id"`
	LibraryID   string              `json:"libraryId"`
	Path        string              `json:"path"`
	SizeBytes   int64               `json:"sizeBytes"`
	ModifiedAt  time.Time           `json:"modifiedAt"`
	Fingerprint string              `json:"fingerprint"`
	Parsed      catalog.ParsedMedia `json:"parsed"`
	Probe       probe.Result        `json:"probe"`
	Subtitles   []string            `json:"subtitles"`
}

type Result struct {
	LibraryID    string    `json:"libraryId"`
	StartedAt    time.Time `json:"startedAt"`
	CompletedAt  time.Time `json:"completedAt"`
	FilesScanned int       `json:"filesScanned"`
	Items        []Item    `json:"items"`
}

type Scanner struct {
	Probe bool
}

var mediaExtensions = map[string]bool{
	".mkv": true,
	".mp4": true,
	".m4v": true,
	".avi": true,
	".mov": true,
	".wmv": true,
	".ts":  true,
}

var subtitleExtensions = map[string]bool{
	".srt": true,
	".ass": true,
	".ssa": true,
	".vtt": true,
}

func (scanner Scanner) Scan(library Library) (Result, error) {
	started := time.Now().UTC()
	result := Result{LibraryID: library.ID, StartedAt: started}
	var mediaPaths []string
	var subtitlePaths []string

	err := filepath.WalkDir(library.Root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch {
		case mediaExtensions[ext]:
			mediaPaths = append(mediaPaths, path)
		case subtitleExtensions[ext]:
			subtitlePaths = append(subtitlePaths, path)
		}
		return nil
	})
	if err != nil {
		return result, err
	}

	sort.Strings(mediaPaths)
	sort.Strings(subtitlePaths)

	for _, mediaPath := range mediaPaths {
		info, err := os.Stat(mediaPath)
		if err != nil {
			return result, err
		}
		parsed := catalog.ParseMediaPath(mediaPath)
		item := Item{
			ID:          fingerprint(mediaPath, info),
			LibraryID:   library.ID,
			Path:        mediaPath,
			SizeBytes:   info.Size(),
			ModifiedAt:  info.ModTime().UTC(),
			Fingerprint: fingerprint(mediaPath, info),
			Parsed:      parsed,
			Subtitles:   findSidecarSubtitles(mediaPath, subtitlePaths),
		}
		if scanner.Probe {
			item.Probe = probe.File(context.Background(), mediaPath)
		}
		result.Items = append(result.Items, item)
		result.FilesScanned++
	}

	result.CompletedAt = time.Now().UTC()
	return result, nil
}

func findSidecarSubtitles(mediaPath string, subtitles []string) []string {
	dir := filepath.Dir(mediaPath)
	base := strings.TrimSuffix(filepath.Base(mediaPath), filepath.Ext(mediaPath))
	var matches []string
	for _, subtitle := range subtitles {
		if filepath.Dir(subtitle) != dir {
			continue
		}
		subtitleBase := strings.TrimSuffix(filepath.Base(subtitle), filepath.Ext(subtitle))
		if subtitleBase == base || strings.HasPrefix(subtitleBase, base+".") {
			matches = append(matches, subtitle)
		}
	}
	return matches
}

func fingerprint(path string, info os.FileInfo) string {
	sum := sha256.Sum256([]byte(path + "|" + info.ModTime().UTC().Format(time.RFC3339Nano) + "|" + info.Name() + "|" + strconv.FormatInt(info.Size(), 10)))
	return hex.EncodeToString(sum[:16])
}
