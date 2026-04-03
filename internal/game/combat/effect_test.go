package combat

import "testing"

func TestAddEffectReplacesSameTypeEffect(t *testing.T) {
	combatant := NewCombatant("c1", "Hero", CombatSideParty, 20, 15, 12)

	combatant.AddEffect(StatusEffect{ID: "e1", Type: EffectStunned, Source: "spell", Duration: 2})
	combatant.AddEffect(StatusEffect{ID: "e2", Type: EffectStunned, Source: "trap", Duration: 1})

	if len(combatant.Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(combatant.Effects))
	}
	if combatant.Effects[0].ID != "e2" {
		t.Fatalf("expected effect id e2, got %q", combatant.Effects[0].ID)
	}
}

func TestRemoveEffectRemovesMatchingEffect(t *testing.T) {
	combatant := NewCombatant("c1", "Hero", CombatSideParty, 20, 15, 12)
	combatant.AddEffect(StatusEffect{ID: "e1", Type: EffectProne, Source: "trip", Duration: 1})

	removed := combatant.RemoveEffect("e1")

	if !removed {
		t.Fatal("expected remove effect to return true")
	}
	if len(combatant.Effects) != 0 {
		t.Fatalf("expected no effects, got %d", len(combatant.Effects))
	}
}

func TestHasEffectAndGetEffectReturnExpectedValues(t *testing.T) {
	combatant := NewCombatant("c1", "Hero", CombatSideParty, 20, 15, 12)
	combatant.AddEffect(StatusEffect{ID: "e1", Type: EffectConcentrating, Source: "spell", Duration: 3})

	if !combatant.HasEffect(EffectConcentrating) {
		t.Fatal("expected combatant to have concentrating effect")
	}
	effect, ok := combatant.GetEffect(EffectConcentrating)
	if !ok {
		t.Fatal("expected get effect to succeed")
	}
	if effect.Duration != 3 {
		t.Fatalf("expected duration 3, got %d", effect.Duration)
	}
}

func TestTickEffectsDecrementsDurationAndRemovesExpired(t *testing.T) {
	combatant := NewCombatant("c1", "Hero", CombatSideParty, 20, 15, 12)
	combatant.AddEffect(StatusEffect{ID: "e1", Type: EffectStunned, Source: "spell", Duration: 2})
	combatant.AddEffect(StatusEffect{ID: "e2", Type: EffectProne, Source: "trip", Duration: 1})

	combatant.TickEffects()

	if len(combatant.Effects) != 1 {
		t.Fatalf("expected 1 effect after tick, got %d", len(combatant.Effects))
	}
	if combatant.Effects[0].Type != EffectStunned || combatant.Effects[0].Duration != 1 {
		t.Fatalf("expected stunned duration 1, got %+v", combatant.Effects[0])
	}
}
