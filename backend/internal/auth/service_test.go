package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/database"
)

func TestServiceCreatesAdminAndAuthenticatesSession(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := Service{Store: store, SessionTTL: time.Hour}
	required, err := service.SetupRequired()
	if err != nil {
		t.Fatal(err)
	}
	if !required {
		t.Fatal("setup should be required before admin creation")
	}

	user, session, err := service.CreateAdmin("admin@example.test", "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "admin@example.test" || session.Token == "" {
		t.Fatalf("unexpected setup result: user=%#v session=%#v", user, session)
	}

	stored, err := store.UserByEmail("admin@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if stored.PasswordHash == "correct horse battery staple" {
		t.Fatal("password must not be stored in plaintext")
	}

	_, _, err = service.Login("admin@example.test", "wrong password")
	if err == nil {
		t.Fatal("wrong password should not authenticate")
	}

	loginUser, loginSession, err := service.Login("admin@example.test", "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if loginUser.ID != user.ID || loginSession.Token == "" {
		t.Fatalf("unexpected login result: user=%#v session=%#v", loginUser, loginSession)
	}

	validated, err := service.UserForToken(loginSession.Token)
	if err != nil {
		t.Fatal(err)
	}
	if validated.ID != user.ID {
		t.Fatalf("validated user = %q, want %q", validated.ID, user.ID)
	}
}

func TestMiddlewareProtectsAPIAndAllowsBearerFallback(t *testing.T) {
	store, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := Service{Store: store, SessionTTL: time.Hour}
	_, session, err := service.CreateAdmin("admin@example.test", "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := Middleware{AdminToken: "automation-token", Service: &service}.Wrap(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
	req.Header.Set("Authorization", "Bearer "+session.Token)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("session status = %d, want 204", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
	req.Header.Set("Authorization", "Bearer automation-token")
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("automation token status = %d, want 204", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("health status = %d, want 204", res.Code)
	}
}
