package database

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupIncludesDatabaseSettingsAndAudit(t *testing.T) {
	configDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(configDir, "mediaar.db"), []byte("db"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(configDir, "audit"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "audit", "events.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	backupPath, err := CreateBackup(configDir, filepath.Join(configDir, "backups"))
	if err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	entries := map[string]bool{}
	for _, file := range reader.File {
		entries[file.Name] = true
	}
	for _, want := range []string{"mediaar.db", "settings.json", "audit/events.jsonl"} {
		if !entries[want] {
			t.Fatalf("backup missing %s; entries: %#v", want, entries)
		}
	}
}
