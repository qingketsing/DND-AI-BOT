package app

import (
	"net/http"
	"testing"

	"DND-AI-BOT/internal/bootstrap"
)

func TestNewAppBuildsCoreServices(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")

	application, err := NewApp(&bootstrap.RuntimeDependencies{})
	if err != nil {
		t.Fatalf("expected app build to succeed, got %v", err)
	}

	if application == nil {
		t.Fatal("expected app to be created")
	}
	if application.Handler == nil {
		t.Fatal("expected handler to be initialized")
	}
	if application.SessionService == nil {
		t.Fatal("expected session service to be initialized")
	}
	if application.AgentService == nil {
		t.Fatal("expected agent service to be initialized")
	}
	if application.GameStateService == nil {
		t.Fatal("expected game state service to be initialized")
	}
	if application.EncounterService == nil {
		t.Fatal("expected encounter service to be initialized")
	}

	if _, ok := application.Handler.(http.Handler); !ok {
		t.Fatal("expected handler to implement http.Handler")
	}
}
