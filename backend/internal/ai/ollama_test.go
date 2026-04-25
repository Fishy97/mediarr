package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaClientReportsModelAvailability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("path = %s, want /api/tags", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:0.6b"}]}`))
	}))
	defer server.Close()

	client := OllamaClient{BaseURL: server.URL, Model: "qwen3:0.6b"}
	status := client.Health(context.Background())
	if status.Status != "configured" || !status.ModelAvailable {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestOllamaClientRejectsMalformedSuggestionJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":"not-json"}`))
	}))
	defer server.Close()

	client := OllamaClient{BaseURL: server.URL, Model: "qwen3:0.6b"}
	_, err := client.SuggestRationale(context.Background(), SuggestionInput{Title: "Review duplicate media"})
	if err == nil {
		t.Fatal("malformed AI response should fail validation")
	}
}
