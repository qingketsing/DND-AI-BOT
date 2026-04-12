package prompt

import (
	"strings"
	"testing"

	"DND-AI-BOT/internal/model"
)

func TestComposeSystemPromptAppendsWarmupSections(t *testing.T) {
	base := "base prompt"
	warmup := model.WarmupBundle{
		BaseRulesSummary:      "rules summary",
		BaseLoreSummary:       "lore summary",
		CharacterRulesSummary: "character summary",
	}

	result := ComposeSystemPrompt(base, warmup)

	for _, expected := range []string{
		"base prompt",
		"已知基础上下文",
		"rules summary",
		"lore summary",
		"character summary",
	} {
		if !strings.Contains(result, expected) {
			t.Fatalf("expected %q in composed prompt, got %q", expected, result)
		}
	}
}

func TestComposeSystemPromptSkipsEmptySections(t *testing.T) {
	result := ComposeSystemPrompt("base prompt", model.WarmupBundle{})
	if strings.Contains(result, "已知基础上下文") {
		t.Fatalf("expected no warmup header when bundle is empty, got %q", result)
	}
}
