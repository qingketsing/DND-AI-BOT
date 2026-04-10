package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestUserAndAuthSessionRedactSecretsAndAllowNullFields(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	user := User{
		ID:           "user-1",
		Email:        "alice@example.com",
		PasswordHash: "secret-hash",
		DisplayName:  "Alice",
		Status:       UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("expected user marshal to succeed, got %v", err)
	}
	if strings.Contains(string(userJSON), "secret-hash") {
		t.Fatalf("expected password hash to be omitted, got %s", userJSON)
	}
	if strings.Contains(string(userJSON), "password_hash") {
		t.Fatalf("expected password hash field to be omitted, got %s", userJSON)
	}

	session := AuthSession{
		ID:        "auth-session-1",
		UserID:    "user-1",
		TokenHash: "session-secret",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}

	sessionJSON, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("expected auth session marshal to succeed, got %v", err)
	}
	if strings.Contains(string(sessionJSON), "session-secret") {
		t.Fatalf("expected token hash to be omitted, got %s", sessionJSON)
	}
	if strings.Contains(string(sessionJSON), "token_hash") {
		t.Fatalf("expected token hash field to be omitted, got %s", sessionJSON)
	}
	if !strings.Contains(string(sessionJSON), "\"user_agent\":null") {
		t.Fatalf("expected nullable user_agent field, got %s", sessionJSON)
	}
	if !strings.Contains(string(sessionJSON), "\"ip_address\":null") {
		t.Fatalf("expected nullable ip_address field, got %s", sessionJSON)
	}
}
