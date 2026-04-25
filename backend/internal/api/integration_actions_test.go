package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Fishy97/mediarr/backend/internal/integrations"
)

func TestIntegrationRefreshRouteCallsConfiguredSyncTarget(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Library/Refresh" || r.Header.Get("X-Emby-Token") != "emby-key" {
			http.Error(w, "bad refresh request", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	server := NewServer(Deps{IntegrationOptions: integrations.Options{EmbyURL: target.URL, EmbyKey: "emby-key"}})
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/integrations/emby/refresh", nil))
	if res.Code != http.StatusAccepted {
		t.Fatalf("refresh status = %d, want 202: %s", res.Code, res.Body.String())
	}
	var body struct {
		Data integrations.RefreshResult `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Data.TargetID != "emby" || body.Data.Status != "requested" {
		t.Fatalf("refresh body = %#v", body.Data)
	}
}
