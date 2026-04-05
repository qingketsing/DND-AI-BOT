package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/repository"
)

func TestCreateEncounterSavesInitialEncounter(t *testing.T) {
	repo := newFakeEncounterRepository()
	service := NewEncounterService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)

	encounter, err := service.Create(ctx, CreateEncounterInput{
		ID:        "encounter-1",
		SessionID: "session-1",
		Combatants: []combat.Combatant{
			combat.NewCombatant("hero-1", "Hero", combat.CombatSideParty, 20, 15, 12),
		},
	}, now)
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if encounter.ID != "encounter-1" {
		t.Fatalf("expected encounter id %q, got %q", "encounter-1", encounter.ID)
	}
	if repo.saved == nil || repo.saved.SessionID != "session-1" {
		t.Fatal("expected repository to save created encounter")
	}
}

func TestGetEncounterReturnsStoredEncounter(t *testing.T) {
	repo := newFakeEncounterRepository()
	service := NewEncounterService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)
	existing := newTestEncounter(now)
	repo.bySessionID["session-1"] = existing

	encounter, err := service.GetBySessionID(ctx, "session-1")
	if err != nil {
		t.Fatalf("expected get to succeed, got %v", err)
	}
	if encounter.ID != existing.ID {
		t.Fatalf("expected encounter id %q, got %q", existing.ID, encounter.ID)
	}
}

func TestApplyDamageUpdatesEncounterAndPersists(t *testing.T) {
	repo := newFakeEncounterRepository()
	service := NewEncounterService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)
	existing := newTestEncounter(now)
	repo.bySessionID["session-1"] = existing

	updated, err := service.ApplyDamage(ctx, ApplyDamageInput{
		SessionID: "session-1",
		TargetID:  "goblin-1",
		Amount:    5,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected apply damage to succeed, got %v", err)
	}
	target, err := updated.FindCombatant("goblin-1")
	if err != nil {
		t.Fatalf("expected combatant to exist, got %v", err)
	}
	if target.CurrentHP != 3 {
		t.Fatalf("expected goblin HP 3, got %d", target.CurrentHP)
	}
}

func TestHealUpdatesEncounterAndPersists(t *testing.T) {
	repo := newFakeEncounterRepository()
	service := NewEncounterService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)
	existing := newTestEncounter(now)
	if err := existing.ApplyDamage("hero-1", 20, now.Add(time.Minute)); err != nil {
		t.Fatalf("expected setup damage to succeed, got %v", err)
	}
	repo.bySessionID["session-1"] = existing

	updated, err := service.Heal(ctx, HealInput{
		SessionID: "session-1",
		TargetID:  "hero-1",
		Amount:    5,
	}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("expected heal to succeed, got %v", err)
	}
	target, err := updated.FindCombatant("hero-1")
	if err != nil {
		t.Fatalf("expected combatant to exist, got %v", err)
	}
	if target.CurrentHP != 5 {
		t.Fatalf("expected hero HP 5, got %d", target.CurrentHP)
	}
	if target.Status != combat.CombatStatusActive {
		t.Fatalf("expected hero status %q, got %q", combat.CombatStatusActive, target.Status)
	}
}

func TestAdvanceTurnUpdatesEncounterAndPersists(t *testing.T) {
	repo := newFakeEncounterRepository()
	service := NewEncounterService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)
	existing := newTestEncounter(now)
	repo.bySessionID["session-1"] = existing

	updated, err := service.AdvanceTurn(ctx, AdvanceTurnInput{SessionID: "session-1"}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected advance turn to succeed, got %v", err)
	}
	if updated.TurnIndex != 1 {
		t.Fatalf("expected turn index 1, got %d", updated.TurnIndex)
	}
}

func TestAddEffectUpdatesEncounterAndPersists(t *testing.T) {
	repo := newFakeEncounterRepository()
	service := NewEncounterService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)
	existing := newTestEncounter(now)
	repo.bySessionID["session-1"] = existing

	updated, err := service.AddEffect(ctx, AddEffectInput{
		SessionID: "session-1",
		TargetID:  "hero-1",
		Effect:    combat.StatusEffect{ID: "effect-1", Type: combat.EffectStunned, Source: "spell", Duration: 1},
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected add effect to succeed, got %v", err)
	}
	target, err := updated.FindCombatant("hero-1")
	if err != nil {
		t.Fatalf("expected combatant to exist, got %v", err)
	}
	if !target.HasEffect(combat.EffectStunned) {
		t.Fatal("expected hero to have stunned effect")
	}
}

func TestRemoveEffectReturnsErrorWhenEffectMissing(t *testing.T) {
	repo := newFakeEncounterRepository()
	service := NewEncounterService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)
	existing := newTestEncounter(now)
	repo.bySessionID["session-1"] = existing

	_, err := service.RemoveEffect(ctx, RemoveEffectInput{
		SessionID: "session-1",
		TargetID:  "hero-1",
		EffectID:  "missing-effect",
	}, now.Add(time.Minute))
	if !errors.Is(err, ErrEffectNotFound) {
		t.Fatalf("expected ErrEffectNotFound, got %v", err)
	}
}

func TestCanActReturnsFalseForStunnedCombatant(t *testing.T) {
	repo := newFakeEncounterRepository()
	service := NewEncounterService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)
	existing := newTestEncounter(now)
	target, err := existing.FindCombatant("hero-1")
	if err != nil {
		t.Fatalf("expected combatant to exist, got %v", err)
	}
	target.AddEffect(combat.StatusEffect{ID: "effect-1", Type: combat.EffectStunned, Source: "spell", Duration: 1})
	repo.bySessionID["session-1"] = existing

	canAct, err := service.CanAct(ctx, CanActInput{SessionID: "session-1", TargetID: "hero-1"})
	if err != nil {
		t.Fatalf("expected can act to succeed, got %v", err)
	}
	if canAct {
		t.Fatal("expected stunned combatant to be unable to act")
	}
}

func newTestEncounter(now time.Time) *combat.Encounter {
	return combat.NewEncounter("encounter-1", "session-1", []combat.Combatant{
		combat.NewCombatant("hero-1", "Hero", combat.CombatSideParty, 20, 15, 12),
		combat.NewCombatant("goblin-1", "Goblin", combat.CombatSideEnemy, 8, 13, 10),
	}, now)
}

type fakeEncounterRepository struct {
	bySessionID map[string]*combat.Encounter
	saved       *combat.Encounter
	saveErr     error
	loadErr     error
}

func newFakeEncounterRepository() *fakeEncounterRepository {
	return &fakeEncounterRepository{
		bySessionID: make(map[string]*combat.Encounter),
	}
}

func (f *fakeEncounterRepository) Save(ctx context.Context, encounter *combat.Encounter) error {
	_ = ctx
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = encounter
	f.bySessionID[encounter.SessionID] = encounter
	return nil
}

func (f *fakeEncounterRepository) LoadBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error) {
	_ = ctx
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	encounter, ok := f.bySessionID[sessionID]
	if !ok {
		return nil, repository.ErrEncounterNotFound
	}
	return encounter, nil
}
