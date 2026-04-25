package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/catalog"
	"github.com/Fishy97/mediarr/backend/internal/database"
	"github.com/Fishy97/mediarr/backend/internal/filescan"
)

func TestCatalogCorrectionRoutesApplyAndClearUserOverrides(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now().UTC()
	if err := store.SaveScan(filescan.Result{
		LibraryID:   "movies",
		CompletedAt: now,
		Items: []filescan.Item{
			{
				ID:          "file_correct",
				LibraryID:   "movies",
				Path:        "/media/movies/Arrivall.2016.mkv",
				Fingerprint: "fingerprint_correct",
				ModifiedAt:  now,
				Parsed: catalog.ParsedMedia{
					Kind:         catalog.KindMovie,
					Title:        "Arrivall",
					Year:         2016,
					CanonicalKey: "movie:arrivall:2016",
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(Deps{Store: store})
	body := bytes.NewBufferString(`{"title":"Arrival","kind":"movie","year":2016,"provider":"tmdb","providerId":"329865","confidence":0.97}`)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPut, "/api/v1/catalog/file_correct/correction", body))
	if res.Code != http.StatusOK {
		t.Fatalf("catalog correction status = %d, want 200: %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("catalog status = %d, want 200", res.Code)
	}
	var catalogBody struct {
		Data []database.CatalogItem `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&catalogBody); err != nil {
		t.Fatal(err)
	}
	if len(catalogBody.Data) != 1 || catalogBody.Data[0].Title != "Arrival" || !catalogBody.Data[0].MetadataCorrected {
		t.Fatalf("catalog correction not applied: %#v", catalogBody.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodDelete, "/api/v1/catalog/file_correct/correction", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("catalog correction delete status = %d, want 200: %s", res.Code, res.Body.String())
	}

	items, err := store.ListCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Title != "Arrivall" || items[0].MetadataCorrected {
		t.Fatalf("catalog correction should be cleared: %#v", items[0])
	}
}
