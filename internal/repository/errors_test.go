package repository

import "testing"

func TestRepositoryErrorsAreComparable(t *testing.T) {
	if ErrUserNotFound == nil || ErrAuthSessionNotFound == nil {
		t.Fatal("expected repository not-found errors to be initialized")
	}
	if ErrUserNotFound == ErrAuthSessionNotFound {
		t.Fatal("expected user and auth session errors to be distinct values")
	}
	if ErrUserNotFound.Error() != "user not found" {
		t.Fatalf("expected ErrUserNotFound message %q, got %q", "user not found", ErrUserNotFound.Error())
	}
	if ErrAuthSessionNotFound.Error() != "auth session not found" {
		t.Fatalf("expected ErrAuthSessionNotFound message %q, got %q", "auth session not found", ErrAuthSessionNotFound.Error())
	}
}
