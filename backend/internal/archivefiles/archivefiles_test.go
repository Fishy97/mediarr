package archivefiles

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListSortsGeneratedArchivesAndIgnoresOtherFiles(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "mediarr-20260426T110000.000000000Z.zip")
	newer := filepath.Join(dir, "mediarr-20260426T120000.000000000Z.zip")
	for _, path := range []string{older, newer, filepath.Join(dir, "mediarr-manual.zip"), filepath.Join(dir, "notes.txt")} {
		if err := os.WriteFile(path, []byte("archive"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	olderTime := time.Date(2026, 4, 26, 11, 0, 0, 0, time.UTC)
	newerTime := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(older, olderTime, olderTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, newerTime, newerTime); err != nil {
		t.Fatal(err)
	}

	archives, err := List(dir, "mediarr-")
	if err != nil {
		t.Fatal(err)
	}
	if len(archives) != 2 {
		t.Fatalf("len(List) = %d, want 2", len(archives))
	}
	if archives[0].Name != filepath.Base(newer) || archives[1].Name != filepath.Base(older) {
		t.Fatalf("archives order = %#v", archives)
	}
}

func TestResolveRejectsUnsafeLocatorsAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "mediarr-20260426T120000.000000000Z.zip")
	if err := os.WriteFile(archive, []byte("archive"), 0o600); err != nil {
		t.Fatal(err)
	}

	resolved, err := Resolve(dir, "mediarr-", filepath.Base(archive))
	if err != nil {
		t.Fatal(err)
	}
	if resolved != archive {
		t.Fatalf("Resolve returned %q, want %q", resolved, archive)
	}

	for _, locator := range []string{"", "../secret.zip", "/tmp/mediarr-20260426T120000.000000000Z.zip", "mediarr-manual.zip"} {
		if _, err := Resolve(dir, "mediarr-", locator); err == nil {
			t.Fatalf("Resolve(%q) succeeded; want error", locator)
		}
	}

	target := filepath.Join(t.TempDir(), "outside.zip")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "mediarr-20260426T130000.000000000Z.zip")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink creation not supported: %v", err)
	}
	if _, err := Resolve(dir, "mediarr-", filepath.Base(link)); err == nil {
		t.Fatal("Resolve succeeded for symlink; want error")
	}
	if _, err := InfoForPath(link, "mediarr-"); err == nil {
		t.Fatal("InfoForPath succeeded for symlink; want error")
	}
}
