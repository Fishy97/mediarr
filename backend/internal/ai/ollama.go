package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type OllamaClient struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

type Health struct {
	Status         string    `json:"status"`
	Model          string    `json:"model"`
	ModelAvailable bool      `json:"modelAvailable"`
	CheckedAt      time.Time `json:"checkedAt"`
}

type SuggestionInput struct {
	Title         string   `json:"title"`
	Explanation   string   `json:"explanation,omitempty"`
	AffectedPaths []string `json:"affectedPaths,omitempty"`
}

type Suggestion struct {
	Rationale  string   `json:"rationale"`
	Tags       []string `json:"tags"`
	Confidence float64  `json:"confidence"`
}

func (client OllamaClient) Health(ctx context.Context) Health {
	health := Health{
		Status:    "not_configured",
		Model:     client.model(),
		CheckedAt: time.Now().UTC(),
	}
	if strings.TrimSpace(client.BaseURL) == "" {
		return health
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL()+"/api/tags", nil)
	if err != nil {
		health.Status = "invalid_config"
		return health
	}
	res, err := client.healthClient().Do(req)
	if err != nil {
		health.Status = "unavailable"
		return health
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		health.Status = "unavailable"
		return health
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		health.Status = "unavailable"
		return health
	}
	for _, model := range payload.Models {
		if model.Name == health.Model {
			health.Status = "configured"
			health.ModelAvailable = true
			return health
		}
	}
	health.Status = "model_missing"
	return health
}

func (client OllamaClient) SuggestRationale(ctx context.Context, input SuggestionInput) (Suggestion, error) {
	if strings.TrimSpace(client.BaseURL) == "" {
		return Suggestion{}, errors.New("ollama url is not configured")
	}
	prompt := "Return only JSON with keys rationale, tags, confidence. Explain this Mediarr advisory recommendation using only the supplied facts: " + mustJSON(input)
	body := map[string]any{
		"model":  client.model(),
		"prompt": prompt,
		"stream": false,
	}
	requestBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL()+"/api/generate", bytes.NewReader(requestBody))
	if err != nil {
		return Suggestion{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.httpClient().Do(req)
	if err != nil {
		return Suggestion{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return Suggestion{}, errors.New("ollama generate failed with status " + res.Status)
	}
	var payload struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return Suggestion{}, err
	}
	var suggestion Suggestion
	if err := json.Unmarshal([]byte(payload.Response), &suggestion); err != nil {
		return Suggestion{}, err
	}
	if strings.TrimSpace(suggestion.Rationale) == "" || suggestion.Confidence < 0 || suggestion.Confidence > 1 {
		return Suggestion{}, errors.New("invalid AI suggestion schema")
	}
	if suggestion.Tags == nil {
		suggestion.Tags = []string{}
	}
	return suggestion, nil
}

func (client OllamaClient) baseURL() string {
	return strings.TrimRight(client.BaseURL, "/")
}

func (client OllamaClient) model() string {
	if strings.TrimSpace(client.Model) == "" {
		return "qwen3:0.6b"
	}
	return strings.TrimSpace(client.Model)
}

func (client OllamaClient) httpClient() *http.Client {
	if client.Client != nil {
		return client.Client
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func (client OllamaClient) healthClient() *http.Client {
	if client.Client != nil {
		return client.Client
	}
	return &http.Client{Timeout: 3 * time.Second}
}

func mustJSON(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}
