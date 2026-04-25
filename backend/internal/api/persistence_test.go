package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/media-steward/media-library-manager/backend/internal/catalog"
	"github.com/media-steward/media-library-manager/backend/internal/database"
	"github.com/media-steward/media-library-manager/backend/internal/filescan"
)

func TestServerReturnsPersistedCatalog(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.SaveScan(filescan.Result{
		LibraryID:   "movies",
		StartedAt:   time.Now().UTC(),
		CompletedAt: time.Now().UTC(),
		Items: []filescan.Item{
			{
				ID:          "file_1",
				LibraryID:   "movies",
				Path:        "/media/movies/Arrival.2016.1080p.mkv",
				SizeBytes:   8_000_000_000,
				Fingerprint: "fingerprint_1",
				ModifiedAt:  time.Now().UTC(),
				Parsed: catalog.ParsedMedia{
					Kind:         catalog.KindMovie,
					Title:        "Arrival",
					Quality:      "1080p",
					CanonicalKey: "movie:arrival:2016",
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(Deps{Store: store})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}

	var body struct {
		Data []database.CatalogItem `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 1 || body.Data[0].Title != "Arrival" {
		t.Fatalf("unexpected catalog response: %#v", body.Data)
	}
}
