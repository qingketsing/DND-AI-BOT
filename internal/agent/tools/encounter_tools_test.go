package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/service"
)

func TestGetEncounterToolCallUsesSessionID(t *testing.T) {
	svc := &fakeEncounterToolService{result: newToolEncounter()}
	tool := NewGetEncounterTool(svc)

	output, err := tool.Call(context.Background(), CallInput{SessionID: "session-1"})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.getSessionID != "session-1" {
		t.Fatalf("expected session id %q, got %q", "session-1", svc.getSessionID)
	}
	if output.ToolName != "get_encounter" {
		t.Fatalf("expected tool name %q, got %q", "get_encounter", output.ToolName)
	}
}

func TestApplyDamageToolCallMapsArgs(t *testing.T) {
	svc := &fakeEncounterToolService{result: newToolEncounter()}
	tool := NewApplyDamageTool(svc)
	now := time.Date(2026, 4, 6, 15, 0, 0, 0, time.UTC)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"target_id":"goblin-1","amount":5}`),
		Now:       now,
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.applyDamageInput.TargetID != "goblin-1" || svc.applyDamageInput.Amount != 5 || !svc.applyDamageNow.Equal(now) {
		t.Fatalf("expected apply damage input to be mapped, got %+v at %v", svc.applyDamageInput, svc.applyDamageNow)
	}
}

func TestHealToolCallMapsArgs(t *testing.T) {
	svc := &fakeEncounterToolService{result: newToolEncounter()}
	tool := NewHealTool(svc)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"target_id":"hero-1","amount":4}`),
		Now:       time.Date(2026, 4, 6, 15, 5, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.healInput.TargetID != "hero-1" || svc.healInput.Amount != 4 {
		t.Fatalf("expected heal input to be mapped, got %+v", svc.healInput)
	}
}

func TestAdvanceTurnToolCallUsesSessionIDAndNow(t *testing.T) {
	svc := &fakeEncounterToolService{result: newToolEncounter()}
	tool := NewAdvanceTurnTool(svc)
	now := time.Date(2026, 4, 6, 15, 10, 0, 0, time.UTC)

	_, err := tool.Call(context.Background(), CallInput{SessionID: "session-1", Now: now})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.advanceTurnInput.SessionID != "session-1" || !svc.advanceTurnNow.Equal(now) {
		t.Fatalf("expected advance turn input to be mapped, got %+v at %v", svc.advanceTurnInput, svc.advanceTurnNow)
	}
}

func TestAddEffectToolCallMapsArgs(t *testing.T) {
	svc := &fakeEncounterToolService{result: newToolEncounter()}
	tool := NewAddEffectTool(svc)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"target_id":"hero-1","effect_id":"effect-1","type":"stunned","source":"spell","duration":2}`),
		Now:       time.Date(2026, 4, 6, 15, 15, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.addEffectInput.TargetID != "hero-1" || svc.addEffectInput.Effect.Type != combat.EffectStunned {
		t.Fatalf("expected add effect input to be mapped, got %+v", svc.addEffectInput)
	}
}

func TestRemoveEffectToolCallMapsArgs(t *testing.T) {
	svc := &fakeEncounterToolService{result: newToolEncounter()}
	tool := NewRemoveEffectTool(svc)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"target_id":"hero-1","effect_id":"effect-1"}`),
		Now:       time.Date(2026, 4, 6, 15, 20, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.removeEffectInput.TargetID != "hero-1" || svc.removeEffectInput.EffectID != "effect-1" {
		t.Fatalf("expected remove effect input to be mapped, got %+v", svc.removeEffectInput)
	}
}

func TestCanActToolCallMapsArgs(t *testing.T) {
	svc := &fakeEncounterToolService{canActResult: true}
	tool := NewCanActTool(svc)

	output, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"target_id":"hero-1"}`),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.canActInput.TargetID != "hero-1" {
		t.Fatalf("expected can act input to be mapped, got %+v", svc.canActInput)
	}
	result, ok := output.Content.(CanActResult)
	if !ok || !result.CanAct {
		t.Fatalf("expected CanActResult true, got %#v", output.Content)
	}
}

func TestEncounterToolsRejectInvalidInput(t *testing.T) {
	tool := NewApplyDamageTool(&fakeEncounterToolService{})

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"amount":"bad"}`),
	})
	if !errors.Is(err, ErrInvalidToolInput) {
		t.Fatalf("expected ErrInvalidToolInput, got %v", err)
	}
}

type fakeEncounterToolService struct {
	result            *combat.Encounter
	err               error
	canActResult      bool
	getSessionID      string
	applyDamageInput  service.ApplyDamageInput
	applyDamageNow    time.Time
	healInput         service.HealInput
	healNow           time.Time
	advanceTurnInput  service.AdvanceTurnInput
	advanceTurnNow    time.Time
	addEffectInput    service.AddEffectInput
	addEffectNow      time.Time
	removeEffectInput service.RemoveEffectInput
	removeEffectNow   time.Time
	canActInput       service.CanActInput
}

func newToolEncounter() *combat.Encounter {
	return combat.NewEncounter("encounter-1", "session-1", []combat.Combatant{
		combat.NewCombatant("hero-1", "Hero", combat.CombatSideParty, 20, 15, 12),
		combat.NewCombatant("goblin-1", "Goblin", combat.CombatSideEnemy, 8, 13, 10),
	}, time.Date(2026, 4, 6, 15, 0, 0, 0, time.UTC))
}

func (f *fakeEncounterToolService) Create(ctx context.Context, input service.CreateEncounterInput, now time.Time) (*combat.Encounter, error) {
	_ = ctx
	_ = input
	_ = now
	return f.result, f.err
}

func (f *fakeEncounterToolService) GetBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error) {
	_ = ctx
	f.getSessionID = sessionID
	return f.result, f.err
}

func (f *fakeEncounterToolService) ApplyDamage(ctx context.Context, input service.ApplyDamageInput, now time.Time) (*combat.Encounter, error) {
	_ = ctx
	f.applyDamageInput = input
	f.applyDamageNow = now
	return f.result, f.err
}

func (f *fakeEncounterToolService) Heal(ctx context.Context, input service.HealInput, now time.Time) (*combat.Encounter, error) {
	_ = ctx
	f.healInput = input
	f.healNow = now
	return f.result, f.err
}

func (f *fakeEncounterToolService) AdvanceTurn(ctx context.Context, input service.AdvanceTurnInput, now time.Time) (*combat.Encounter, error) {
	_ = ctx
	f.advanceTurnInput = input
	f.advanceTurnNow = now
	return f.result, f.err
}

func (f *fakeEncounterToolService) AddEffect(ctx context.Context, input service.AddEffectInput, now time.Time) (*combat.Encounter, error) {
	_ = ctx
	f.addEffectInput = input
	f.addEffectNow = now
	return f.result, f.err
}

func (f *fakeEncounterToolService) RemoveEffect(ctx context.Context, input service.RemoveEffectInput, now time.Time) (*combat.Encounter, error) {
	_ = ctx
	f.removeEffectInput = input
	f.removeEffectNow = now
	return f.result, f.err
}

func (f *fakeEncounterToolService) CanAct(ctx context.Context, input service.CanActInput) (bool, error) {
	_ = ctx
	f.canActInput = input
	return f.canActResult, f.err
}
