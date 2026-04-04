package state

import "testing"

func TestGetItemDefinitionReturnsStoredDefinition(t *testing.T) {
	store := NewInMemoryItemDefinitionStore([]ItemDefinition{
		{
			ItemID: "healing-potion",
			Name:   "Healing Potion",
			Effects: []ItemEffect{
				{Type: ItemEffectHeal, Target: "self", Value: 10},
			},
		},
	})

	definition, err := store.GetItemDefinition("healing-potion")
	if err != nil {
		t.Fatalf("expected get item definition to succeed, got %v", err)
	}
	if definition.Name != "Healing Potion" {
		t.Fatalf("expected item name Healing Potion, got %q", definition.Name)
	}
}

func TestGetItemEffectsReturnsEffectsForDefinition(t *testing.T) {
	store := NewInMemoryItemDefinitionStore([]ItemDefinition{
		{
			ItemID: "bomb",
			Name:   "Bomb",
			Effects: []ItemEffect{
				{Type: ItemEffectDamage, Target: "enemy", Value: 8},
				{Type: ItemEffectUtility, Target: "area", Key: "explosive"},
			},
		},
	})

	effects, err := store.GetItemEffects("bomb")
	if err != nil {
		t.Fatalf("expected get item effects to succeed, got %v", err)
	}
	if len(effects) != 2 {
		t.Fatalf("expected 2 effects, got %d", len(effects))
	}
	if effects[0].Type != ItemEffectDamage {
		t.Fatalf("expected first effect type %q, got %q", ItemEffectDamage, effects[0].Type)
	}
}

func TestGetItemDefinitionReturnsErrorWhenMissing(t *testing.T) {
	store := NewInMemoryItemDefinitionStore(nil)

	_, err := store.GetItemDefinition("missing")
	if err == nil {
		t.Fatal("expected missing item definition error")
	}
}
