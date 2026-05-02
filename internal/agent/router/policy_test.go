package router

import (
	"testing"

	"DND-AI-BOT/internal/agent/client"
	"DND-AI-BOT/internal/agent/intent"
)

func TestPolicyRoutesStatusQueryToFast(t *testing.T) {
	decision := DefaultPolicy().Decide(intent.Result{Kind: intent.KindStatusQuery})

	if decision.ModelRole != client.ModelRoleFast {
		t.Fatalf("expected model role %q, got %q", client.ModelRoleFast, decision.ModelRole)
	}
	if decision.MaxSteps != 2 {
		t.Fatalf("expected max steps 2, got %d", decision.MaxSteps)
	}
	if decision.Reason == "" {
		t.Fatal("expected route reason")
	}
}

func TestPolicyRoutesCharacterDraftToFast(t *testing.T) {
	decision := DefaultPolicy().Decide(intent.Result{Kind: intent.KindCharacterDraft})

	if decision.ModelRole != client.ModelRoleFast {
		t.Fatalf("expected model role %q, got %q", client.ModelRoleFast, decision.ModelRole)
	}
}

func TestPolicyRoutesCombatActionToPrimary(t *testing.T) {
	decision := DefaultPolicy().Decide(intent.Result{Kind: intent.KindCombatAction})

	if decision.ModelRole != client.ModelRolePrimary {
		t.Fatalf("expected model role %q, got %q", client.ModelRolePrimary, decision.ModelRole)
	}
	if decision.MaxSteps != 8 {
		t.Fatalf("expected max steps 8, got %d", decision.MaxSteps)
	}
}

func TestPolicyRoutesUnknownToPrimary(t *testing.T) {
	decision := DefaultPolicy().Decide(intent.Result{Kind: intent.KindUnknown})

	if decision.ModelRole != client.ModelRolePrimary {
		t.Fatalf("expected model role %q, got %q", client.ModelRolePrimary, decision.ModelRole)
	}
}
