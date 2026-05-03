package soak

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigExpandsEnvironmentValues(t *testing.T) {
	t.Setenv("SOAK_BASE_URL", "http://localhost:8080")
	t.Setenv("SOAK_TOKEN", "secret-token")
	path := filepath.Join(t.TempDir(), "config.json")
	body := `{
		"base_url": "${SOAK_BASE_URL}",
		"session_id": "session-1",
		"user_token": "${SOAK_TOKEN}",
		"rounds": 50,
		"scenario": {
			"name": "the-city",
			"objective": "测试长会话稳定性",
			"seed_prompt": "从 the city 开始"
		},
		"player_model": {"provider": "mock"},
		"judge_model": {"provider": "mock"},
		"timeout_seconds": 30,
		"output_path": "reports/soak/report.json"
	}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("expected config fixture write to succeed, got %v", err)
	}

	config, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}

	if config.BaseURL != "http://localhost:8080" {
		t.Fatalf("expected env-expanded base url, got %q", config.BaseURL)
	}
	if config.UserToken != "secret-token" {
		t.Fatalf("expected env-expanded token, got %q", config.UserToken)
	}
	if config.Rounds != 50 {
		t.Fatalf("expected 50 rounds, got %d", config.Rounds)
	}
}
