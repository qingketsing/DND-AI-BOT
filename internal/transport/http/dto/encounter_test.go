package dto

import (
	"testing"
	"time"

	"DND-AI-BOT/internal/game/combat"
)

func TestToEncounterResponseMapsDomainModel(t *testing.T) {
	now := time.Date(2026, 4, 5, 18, 30, 0, 0, time.UTC)
	encounter := combat.NewEncounter("encounter-1", "session-1", []combat.Combatant{
		{
			ID:         "hero-1",
			Name:       "Hero",
			Side:       combat.CombatSideParty,
			CurrentHP:  20,
			MaxHP:      20,
			ArmorClass: 15,
			Initiative: 12,
			Status:     combat.CombatStatusActive,
			Effects: []combat.StatusEffect{
				{ID: "effect-1", Type: combat.EffectStunned, Source: "spell", Duration: 1},
			},
		},
	}, now)

	response := ToEncounterResponse(encounter)

	if response.ID != "encounter-1" || response.SessionID != "session-1" {
		t.Fatalf("expected ids to be mapped, got %+v", response)
	}
	if len(response.Combatants) != 1 || response.Combatants[0].ID != "hero-1" {
		t.Fatalf("expected combatants to be mapped, got %+v", response.Combatants)
	}
	if len(response.Combatants[0].Effects) != 1 || response.Combatants[0].Effects[0].Type != "stunned" {
		t.Fatalf("expected effects to be mapped, got %+v", response.Combatants[0].Effects)
	}
}

func TestToCombatantsMapsDTOToDomainModel(t *testing.T) {
	combatants := ToCombatants([]CombatantDTO{
		{
			ID:         "goblin-1",
			Name:       "Goblin",
			Side:       "enemy",
			CurrentHP:  8,
			MaxHP:      8,
			ArmorClass: 13,
			Initiative: 10,
			Status:     "active",
			Effects: []StatusEffectDTO{
				{ID: "effect-1", Type: "poisoned", Source: "trap", Duration: 2},
			},
		},
	})

	if len(combatants) != 1 {
		t.Fatalf("expected 1 combatant, got %d", len(combatants))
	}
	if combatants[0].Side != combat.CombatSideEnemy || combatants[0].Status != combat.CombatStatusActive {
		t.Fatalf("expected combatant enums to be mapped, got %+v", combatants[0])
	}
	if len(combatants[0].Effects) != 1 || combatants[0].Effects[0].Type != combat.EffectPoisoned {
		t.Fatalf("expected effects to be mapped, got %+v", combatants[0].Effects)
	}
}

func TestToCombatantDTOAndStatusEffectMapFields(t *testing.T) {
	combatantItem := ToCombatantDTO(combat.Combatant{
		ID:         "hero-1",
		Name:       "Hero",
		Side:       combat.CombatSideParty,
		CurrentHP:  18,
		MaxHP:      20,
		ArmorClass: 15,
		Initiative: 12,
		Status:     combat.CombatStatusDown,
		Effects: []combat.StatusEffect{
			{ID: "effect-1", Type: combat.EffectInvisible, Source: "cloak", Duration: 3},
		},
	})
	effect := ToStatusEffect(StatusEffectDTO{ID: "effect-2", Type: "charmed", Source: "spell", Duration: 1})

	if combatantItem.Side != "party" || combatantItem.Status != "down" {
		t.Fatalf("expected combatant dto enums to be mapped, got %+v", combatantItem)
	}
	if len(combatantItem.Effects) != 1 || combatantItem.Effects[0].Type != "invisible" {
		t.Fatalf("expected combatant dto effects to be mapped, got %+v", combatantItem.Effects)
	}
	if effect.Type != combat.EffectCharmed || effect.Source != "spell" {
		t.Fatalf("expected status effect helper to map fields, got %+v", effect)
	}
}
