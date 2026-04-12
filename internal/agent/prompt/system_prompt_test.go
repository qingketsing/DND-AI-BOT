package prompt

import (
	"strings"
	"testing"
)

func TestDefaultSystemPromptLimitsSupplementalSearchToOneRetry(t *testing.T) {
	if !strings.Contains(DefaultSystemPrompt, "最多补查 1 次") {
		t.Fatalf("expected prompt to limit supplemental search to one retry, got %q", DefaultSystemPrompt)
	}
}

func TestDefaultSystemPromptRequiresAgentContextForSessionFacts(t *testing.T) {
	if !strings.Contains(DefaultSystemPrompt, "get_agent_context") {
		t.Fatalf("expected prompt to mention get_agent_context for session facts, got %q", DefaultSystemPrompt)
	}
}
