package filescan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScannerIndexesMediaFilesAndSidecarSubtitles(t *testing.T) {
	root := t.TempDir()
	movieDir := filepath.Join(root, "movies", "Arrival (2016)")
	if err := os.MkdirAll(movieDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mediaPath := filepath.Join(movieDir, "Arrival.2016.1080p.mkv")
	subtitlePath := filepath.Join(movieDir, "Arrival.2016.1080p.en.srt")
	if err := os.WriteFile(mediaPath, []byte("fake media bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subtitlePath, []byte("subtitle"), 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := Scanner{}
	result, err := scanner.Scan(Library{ID: "lib_movies", Kind: "movies", Root: filepath.Join(root, "movies")})
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesScanned != 1 {
		t.Fatalf("files scanned = %d, want 1", result.FilesScanned)
	}
	if len(result.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(result.Items))
	}
	if result.Items[0].Path != mediaPath {
		t.Fatalf("path = %q, want %q", result.Items[0].Path, mediaPath)
	}
	if len(result.Items[0].Subtitles) != 1 || result.Items[0].Subtitles[0] != subtitlePath {
		t.Fatalf("subtitles = %#v, want %q", result.Items[0].Subtitles, subtitlePath)
	}
	if result.Items[0].Fingerprint == "" {
		t.Fatal("fingerprint should be populated")
	}
}

func TestScannerSkipsUnsupportedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("ignore me"), 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := Scanner{}
	result, err := scanner.Scan(Library{ID: "lib", Kind: "movies", Root: root})
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesScanned != 0 {
		t.Fatalf("files scanned = %d, want 0", result.FilesScanned)
	}
}
