package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/ai"
)

func TestAIStatusRouteReportsConfiguredClient(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:0.6b"}]}`))
	}))
	defer ollama.Close()

	client := ai.OllamaClient{BaseURL: ollama.URL, Model: "qwen3:0.6b"}
	server := NewServer(Deps{AI: &client})

	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/ai/status", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if body := res.Body.String(); !containsAll(body, "configured", "qwen3:0.6b") {
		t.Fatalf("unexpected body: %s", body)
	}
}

func containsAll(value string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}
