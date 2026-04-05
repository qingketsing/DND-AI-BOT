package app

import (
	"net/http"
	"testing"

	"DND-AI-BOT/internal/bootstrap"
)

func TestNewAppBuildsCoreServices(t *testing.T) {
	application := NewApp(&bootstrap.RuntimeDependencies{})

	if application == nil {
		t.Fatal("expected app to be created")
	}
	if application.Handler == nil {
		t.Fatal("expected handler to be initialized")
	}
	if application.SessionService == nil {
		t.Fatal("expected session service to be initialized")
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
