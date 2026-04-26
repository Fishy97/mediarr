package database

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/archivefiles"
)

const backupArchivePrefix = "mediarr-"

type RestoreResult struct {
	PreRestoreBackup string   `json:"preRestoreBackup"`
	Restored         []string `json:"restored"`
}

type BackupInfo = archivefiles.Info

func CreateBackup(configDir string, backupDir string) (string, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	backupPath := filepath.Join(backupDir, archivefiles.Name(backupArchivePrefix, time.Now()))
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
	return archivefiles.List(backupDir, backupArchivePrefix)
}

func ResolveBackupPath(backupDir string, locator string) (string, error) {
	return archivefiles.Resolve(backupDir, backupArchivePrefix, locator)
}

func BackupInfoForPath(path string) (BackupInfo, error) {
	return archivefiles.InfoForPath(path, backupArchivePrefix)
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
