package combat

import (
	"errors"
	"testing"
	"time"
)

func TestNewEncounterInitializesState(t *testing.T) {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	encounter := NewEncounter("encounter-1", "session-1", []Combatant{
		NewCombatant("c1", "Hero", CombatSideParty, 20, 15, 12),
	}, now)

	if encounter.ID != "encounter-1" {
		t.Fatalf("expected encounter id encounter-1, got %q", encounter.ID)
	}
	if encounter.SessionID != "session-1" {
		t.Fatalf("expected session id session-1, got %q", encounter.SessionID)
	}
	if encounter.Round != 1 {
		t.Fatalf("expected round 1, got %d", encounter.Round)
	}
	if encounter.TurnIndex != 0 {
		t.Fatalf("expected turn index 0, got %d", encounter.TurnIndex)
	}
	if !encounter.StartedAt.Equal(now) || !encounter.UpdatedAt.Equal(now) {
		t.Fatalf("expected timestamps %v, got started=%v updated=%v", now, encounter.StartedAt, encounter.UpdatedAt)
	}
}

func TestCurrentCombatantReturnsCurrentUnit(t *testing.T) {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	encounter := NewEncounter("encounter-1", "session-1", []Combatant{
		NewCombatant("c1", "Hero", CombatSideParty, 20, 15, 12),
		NewCombatant("c2", "Goblin", CombatSideEnemy, 8, 13, 10),
	}, now)

	current, ok := encounter.CurrentCombatant()
	if !ok {
		t.Fatal("expected current combatant to exist")
	}
	if current.ID != "c1" {
		t.Fatalf("expected current combatant c1, got %q", current.ID)
	}
}

func TestAdvanceTurnMovesToNextCombatantAndRound(t *testing.T) {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	encounter := NewEncounter("encounter-1", "session-1", []Combatant{
		NewCombatant("c1", "Hero", CombatSideParty, 20, 15, 12),
		NewCombatant("c2", "Goblin", CombatSideEnemy, 8, 13, 10),
	}, now)

	encounter.AdvanceTurn(now.Add(time.Minute))
	if encounter.TurnIndex != 1 {
		t.Fatalf("expected turn index 1, got %d", encounter.TurnIndex)
	}
	if encounter.Round != 1 {
		t.Fatalf("expected round 1, got %d", encounter.Round)
	}

	encounter.AdvanceTurn(now.Add(2 * time.Minute))
	if encounter.TurnIndex != 0 {
		t.Fatalf("expected turn index 0, got %d", encounter.TurnIndex)
	}
	if encounter.Round != 2 {
		t.Fatalf("expected round 2, got %d", encounter.Round)
	}
}

func TestApplyDamageReducesHPAndSetsDownStatus(t *testing.T) {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	encounter := NewEncounter("encounter-1", "session-1", []Combatant{
		NewCombatant("c1", "Hero", CombatSideParty, 20, 15, 12),
	}, now)

	if err := encounter.ApplyDamage("c1", 25, now.Add(time.Minute)); err != nil {
		t.Fatalf("expected apply damage to succeed, got %v", err)
	}

	target, err := encounter.FindCombatant("c1")
	if err != nil {
		t.Fatalf("expected combatant to exist, got %v", err)
	}
	if target.CurrentHP != 0 {
		t.Fatalf("expected current hp 0, got %d", target.CurrentHP)
	}
	if target.Status != CombatStatusDown {
		t.Fatalf("expected status %q, got %q", CombatStatusDown, target.Status)
	}
}

func TestHealRestoresHPAndActiveStatus(t *testing.T) {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	encounter := NewEncounter("encounter-1", "session-1", []Combatant{
		NewCombatant("c1", "Hero", CombatSideParty, 20, 15, 12),
	}, now)

	if err := encounter.ApplyDamage("c1", 20, now.Add(time.Minute)); err != nil {
		t.Fatalf("expected apply damage to succeed, got %v", err)
	}
	if err := encounter.Heal("c1", 5, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("expected heal to succeed, got %v", err)
	}

	target, err := encounter.FindCombatant("c1")
	if err != nil {
		t.Fatalf("expected combatant to exist, got %v", err)
	}
	if target.CurrentHP != 5 {
		t.Fatalf("expected current hp 5, got %d", target.CurrentHP)
	}
	if target.Status != CombatStatusActive {
		t.Fatalf("expected status %q, got %q", CombatStatusActive, target.Status)
	}
}

func TestFindCombatantReturnsErrorWhenMissing(t *testing.T) {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	encounter := NewEncounter("encounter-1", "session-1", []Combatant{
		NewCombatant("c1", "Hero", CombatSideParty, 20, 15, 12),
	}, now)

	_, err := encounter.FindCombatant("missing")
	if !errors.Is(err, ErrCombatantNotFound) {
		t.Fatalf("expected ErrCombatantNotFound, got %v", err)
	}
}

func TestCanActReturnsFalseForStunnedCombatant(t *testing.T) {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	combatant := NewCombatant("c1", "Hero", CombatSideParty, 20, 15, 12)
	combatant.AddEffect(StatusEffect{ID: "e1", Type: EffectStunned, Source: "spell", Duration: 1})
	encounter := NewEncounter("encounter-1", "session-1", []Combatant{combatant}, now)

	canAct, err := encounter.CanAct("c1")
	if err != nil {
		t.Fatalf("expected can act to succeed, got %v", err)
	}
	if canAct {
		t.Fatal("expected stunned combatant to be unable to act")
	}
}

func TestCanActReturnsFalseForDownCombatant(t *testing.T) {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	combatant := NewCombatant("c1", "Hero", CombatSideParty, 20, 15, 12)
	combatant.Status = CombatStatusDown
	encounter := NewEncounter("encounter-1", "session-1", []Combatant{combatant}, now)

	canAct, err := encounter.CanAct("c1")
	if err != nil {
		t.Fatalf("expected can act to succeed, got %v", err)
	}
	if canAct {
		t.Fatal("expected down combatant to be unable to act")
	}
}
