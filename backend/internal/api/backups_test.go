package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/database"
)

func TestBackupRestoreDryRunRouteInspectsArchive(t *testing.T) {
	configDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(configDir, "mediarr.db"), []byte("db"), 0o600); err != nil {
		t.Fatal(err)
	}
	backupPath, err := database.CreateBackup(configDir, filepath.Join(configDir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(Deps{ConfigDir: configDir})
	body := bytes.NewBufferString(`{"path":"` + backupPath + `","dryRun":true}`)

	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/backups/restore", body))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", res.Code, res.Body.String())
	}
	if !bytes.Contains(res.Body.Bytes(), []byte("mediarr.db")) {
		t.Fatalf("restore dry run response missing entries: %s", res.Body.String())
	}
}
