package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/catalog"
)

func TestTMDbProviderSearchesMovieCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/3/search/movie" {
			t.Fatalf("path = %s, want /3/search/movie", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "Arrival" || r.URL.Query().Get("year") != "2016" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":329865,"title":"Arrival","release_date":"2016-11-10","overview":"A linguist works with the military."}]}`))
	}))
	defer server.Close()

	provider := TMDbProvider{BaseURL: server.URL, Token: "tmdb-token"}
	candidates, err := provider.Search(context.Background(), Query{Kind: catalog.KindMovie, Title: "Arrival", Year: 2016})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(candidates))
	}
	if candidates[0].ProviderID != "329865" || candidates[0].Confidence < 0.95 {
		t.Fatalf("unexpected candidate: %#v", candidates[0])
	}
}

func TestSearchProviderRequiresConfiguration(t *testing.T) {
	provider := TMDbProvider{}
	_, err := provider.Search(context.Background(), Query{Kind: catalog.KindMovie, Title: "Arrival", Year: 2016})
	if err == nil {
		t.Fatal("unconfigured provider should return an error")
	}
}
