package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/database"
)

func TestAppearanceSettingsRoutesPersistAndValidate(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	server := NewServer(Deps{Store: store})

	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/appearance", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("appearance status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var defaults struct {
		Data database.AppearanceSettings `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&defaults); err != nil {
		t.Fatal(err)
	}
	if defaults.Data.Theme != "system" || defaults.Data.CustomCSS != "" {
		t.Fatalf("default appearance = %#v", defaults.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPut, "/api/v1/appearance", jsonBody(t, database.AppearanceSettingsInput{
		Theme:     "dark",
		CustomCSS: ".panel { border-radius: 6px; }",
	})))
	if res.Code != http.StatusOK {
		t.Fatalf("appearance update status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var updated struct {
		Data database.AppearanceSettings `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Data.Theme != "dark" || updated.Data.CustomCSS != ".panel { border-radius: 6px; }" {
		t.Fatalf("updated appearance = %#v", updated.Data)
	}

	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPut, "/api/v1/appearance", jsonBody(t, database.AppearanceSettingsInput{
		Theme:     "light",
		CustomCSS: "body { background-image: url('https://example.test/track'); }",
	})))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("unsafe custom css status = %d, want 400", res.Code)
	}
}
