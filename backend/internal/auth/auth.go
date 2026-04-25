package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const SessionCookieName = "mediarr_session"

type Middleware struct {
	AdminToken string
	Service    *Service
}

func (middleware Middleware) Wrap(next http.Handler) http.Handler {
	if middleware.AdminToken == "" && middleware.Service == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublic(r) {
			next.ServeHTTP(w, r)
			return
		}

		token := bearerToken(r)
		if middleware.AdminToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(middleware.AdminToken)) == 1 {
			next.ServeHTTP(w, r)
			return
		}

		if middleware.Service != nil {
			if token == "" {
				if cookie, err := r.Cookie(SessionCookieName); err == nil {
					token = cookie.Value
				}
			}
			if token != "" {
				if _, err := middleware.Service.UserForToken(token); err == nil {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

func bearerToken(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return ""
	}
	return strings.TrimSpace(value[7:])
}

func isPublic(r *http.Request) bool {
	if r.Method == http.MethodOptions {
		return true
	}
	if r.URL.Path == "/api/v1/health" || r.URL.Path == "/api/v1/setup/status" {
		return true
	}
	if r.URL.Path == "/api/v1/setup/admin" && r.Method == http.MethodPost {
		return true
	}
	if r.URL.Path == "/api/v1/auth/login" && r.Method == http.MethodPost {
		return true
	}
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		return true
	}
	return false
}
