package database

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func CreateBackup(configDir string, backupDir string) (string, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	backupPath := filepath.Join(backupDir, "mediarr-"+time.Now().UTC().Format("20060102T150405Z")+".zip")
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
