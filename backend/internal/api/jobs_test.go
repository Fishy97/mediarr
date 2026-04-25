package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/filescan"
)

func TestScanRouteStartsBackgroundJobWithProgress(t *testing.T) {
	root := t.TempDir()
	mediaPath := filepath.Join(root, "Arrival.2016.1080p.mkv")
	if err := os.WriteFile(mediaPath, []byte("fake media bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := NewServer(Deps{
		Store: store,
		Libraries: []filescan.Library{{
			ID:   "movies",
			Name: "Movies",
			Kind: "movies",
			Root: root,
		}},
	})

	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/scans", nil))
	if res.Code != http.StatusAccepted {
		t.Fatalf("scan status = %d, want 202: %s", res.Code, res.Body.String())
	}
	var body struct {
		Data database.Job `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Data.ID == "" || body.Data.Kind != "filesystem_scan" {
		t.Fatalf("scan job = %#v", body.Data)
	}

	job := waitForJobStatus(t, store, body.Data.ID, "completed")
	if job.Processed != 1 || job.Total != 1 {
		t.Fatalf("completed job = %#v", job)
	}

	events, err := store.ListJobEvents(job.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("expected scan progress events")
	}
	for _, event := range events {
		if event.CurrentLabel == mediaPath {
			t.Fatalf("progress leaked full path: %#v", event)
		}
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+job.ID, nil))
	if res.Code != http.StatusOK {
		t.Fatalf("job detail status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var detail struct {
		Data struct {
			database.Job
			Events []database.JobEvent `json:"events"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.Data.ID != job.ID || len(detail.Data.Events) == 0 {
		t.Fatalf("job detail = %#v", detail.Data)
	}
}

func waitForJobStatus(t *testing.T, store *database.Store, id string, status string) database.Job {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, err := store.GetJob(id)
		if err == nil && job.Status == status {
			return job
		}
		time.Sleep(20 * time.Millisecond)
	}
	job, err := store.GetJob(id)
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("job %s status = %s, want %s", id, job.Status, status)
	return database.Job{}
}
