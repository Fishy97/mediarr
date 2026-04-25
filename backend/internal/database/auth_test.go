package database

import (
	"testing"
	"time"
)

func TestAdminUserAndSessionLifecycle(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	exists, err := store.AdminUserExists()
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("new database should not have an admin user")
	}

	user, err := store.CreateAdminUser("admin@example.test", "hash-not-plaintext")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID == "" || user.Email != "admin@example.test" {
		t.Fatalf("unexpected user: %#v", user)
	}

	exists, err = store.AdminUserExists()
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("admin user should exist after creation")
	}

	found, err := store.UserByEmail("admin@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if found.PasswordHash != "hash-not-plaintext" {
		t.Fatalf("password hash = %q", found.PasswordHash)
	}

	expiresAt := time.Now().UTC().Add(time.Hour)
	if err := store.CreateSession("session-hash", user.ID, expiresAt); err != nil {
		t.Fatal(err)
	}
	sessionUser, err := store.UserBySessionHash("session-hash", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if sessionUser.ID != user.ID {
		t.Fatalf("session user = %q, want %q", sessionUser.ID, user.ID)
	}

	if err := store.DeleteSession("session-hash"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UserBySessionHash("session-hash", time.Now().UTC()); err == nil {
		t.Fatal("deleted session should not authenticate")
	}
}
