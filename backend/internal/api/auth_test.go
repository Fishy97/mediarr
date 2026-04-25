package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/auth"
	"github.com/Fishy97/mediarr/backend/internal/database"
)

func TestSetupAndLoginRoutesCreateUsableSession(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := auth.Service{Store: store, SessionTTL: time.Hour}
	server := NewServer(Deps{Store: store, Auth: &service})
	handler := auth.Middleware{Service: &service}.Wrap(server)

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("setup status = %d, want 200", res.Code)
	}
	var status struct {
		Data struct {
			SetupRequired bool `json:"setupRequired"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if !status.Data.SetupRequired {
		t.Fatal("setup should be required")
	}

	body := bytes.NewBufferString(`{"email":"admin@example.test","password":"correct horse battery staple"}`)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/setup/admin", body))
	if res.Code != http.StatusCreated {
		t.Fatalf("admin setup status = %d, want 201: %s", res.Code, res.Body.String())
	}
	var setup struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&setup); err != nil {
		t.Fatal(err)
	}
	if setup.Data.Token == "" {
		t.Fatal("setup should return a session token")
	}

	res = httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated catalog = %d, want 401", res.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
	req.Header.Set("Authorization", "Bearer "+setup.Data.Token)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("authenticated catalog = %d, want 200", res.Code)
	}

	body = bytes.NewBufferString(`{"email":"admin@example.test","password":"correct horse battery staple"}`)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body))
	if res.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200: %s", res.Code, res.Body.String())
	}
}
