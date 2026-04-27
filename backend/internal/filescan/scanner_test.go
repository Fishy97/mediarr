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

func TestScannerReportsProgressWithoutFullPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Arrival.2016.1080p.mkv"), []byte("fake media bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Blade.Runner.2049.2017.mkv"), []byte("fake media bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	var events []Progress
	scanner := Scanner{
		Progress: func(progress Progress) {
			events = append(events, progress)
		},
	}
	if _, err := scanner.Scan(Library{ID: "lib_movies", Kind: "movies", Root: root}); err != nil {
		t.Fatal(err)
	}

	if len(events) < 3 {
		t.Fatalf("progress events = %#v", events)
	}
	if events[0].Phase != "discovering" || events[0].Message == "" {
		t.Fatalf("first event = %#v", events[0])
	}
	last := events[len(events)-1]
	if last.Phase != "complete" || last.Processed != 2 || last.Total != 2 {
		t.Fatalf("last event = %#v", last)
	}
	for _, event := range events {
		if event.CurrentLabel == filepath.Join(root, "Arrival.2016.1080p.mkv") {
			t.Fatalf("progress leaked full path: %#v", event)
		}
	}
}
