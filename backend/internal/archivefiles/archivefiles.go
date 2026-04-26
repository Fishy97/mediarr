package archivefiles

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const TimestampFormat = "20060102T150405.000000000Z"

type Info struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
}

func List(dir string, prefix string) ([]Info, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, errors.New("archive directory is required")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Info{}, nil
		}
		return nil, err
	}
	archives := []Info{}
	for _, entry := range entries {
		if !ValidName(prefix, entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := regularArchiveInfo(path, prefix)
		if err != nil {
			if errors.Is(err, errInvalidArchive) {
				continue
			}
			return nil, err
		}
		archives = append(archives, Info{
			Name:      entry.Name(),
			Path:      path,
			SizeBytes: info.Size(),
			CreatedAt: info.ModTime().UTC(),
		})
	}
	sortInfos(archives)
	return archives, nil
}

func Resolve(dir string, prefix string, locator string) (string, error) {
	dir = strings.TrimSpace(dir)
	locator = strings.TrimSpace(locator)
	if dir == "" {
		return "", errors.New("archive directory is required")
	}
	if locator == "" {
		return "", errors.New("archive path or name is required")
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	var candidate string
	if filepath.IsAbs(locator) {
		candidate, err = filepath.Abs(locator)
	} else {
		if filepath.Base(locator) != locator {
			return "", errors.New("invalid archive name")
		}
		candidate, err = filepath.Abs(filepath.Join(root, locator))
	}
	if err != nil {
		return "", err
	}
	if filepath.Dir(candidate) != root {
		return "", errors.New("archive path escapes archive directory")
	}
	if !ValidName(prefix, filepath.Base(candidate)) {
		return "", errors.New("invalid archive name")
	}
	if _, err := regularArchiveInfo(candidate, prefix); err != nil {
		return "", err
	}
	return candidate, nil
}

func InfoForPath(path string, prefix string) (Info, error) {
	info, err := regularArchiveInfo(path, prefix)
	if err != nil {
		return Info{}, err
	}
	return Info{
		Name:      filepath.Base(path),
		Path:      path,
		SizeBytes: info.Size(),
		CreatedAt: info.ModTime().UTC(),
	}, nil
}

func Name(prefix string, createdAt time.Time) string {
	return strings.TrimSpace(prefix) + createdAt.UTC().Format(TimestampFormat) + ".zip"
}

func ValidName(prefix string, name string) bool {
	if name == "" || filepath.Base(name) != name || strings.Contains(name, "..") {
		return false
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".zip") {
		return false
	}
	timestamp := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".zip")
	parsed, err := time.Parse(TimestampFormat, timestamp)
	return err == nil && parsed.UTC().Format(TimestampFormat) == timestamp
}

var errInvalidArchive = errors.New("invalid archive file")

func regularArchiveInfo(path string, prefix string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !ValidName(prefix, filepath.Base(path)) || !info.Mode().IsRegular() {
		return nil, errInvalidArchive
	}
	return info, nil
}

func sortInfos(archives []Info) {
	sort.Slice(archives, func(i, j int) bool {
		if archives[i].CreatedAt.Equal(archives[j].CreatedAt) {
			return archives[i].Name > archives[j].Name
		}
		return archives[i].CreatedAt.After(archives[j].CreatedAt)
	})
}
