package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/database"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrSetupComplete       = errors.New("setup already completed")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrPasswordTooShort    = errors.New("password must be at least 12 characters")
	ErrAuthenticationStore = errors.New("authentication store is not configured")
)

type Service struct {
	Store      *database.Store
	SessionTTL time.Duration
}

type Session struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (service Service) SetupRequired() (bool, error) {
	if service.Store == nil {
		return false, ErrAuthenticationStore
	}
	exists, err := service.Store.AdminUserExists()
	if err != nil {
		return false, err
	}
	return !exists, nil
}

func (service Service) CreateAdmin(email string, password string) (database.User, Session, error) {
	if service.Store == nil {
		return database.User{}, Session{}, ErrAuthenticationStore
	}
	_, _ = service.Store.DeleteExpiredSessions(time.Now().UTC())
	required, err := service.SetupRequired()
	if err != nil {
		return database.User{}, Session{}, err
	}
	if !required {
		return database.User{}, Session{}, ErrSetupComplete
	}
	hash, err := hashPassword(password)
	if err != nil {
		return database.User{}, Session{}, err
	}
	user, err := service.Store.CreateAdminUser(email, hash)
	if err != nil {
		return database.User{}, Session{}, err
	}
	session, err := service.createSession(user.ID)
	if err != nil {
		return database.User{}, Session{}, err
	}
	return user, session, nil
}

func (service Service) Login(email string, password string) (database.User, Session, error) {
	if service.Store == nil {
		return database.User{}, Session{}, ErrAuthenticationStore
	}
	_, _ = service.Store.DeleteExpiredSessions(time.Now().UTC())
	user, err := service.Store.UserByEmail(email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.User{}, Session{}, ErrInvalidCredentials
		}
		return database.User{}, Session{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return database.User{}, Session{}, ErrInvalidCredentials
	}
	session, err := service.createSession(user.ID)
	if err != nil {
		return database.User{}, Session{}, err
	}
	return user, session, nil
}

func (service Service) UserForToken(token string) (database.User, error) {
	if service.Store == nil {
		return database.User{}, ErrAuthenticationStore
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return database.User{}, ErrInvalidCredentials
	}
	user, err := service.Store.UserBySessionHash(HashToken(token), time.Now().UTC())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.User{}, ErrInvalidCredentials
		}
		return database.User{}, err
	}
	return user, nil
}

func (service Service) Logout(token string) error {
	if service.Store == nil {
		return ErrAuthenticationStore
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return service.Store.DeleteSession(HashToken(token))
}

func (service Service) createSession(userID string) (Session, error) {
	token, err := newToken()
	if err != nil {
		return Session{}, err
	}
	ttl := service.SessionTTL
	if ttl == 0 {
		ttl = 30 * 24 * time.Hour
	}
	session := Session{
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
	if err := service.Store.CreateSession(HashToken(token), userID, session.ExpiresAt); err != nil {
		return Session{}, err
	}
	return session, nil
}

func hashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", ErrPasswordTooShort
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
