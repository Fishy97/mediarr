package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestBackupRoutesListDownloadAndRestoreSelectedArchive(t *testing.T) {
	configDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(configDir, "mediarr.db"), []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}

	server := NewServer(Deps{ConfigDir: configDir})
	createRes := httptest.NewRecorder()
	server.ServeHTTP(createRes, httptest.NewRequest(http.MethodPost, "/api/v1/backups", nil))
	if createRes.Code != http.StatusCreated {
		t.Fatalf("backup create status = %d, want 201: %s", createRes.Code, createRes.Body.String())
	}
	var createBody struct {
		Data struct {
			Name      string `json:"name"`
			Path      string `json:"path"`
			SizeBytes int64  `json:"sizeBytes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createRes.Body).Decode(&createBody); err != nil {
		t.Fatal(err)
	}
	if createBody.Data.Name == "" || createBody.Data.Path == "" || createBody.Data.SizeBytes <= 0 {
		t.Fatalf("backup create body = %#v", createBody.Data)
	}

	listRes := httptest.NewRecorder()
	server.ServeHTTP(listRes, httptest.NewRequest(http.MethodGet, "/api/v1/backups", nil))
	if listRes.Code != http.StatusOK {
		t.Fatalf("backup list status = %d, want 200: %s", listRes.Code, listRes.Body.String())
	}
	var listBody struct {
		Data []struct {
			Name      string `json:"name"`
			Path      string `json:"path"`
			SizeBytes int64  `json:"sizeBytes"`
			CreatedAt string `json:"createdAt"`
		} `json:"data"`
	}
	if err := json.NewDecoder(listRes.Body).Decode(&listBody); err != nil {
		t.Fatal(err)
	}
	if len(listBody.Data) != 1 || listBody.Data[0].Name != createBody.Data.Name || listBody.Data[0].CreatedAt == "" {
		t.Fatalf("backup list body = %#v", listBody.Data)
	}

	downloadRes := httptest.NewRecorder()
	server.ServeHTTP(downloadRes, httptest.NewRequest(http.MethodGet, "/api/v1/backups/"+createBody.Data.Name, nil))
	if downloadRes.Code != http.StatusOK {
		t.Fatalf("backup download status = %d, want 200: %s", downloadRes.Code, downloadRes.Body.String())
	}
	if got := downloadRes.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("download content-type = %q, want application/zip", got)
	}
	if !strings.Contains(downloadRes.Header().Get("Content-Disposition"), createBody.Data.Name) {
		t.Fatalf("download content-disposition = %q", downloadRes.Header().Get("Content-Disposition"))
	}
	if !bytes.Contains(downloadRes.Body.Bytes(), []byte("mediarr.db")) {
		t.Fatalf("downloaded archive does not look like a backup")
	}

	dryRun := bytes.NewBufferString(`{"name":"` + createBody.Data.Name + `","dryRun":true}`)
	dryRunRes := httptest.NewRecorder()
	server.ServeHTTP(dryRunRes, httptest.NewRequest(http.MethodPost, "/api/v1/backups/restore", dryRun))
	if dryRunRes.Code != http.StatusOK {
		t.Fatalf("backup selected dry run status = %d, want 200: %s", dryRunRes.Code, dryRunRes.Body.String())
	}
	if !bytes.Contains(dryRunRes.Body.Bytes(), []byte("mediarr.db")) {
		t.Fatalf("selected dry run missing entries: %s", dryRunRes.Body.String())
	}

	if err := os.WriteFile(filepath.Join(configDir, "mediarr.db"), []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}
	restore := bytes.NewBufferString(`{"name":"` + createBody.Data.Name + `","dryRun":false,"confirmRestore":true}`)
	restoreRes := httptest.NewRecorder()
	server.ServeHTTP(restoreRes, httptest.NewRequest(http.MethodPost, "/api/v1/backups/restore", restore))
	if restoreRes.Code != http.StatusOK {
		t.Fatalf("backup selected restore status = %d, want 200: %s", restoreRes.Code, restoreRes.Body.String())
	}
	restored, err := os.ReadFile(filepath.Join(configDir, "mediarr.db"))
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != "before" {
		t.Fatalf("database after selected restore = %q, want before", string(restored))
	}

	unsafeDownload := httptest.NewRecorder()
	server.ServeHTTP(unsafeDownload, httptest.NewRequest(http.MethodGet, "/api/v1/backups/..%2Fmediarr.db", nil))
	if unsafeDownload.Code != http.StatusBadRequest {
		t.Fatalf("unsafe backup download status = %d, want 400", unsafeDownload.Code)
	}
	unsafeRestore := bytes.NewBufferString(`{"path":"/tmp/not-a-mediarr-backup.zip","dryRun":true}`)
	unsafeRestoreRes := httptest.NewRecorder()
	server.ServeHTTP(unsafeRestoreRes, httptest.NewRequest(http.MethodPost, "/api/v1/backups/restore", unsafeRestore))
	if unsafeRestoreRes.Code != http.StatusBadRequest {
		t.Fatalf("unsafe backup restore status = %d, want 400", unsafeRestoreRes.Code)
	}
}

func TestSupportBundleRouteCreatesRedactedArchive(t *testing.T) {
	configDir := t.TempDir()
	store, err := database.Open(configDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if _, err := store.UpsertIntegrationSetting(database.IntegrationSettingInput{
		Integration: "jellyfin",
		BaseURL:     "http://jellyfin.local",
		APIKey:      "jellyfin-secret-token",
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(Deps{ConfigDir: configDir, Store: store})
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/support/bundles", nil))
	if res.Code != http.StatusCreated {
		t.Fatalf("support bundle status = %d, want 201: %s", res.Code, res.Body.String())
	}

	var body struct {
		Data struct {
			Path      string   `json:"path"`
			SizeBytes int64    `json:"sizeBytes"`
			Files     []string `json:"files"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Path == "" || body.Data.SizeBytes <= 0 {
		t.Fatalf("support bundle response = %#v", body.Data)
	}
	if !strings.HasPrefix(body.Data.Path, filepath.Join(configDir, "support")) {
		t.Fatalf("support bundle path %q is outside /config/support", body.Data.Path)
	}

	content := readZipArchive(t, body.Data.Path)
	if strings.Contains(content, "jellyfin-secret-token") {
		t.Fatalf("support bundle leaked integration secret: %s", content)
	}
	for _, required := range []string{"manifest.json", "settings/integrations.json", "safety.json"} {
		if !strings.Contains(content, required) {
			t.Fatalf("support bundle content missing %s: %s", required, content)
		}
	}
}

func TestSupportBundleRouteListsAndDownloadsArchives(t *testing.T) {
	configDir := t.TempDir()
	store, err := database.Open(configDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := NewServer(Deps{ConfigDir: configDir, Store: store})
	createRes := httptest.NewRecorder()
	server.ServeHTTP(createRes, httptest.NewRequest(http.MethodPost, "/api/v1/support/bundles", nil))
	if createRes.Code != http.StatusCreated {
		t.Fatalf("support bundle create status = %d, want 201: %s", createRes.Code, createRes.Body.String())
	}
	var createBody struct {
		Data struct {
			Path string `json:"path"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createRes.Body).Decode(&createBody); err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(createBody.Data.Path)

	listRes := httptest.NewRecorder()
	server.ServeHTTP(listRes, httptest.NewRequest(http.MethodGet, "/api/v1/support/bundles", nil))
	if listRes.Code != http.StatusOK {
		t.Fatalf("support bundle list status = %d, want 200: %s", listRes.Code, listRes.Body.String())
	}
	var listBody struct {
		Data []struct {
			Name      string `json:"name"`
			Path      string `json:"path"`
			SizeBytes int64  `json:"sizeBytes"`
			CreatedAt string `json:"createdAt"`
		} `json:"data"`
	}
	if err := json.NewDecoder(listRes.Body).Decode(&listBody); err != nil {
		t.Fatal(err)
	}
	if len(listBody.Data) != 1 || listBody.Data[0].Name != name || listBody.Data[0].SizeBytes <= 0 || listBody.Data[0].CreatedAt == "" {
		t.Fatalf("support bundle list body = %#v", listBody.Data)
	}

	downloadRes := httptest.NewRecorder()
	server.ServeHTTP(downloadRes, httptest.NewRequest(http.MethodGet, "/api/v1/support/bundles/"+name, nil))
	if downloadRes.Code != http.StatusOK {
		t.Fatalf("support bundle download status = %d, want 200: %s", downloadRes.Code, downloadRes.Body.String())
	}
	if got := downloadRes.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("download content-type = %q, want application/zip", got)
	}
	if !strings.Contains(downloadRes.Header().Get("Content-Disposition"), name) {
		t.Fatalf("download content-disposition = %q", downloadRes.Header().Get("Content-Disposition"))
	}
	if !bytes.Contains(downloadRes.Body.Bytes(), []byte("manifest.json")) {
		t.Fatalf("downloaded archive does not look like a support bundle")
	}

	unsafeRes := httptest.NewRecorder()
	server.ServeHTTP(unsafeRes, httptest.NewRequest(http.MethodGet, "/api/v1/support/bundles/..%2Fmediarr.db", nil))
	if unsafeRes.Code != http.StatusBadRequest {
		t.Fatalf("unsafe support bundle download status = %d, want 400", unsafeRes.Code)
	}
}

func readZipArchive(t *testing.T, path string) string {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	var builder strings.Builder
	for _, file := range reader.File {
		builder.WriteString(file.Name)
		builder.WriteByte('\n')
		handle, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(handle)
		_ = handle.Close()
		if err != nil {
			t.Fatal(err)
		}
		builder.Write(body)
		builder.WriteByte('\n')
	}
	return builder.String()
}
