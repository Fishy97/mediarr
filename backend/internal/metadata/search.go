package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/catalog"
)

var ErrProviderNotConfigured = errors.New("metadata provider is not configured")

type Query struct {
	Kind  catalog.Kind
	Title string
	Year  int
}

type Candidate struct {
	Provider    string       `json:"provider"`
	ProviderID  string       `json:"providerId"`
	Kind        catalog.Kind `json:"kind"`
	Title       string       `json:"title"`
	Year        int          `json:"year,omitempty"`
	Overview    string       `json:"overview,omitempty"`
	Confidence  float64      `json:"confidence"`
	Attribution string       `json:"attribution"`
}

type TMDbProvider struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func (provider TMDbProvider) Search(ctx context.Context, query Query) ([]Candidate, error) {
	if strings.TrimSpace(provider.Token) == "" {
		return nil, ErrProviderNotConfigured
	}
	baseURL := strings.TrimRight(provider.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.themoviedb.org"
	}
	endpoint := "/3/search/movie"
	yearParam := "year"
	if query.Kind == catalog.KindSeries || query.Kind == catalog.KindAnime {
		endpoint = "/3/search/tv"
		yearParam = "first_air_date_year"
	}
	values := url.Values{}
	values.Set("query", query.Title)
	if query.Year > 0 {
		values.Set(yearParam, strconv.Itoa(query.Year))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+endpoint+"?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+provider.Token)
	req.Header.Set("Accept", "application/json")
	client := provider.Client
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests {
		return nil, errors.New("tmdb rate limited")
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, errors.New("tmdb search failed with status " + res.Status)
	}
	var payload tmdbSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}
	candidates := make([]Candidate, 0, len(payload.Results))
	for _, result := range payload.Results {
		title := result.Title
		date := result.ReleaseDate
		kind := catalog.KindMovie
		if title == "" {
			title = result.Name
			date = result.FirstAirDate
			kind = catalog.KindSeries
		}
		year := yearFromDate(date)
		candidates = append(candidates, Candidate{
			Provider:    "tmdb",
			ProviderID:  strconv.Itoa(result.ID),
			Kind:        kind,
			Title:       title,
			Year:        year,
			Overview:    result.Overview,
			Confidence:  candidateConfidence(query, title, year),
			Attribution: "Metadata provided by TMDb.",
		})
	}
	return candidates, nil
}

type tmdbSearchResponse struct {
	Results []struct {
		ID           int    `json:"id"`
		Title        string `json:"title"`
		Name         string `json:"name"`
		ReleaseDate  string `json:"release_date"`
		FirstAirDate string `json:"first_air_date"`
		Overview     string `json:"overview"`
	} `json:"results"`
}

func yearFromDate(value string) int {
	if len(value) < 4 {
		return 0
	}
	year, _ := strconv.Atoi(value[:4])
	return year
}

func candidateConfidence(query Query, title string, year int) float64 {
	score := 0.65
	if strings.EqualFold(strings.TrimSpace(query.Title), strings.TrimSpace(title)) {
		score += 0.25
	}
	if query.Year > 0 && query.Year == year {
		score += 0.10
	}
	if score > 0.99 {
		return 0.99
	}
	return score
}
