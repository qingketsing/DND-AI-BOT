package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"DND-AI-BOT/internal/bootstrap"
	goredis "github.com/redis/go-redis/v9"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestIntegrationMigrationsCreateSessionsListCompositeIndex(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("integration test disabled; set INTEGRATION_TEST=1 to enable")
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("expected getwd to succeed, got %v", err)
	}
	if err := os.Chdir(filepath.Join("..", "..")); err != nil {
		t.Fatalf("expected chdir to repo root to succeed, got %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("expected restore working directory to succeed, got %v", err)
		}
	})

	ctx := context.Background()
	deps := openAppIntegrationDependencies(t)
	defer deps.DB.Close()
	defer deps.RedisClient.Close()

	if err = bootstrap.RunEmbeddedMigrations(ctx, deps.DB); err != nil {
		t.Fatalf("expected migrations to run, got %v", err)
	}

	var indexDefinition string
	err = deps.DB.QueryRowContext(ctx, `
		SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = 'public'
		  AND tablename = 'sessions'
		  AND indexname = 'idx_sessions_user_id_updated_at_desc'
	`).Scan(&indexDefinition)
	if err != nil {
		t.Fatalf("expected composite sessions index to exist, got %v", err)
	}
	if indexDefinition == "" {
		t.Fatal("expected composite sessions index definition to be present")
	}
}

func TestIntegrationHTTPAgentFlow(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("integration test disabled; set INTEGRATION_TEST=1 to enable")
	}

	ctx := context.Background()
	deps := openAppIntegrationDependencies(t)
	defer deps.DB.Close()
	defer deps.RedisClient.Close()

	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "60")

	if err := bootstrap.RunEmbeddedMigrations(ctx, deps.DB); err != nil {
		t.Fatalf("expected migrations to run, got %v", err)
	}
	resetAppIntegrationState(t, ctx, deps.DB, deps.RedisClient)

	application, err := NewApp(deps)
	if err != nil {
		t.Fatalf("expected app build to succeed, got %v", err)
	}

	server := httptest.NewServer(application.Handler)
	defer server.Close()

	createResp, err := http.Post(server.URL+"/sessions", "application/json", bytes.NewBufferString(`{"channel":"web"}`))
	if err != nil {
		t.Fatalf("expected create session request to succeed, got %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create session status %d, got %d", http.StatusCreated, createResp.StatusCode)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("expected create session response to decode, got %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected created session id to be present")
	}

	body := bytes.NewBufferString(`{"user_id":"user-1","user_name":"Alice","content":"hello world"}`)
	messageResp, err := http.Post(server.URL+"/sessions/"+created.ID+"/messages", "application/json", body)
	if err != nil {
		t.Fatalf("expected send message request to succeed, got %v", err)
	}
	defer messageResp.Body.Close()
	if messageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected send message status %d, got %d", http.StatusOK, messageResp.StatusCode)
	}

	var updated struct {
		History []struct {
			Source  string `json:"source"`
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"history"`
	}
	if err := json.NewDecoder(messageResp.Body).Decode(&updated); err != nil {
		t.Fatalf("expected send message response to decode, got %v", err)
	}
	if len(updated.History) != 2 {
		t.Fatalf("expected 2 history records, got %d", len(updated.History))
	}
	if updated.History[1].Source != "agent" {
		t.Fatalf("expected second history source %q, got %q", "agent", updated.History[1].Source)
	}
	if updated.History[1].Message.Content != "mock reply: hello world" {
		t.Fatalf("expected mock runtime reply, got %q", updated.History[1].Message.Content)
	}
}

func openAppIntegrationDependencies(t *testing.T) *bootstrap.RuntimeDependencies {
	t.Helper()

	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Fatal("expected POSTGRES_TEST_DSN to be set")
	}
	redisAddr := os.Getenv("REDIS_TEST_ADDR")
	if redisAddr == "" {
		t.Fatal("expected REDIS_TEST_ADDR to be set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("expected postgres open to succeed, got %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("expected postgres ping to succeed, got %v", err)
	}

	redisClient := goredis.NewClient(&goredis.Options{Addr: redisAddr})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("expected redis ping to succeed, got %v", err)
	}

	return &bootstrap.RuntimeDependencies{
		DB:          db,
		RedisClient: redisClient,
	}
}

func resetAppIntegrationState(t *testing.T, ctx context.Context, db *sql.DB, client *goredis.Client) {
	t.Helper()

	if _, err := db.ExecContext(ctx, `
		TRUNCATE TABLE
			session_messages,
			sessions,
			game_states,
			encounters
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("expected database cleanup to succeed, got %v", err)
	}
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("expected redis cleanup to succeed, got %v", err)
	}
}
