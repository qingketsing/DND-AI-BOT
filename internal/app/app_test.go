package app

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"DND-AI-BOT/internal/bootstrap"
	"DND-AI-BOT/internal/queue"
	goredis "github.com/redis/go-redis/v9"
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
	if application.AuthService == nil {
		t.Fatal("expected auth service to be initialized")
	}
	if application.GameStateService == nil {
		t.Fatal("expected game state service to be initialized")
	}
	if application.EncounterService == nil {
		t.Fatal("expected encounter service to be initialized")
	}
	if application.SessionMemoryService == nil {
		t.Fatal("expected session memory service to be initialized")
	}
	if application.SessionMemoryRefresher == nil {
		t.Fatal("expected session memory refresher to be initialized")
	}
	if application.KnowledgeWarmupService == nil {
		t.Fatal("expected knowledge warmup service to be initialized")
	}

	if _, ok := application.Handler.(http.Handler); !ok {
		t.Fatal("expected handler to implement http.Handler")
	}
}

func TestNewAppRejectsAsyncMessageModeWithoutPersistentDependencies(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")
	t.Setenv("ASYNC_MESSAGE_ENABLED", "true")

	_, err := NewApp(&bootstrap.RuntimeDependencies{})
	if err == nil {
		t.Fatal("expected async message mode without db and redis to fail")
	}
}

func TestNewAppRejectsAsyncMessageModeWithoutRabbitMQPublisher(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")
	t.Setenv("ASYNC_MESSAGE_ENABLED", "true")

	_, err := NewApp(&bootstrap.RuntimeDependencies{
		DB:          &sql.DB{},
		RedisClient: goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:0"}),
	})
	if err == nil {
		t.Fatal("expected async message mode without rabbitmq publisher to fail")
	}
}

func TestNewAppBuildsOutboxDispatcherWhenAsyncMessageModeEnabled(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")
	t.Setenv("ASYNC_MESSAGE_ENABLED", "true")

	application, err := NewApp(&bootstrap.RuntimeDependencies{
		DB:                &sql.DB{},
		RedisClient:       goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:0"}),
		RabbitMQPublisher: &fakeAppMessageJobPublisher{},
	})
	if err != nil {
		t.Fatalf("expected async app build to succeed, got %v", err)
	}
	if application.AsyncMessageService == nil {
		t.Fatal("expected async message service to be configured")
	}
	if application.OutboxDispatcher == nil {
		t.Fatal("expected outbox dispatcher to be configured")
	}
	if application.OutboxDispatchLoop == nil {
		t.Fatal("expected outbox dispatch loop to be configured")
	}
	if err := application.StartBackgrounds(context.Background()); err != nil {
		t.Fatalf("expected background start to succeed, got %v", err)
	}
	defer application.Close()
}

func TestNewAppProtectsAuthMeRoute(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")

	application, err := NewApp(&bootstrap.RuntimeDependencies{})
	if err != nil {
		t.Fatalf("expected app build to succeed, got %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	recorder := httptest.NewRecorder()

	application.Handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestNewAppProtectsSessionsRoute(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")

	application, err := NewApp(&bootstrap.RuntimeDependencies{})
	if err != nil {
		t.Fatalf("expected app build to succeed, got %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	recorder := httptest.NewRecorder()

	application.Handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestNewAppExposesMetricsWithoutAuth(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")
	t.Setenv("METRICS_ALLOWED_CIDRS", "192.0.2.0/24")

	application, err := NewApp(&bootstrap.RuntimeDependencies{})
	if err != nil {
		t.Fatalf("expected app build to succeed, got %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()

	application.Handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestNewAppProtectsMetricsWhenRemoteAddressIsNotAllowed(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")
	t.Setenv("METRICS_ALLOWED_CIDRS", "127.0.0.1/32")

	application, err := NewApp(&bootstrap.RuntimeDependencies{})
	if err != nil {
		t.Fatalf("expected app build to succeed, got %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	request.RemoteAddr = "203.0.113.10:12345"
	recorder := httptest.NewRecorder()

	application.Handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}
}

type fakeAppMessageJobPublisher struct{}

func (f *fakeAppMessageJobPublisher) Publish(ctx context.Context, payload queue.MessageJobPayload) error {
	_ = ctx
	_ = payload
	return nil
}
