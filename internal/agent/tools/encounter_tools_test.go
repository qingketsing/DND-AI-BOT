package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/service"
)

func TestCreateEncounterToolCallMapsArgs(t *testing.T) {
	svc := &fakeEncounterToolService{result: newToolEncounter()}
	tool := NewCreateEncounterTool(svc)
	now := time.Date(2026, 4, 16, 18, 0, 0, 0, time.UTC)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw: json.RawMessage(`{
			"id":"encounter-custom",
			"combatants":[
				{"id":"hero-1","name":"青稞","side":"party","current_hp":7,"max_hp":7,"armor_class":12,"initiative":17},
				{"id":"goblin-1","name":"地精#1","side":"enemy","current_hp":7,"max_hp":7,"armor_class":15,"initiative":12}
			]
		}`),
		Now: now,
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.createInput.ID != "encounter-custom" || svc.createInput.SessionID != "session-1" {
		t.Fatalf("expected create input ids to be mapped, got %+v", svc.createInput)
	}
	if len(svc.createInput.Combatants) != 2 {
		t.Fatalf("expected two combatants, got %d", len(svc.createInput.Combatants))
	}
	goblin := svc.createInput.Combatants[1]
	if goblin.ID != "goblin-1" || goblin.Side != combat.CombatSideEnemy || goblin.CurrentHP != 7 || goblin.ArmorClass != 15 || goblin.Initiative != 12 {
		t.Fatalf("expected goblin combatant to be mapped, got %+v", goblin)
	}
	if !svc.createNow.Equal(now) {
		t.Fatalf("expected create time %v, got %v", now, svc.createNow)
	}
}

func TestCreateEncounterToolSpecRequiresConfirmedPlayerCombatStats(t *testing.T) {
	spec := NewCreateEncounterTool(&fakeEncounterToolService{}).Spec()
	for _, expected := range []string{
		"玩家角色",
		"HP",
		"AC",
		"先攻",
		"已确认",
		"规则推导",
	} {
		if !strings.Contains(spec.Description, expected) {
			t.Fatalf("expected create_encounter description to mention %q, got %q", expected, spec.Description)
		}
	}
}

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

func TestResolveAttackActionToolCallMapsArgs(t *testing.T) {
	svc := &fakeEncounterToolService{
		resolveAttackResult: service.ResolveAttackActionResult{
			EncounterExists: true,
			ActionResolved:  true,
			Hit:             true,
			TargetHP:        0,
			TargetMaxHP:     8,
			TargetDown:      true,
		},
	}
	tool := NewResolveAttackActionTool(svc)
	now := time.Date(2026, 4, 24, 17, 0, 0, 0, time.UTC)

	output, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw: json.RawMessage(`{
			"attacker_id":"hero-1",
			"target_id":"goblin-1",
			"attack_bonus":5,
			"damage_dice":"1d8",
			"damage_bonus":3,
			"advance_turn":true
		}`),
		Now: now,
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.resolveAttackInput.AttackerID != "hero-1" || svc.resolveAttackInput.TargetID != "goblin-1" {
		t.Fatalf("expected resolve attack input to be mapped, got %+v", svc.resolveAttackInput)
	}
	if svc.resolveAttackInput.DamageDice != "1d8" || svc.resolveAttackInput.DamageBonus != 3 || !svc.resolveAttackInput.AdvanceTurn {
		t.Fatalf("expected damage fields to be mapped, got %+v", svc.resolveAttackInput)
	}
	if !svc.resolveAttackNow.Equal(now) {
		t.Fatalf("expected resolve attack time %v, got %v", now, svc.resolveAttackNow)
	}
	result, ok := output.Content.(service.ResolveAttackActionResult)
	if !ok || !result.Hit || !result.TargetDown {
		t.Fatalf("expected structured attack resolution result, got %#v", output.Content)
	}
}

func TestResolveAttackActionToolReturnsMissingEncounterResultWhenEncounterNotFound(t *testing.T) {
	tool := NewResolveAttackActionTool(&fakeEncounterToolService{resolveAttackErr: repository.ErrEncounterNotFound})

	output, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw: json.RawMessage(`{
			"attacker_id":"hero-1",
			"target_id":"goblin-1",
			"attack_bonus":5,
			"damage_dice":"1d8",
			"damage_bonus":3
		}`),
		Now: time.Date(2026, 4, 24, 17, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected encounter not found to be returned as tool content, got error %v", err)
	}
	result, ok := output.Content.(EncounterMissingResult)
	if !ok {
		t.Fatalf("expected EncounterMissingResult, got %#v", output.Content)
	}
	if result.EncounterExists || !result.RequiresCreateEncounter {
		t.Fatalf("expected missing encounter result to require create_encounter, got %+v", result)
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

func TestApplyDamageToolReturnsMissingEncounterResultWhenEncounterNotFound(t *testing.T) {
	tool := NewApplyDamageTool(&fakeEncounterToolService{err: repository.ErrEncounterNotFound})

	output, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"target_id":"goblin-1","amount":5}`),
		Now:       time.Date(2026, 4, 16, 18, 10, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected encounter not found to be returned as tool content, got error %v", err)
	}
	result, ok := output.Content.(EncounterMissingResult)
	if !ok {
		t.Fatalf("expected EncounterMissingResult, got %#v", output.Content)
	}
	if result.EncounterExists || !result.RequiresCreateEncounter {
		t.Fatalf("expected missing encounter result to require create_encounter, got %+v", result)
	}
}

type fakeEncounterToolService struct {
	result            *combat.Encounter
	err               error
	canActResult      bool
	createInput       service.CreateEncounterInput
	createNow         time.Time
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
	resolveAttackInput  service.ResolveAttackActionInput
	resolveAttackNow    time.Time
	resolveAttackResult service.ResolveAttackActionResult
	resolveAttackErr    error
}

func newToolEncounter() *combat.Encounter {
	return combat.NewEncounter("encounter-1", "session-1", []combat.Combatant{
		combat.NewCombatant("hero-1", "Hero", combat.CombatSideParty, 20, 15, 12),
		combat.NewCombatant("goblin-1", "Goblin", combat.CombatSideEnemy, 8, 13, 10),
	}, time.Date(2026, 4, 6, 15, 0, 0, 0, time.UTC))
}

func (f *fakeEncounterToolService) Create(ctx context.Context, input service.CreateEncounterInput, now time.Time) (*combat.Encounter, error) {
	_ = ctx
	f.createInput = input
	f.createNow = now
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

func (f *fakeEncounterToolService) ResolveAttackAction(ctx context.Context, input service.ResolveAttackActionInput, now time.Time) (service.ResolveAttackActionResult, error) {
	_ = ctx
	f.resolveAttackInput = input
	f.resolveAttackNow = now
	return f.resolveAttackResult, f.resolveAttackErr
}
