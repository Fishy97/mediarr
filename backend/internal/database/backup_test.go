package database

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackupIncludesDatabaseSettingsAndAudit(t *testing.T) {
	configDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(configDir, "mediarr.db"), []byte("db"), 0o600); err != nil {
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
	for _, want := range []string{"mediarr.db", "settings.json", "audit/events.jsonl"} {
		if !entries[want] {
			t.Fatalf("backup missing %s; entries: %#v", want, entries)
		}
	}
}

func TestInspectAndRestoreBackup(t *testing.T) {
	configDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(configDir, "mediarr.db"), []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	backupPath, err := CreateBackup(configDir, filepath.Join(configDir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "mediarr.db"), []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}

	entries, err := InspectBackup(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 || entries[0] != "mediarr.db" {
		t.Fatalf("entries = %#v", entries)
	}

	result, err := RestoreBackup(configDir, backupPath, filepath.Join(configDir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	if result.PreRestoreBackup == "" || len(result.Restored) == 0 {
		t.Fatalf("unexpected restore result: %#v", result)
	}
	data, err := os.ReadFile(filepath.Join(configDir, "mediarr.db"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "before" {
		t.Fatalf("database after restore = %q, want before", string(data))
	}
}

func TestRestoreRejectsUnsafeArchivePaths(t *testing.T) {
	configDir := t.TempDir()
	backupPath := filepath.Join(configDir, "unsafe.zip")
	file, err := os.Create(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("../outside.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("bad")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := RestoreBackup(configDir, backupPath, filepath.Join(configDir, "backups")); err == nil {
		t.Fatal("unsafe archive path should be rejected")
	}
}

func TestListBackupsSortsArchivesAndResolveRejectsUnsafeNames(t *testing.T) {
	backupDir := t.TempDir()
	olderName := "mediarr-20260426T110000.000000000Z.zip"
	newerName := "mediarr-20260426T120000.000000000Z.zip"
	olderPath := filepath.Join(backupDir, olderName)
	newerPath := filepath.Join(backupDir, newerName)
	if err := os.WriteFile(olderPath, []byte("older"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newerPath, []byte("newer"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "notes.txt"), []byte("ignore"), 0o600); err != nil {
		t.Fatal(err)
	}
	olderTime := time.Date(2026, 4, 26, 11, 0, 0, 0, time.UTC)
	newerTime := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(olderPath, olderTime, olderTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newerPath, newerTime, newerTime); err != nil {
		t.Fatal(err)
	}

	backups, err := ListBackups(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 2 {
		t.Fatalf("backup count = %d, want 2: %#v", len(backups), backups)
	}
	if backups[0].Name != newerName || backups[1].Name != olderName {
		t.Fatalf("backup order = %#v", backups)
	}
	if backups[0].Path != newerPath || backups[0].SizeBytes != int64(len("newer")) || !backups[0].CreatedAt.Equal(newerTime) {
		t.Fatalf("newest backup metadata = %#v", backups[0])
	}

	resolved, err := ResolveBackupPath(backupDir, newerName)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != newerPath {
		t.Fatalf("resolved path = %q, want %q", resolved, newerPath)
	}
	resolved, err = ResolveBackupPath(backupDir, newerPath)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != newerPath {
		t.Fatalf("resolved absolute path = %q, want %q", resolved, newerPath)
	}
	for _, name := range []string{"", "../secret.zip", "/tmp/secret.zip", "mediarr-20260426.txt", "not-mediarr.zip"} {
		if _, err := ResolveBackupPath(backupDir, name); err == nil {
			t.Fatalf("ResolveBackupPath(%q) succeeded; want error", name)
		}
	}
}
