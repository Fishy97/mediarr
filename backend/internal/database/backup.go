package database

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type RestoreResult struct {
	PreRestoreBackup string   `json:"preRestoreBackup"`
	Restored         []string `json:"restored"`
}

type BackupInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
}

func CreateBackup(configDir string, backupDir string) (string, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	backupPath := filepath.Join(backupDir, "mediarr-"+time.Now().UTC().Format("20060102T150405.000000000Z")+".zip")
	file, err := os.Create(backupPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	defer archive.Close()

	include := map[string]bool{
		"mediarr.db":    true,
		"settings.json": true,
		"audit":         true,
		"providers":     true,
		"artwork":       true,
	}

	err = filepath.WalkDir(configDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == configDir {
			return nil
		}
		rel, err := filepath.Rel(configDir, path)
		if err != nil {
			return err
		}
		top := strings.Split(filepath.ToSlash(rel), "/")[0]
		if top == "backups" || !include[top] {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		return addFile(archive, path, filepath.ToSlash(rel))
	})
	if err != nil {
		return "", err
	}
	return backupPath, nil
}

func ListBackups(backupDir string) ([]BackupInfo, error) {
	backupDir = strings.TrimSpace(backupDir)
	if backupDir == "" {
		return nil, errors.New("backup directory is required")
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []BackupInfo{}, nil
		}
		return nil, err
	}
	backups := []BackupInfo{}
	for _, entry := range entries {
		if entry.IsDir() || !validBackupName(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		backups = append(backups, BackupInfo{
			Name:      entry.Name(),
			Path:      filepath.Join(backupDir, entry.Name()),
			SizeBytes: info.Size(),
			CreatedAt: info.ModTime().UTC(),
		})
	}
	sortBackupInfos(backups)
	return backups, nil
}

func ResolveBackupPath(backupDir string, locator string) (string, error) {
	backupDir = strings.TrimSpace(backupDir)
	locator = strings.TrimSpace(locator)
	if backupDir == "" {
		return "", errors.New("backup directory is required")
	}
	if locator == "" {
		return "", errors.New("backup path or name is required")
	}
	root, err := filepath.Abs(backupDir)
	if err != nil {
		return "", err
	}
	var candidate string
	if filepath.IsAbs(locator) {
		candidate, err = filepath.Abs(locator)
	} else {
		if filepath.Base(locator) != locator {
			return "", errors.New("invalid backup name")
		}
		candidate, err = filepath.Abs(filepath.Join(root, locator))
	}
	if err != nil {
		return "", err
	}
	if filepath.Dir(candidate) != root {
		return "", errors.New("backup path escapes backup directory")
	}
	if !validBackupName(filepath.Base(candidate)) {
		return "", errors.New("invalid backup name")
	}
	if _, err := os.Stat(candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func BackupInfoForPath(path string) (BackupInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return BackupInfo{}, err
	}
	if info.IsDir() || !validBackupName(filepath.Base(path)) {
		return BackupInfo{}, errors.New("invalid backup path")
	}
	return BackupInfo{
		Name:      filepath.Base(path),
		Path:      path,
		SizeBytes: info.Size(),
		CreatedAt: info.ModTime().UTC(),
	}, nil
}

func InspectBackup(backupPath string) ([]string, error) {
	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	entries := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		if err := validateArchiveName(file.Name); err != nil {
			return nil, err
		}
		entries = append(entries, file.Name)
	}
	return entries, nil
}

func RestoreBackup(configDir string, backupPath string, backupDir string) (RestoreResult, error) {
	entries, err := InspectBackup(backupPath)
	if err != nil {
		return RestoreResult{}, err
	}
	preRestore, err := CreateBackup(configDir, backupDir)
	if err != nil {
		return RestoreResult{}, err
	}
	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		return RestoreResult{}, err
	}
	defer reader.Close()
	for _, file := range reader.File {
		if err := restoreFile(configDir, file); err != nil {
			return RestoreResult{}, err
		}
	}
	return RestoreResult{PreRestoreBackup: preRestore, Restored: entries}, nil
}

func restoreFile(configDir string, file *zip.File) error {
	if err := validateArchiveName(file.Name); err != nil {
		return err
	}
	target := filepath.Join(configDir, filepath.FromSlash(file.Name))
	cleanConfig, err := filepath.Abs(configDir)
	if err != nil {
		return err
	}
	cleanTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if cleanTarget != cleanConfig && !strings.HasPrefix(cleanTarget, cleanConfig+string(os.PathSeparator)) {
		return errors.New("backup entry escapes config directory")
	}
	if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
		return err
	}
	input, err := file.Open()
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer output.Close()
	_, err = io.Copy(output, input)
	return err
}

func validateArchiveName(name string) error {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return errors.New("unsafe backup entry path")
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || clean == ".." {
		return errors.New("unsafe backup entry path")
	}
	return nil
}

func addFile(archive *zip.Writer, source string, name string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, input)
	return err
}

func validBackupName(name string) bool {
	return name != "" &&
		filepath.Base(name) == name &&
		!strings.Contains(name, "..") &&
		strings.HasPrefix(name, "mediarr-") &&
		strings.HasSuffix(name, ".zip")
}

func sortBackupInfos(backups []BackupInfo) {
	sort.Slice(backups, func(i, j int) bool {
		if backups[i].CreatedAt.Equal(backups[j].CreatedAt) {
			return backups[i].Name > backups[j].Name
		}
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})
}
