package dto

import (
	"testing"
	"time"

	"DND-AI-BOT/internal/game/state"
)

func TestToGameStateResponseMapsDomainModel(t *testing.T) {
	now := time.Date(2026, 4, 5, 18, 0, 0, 0, time.UTC)
	gameState := state.NewGameState("state-1", "session-1", state.PlayerState{
		Name:  "Alice",
		Level: 2,
		Gold:  15,
		Stats: state.CharacterStats{STR: 10, DEX: 12, CON: 11, INT: 13, WIS: 14, CHA: 8},
		Inventory: []state.InventoryItem{
			{ID: "inv-1", ItemID: "potion", Name: "Potion", Quantity: 2},
		},
		Quests: []state.QuestProgress{
			{ID: "quest-1", Title: "Find Key", Status: state.QuestStatusActive, Description: "Find the silver key"},
		},
	}, now)
	gameState.SetCurrentScene("forest", now)

	response := ToGameStateResponse(gameState)

	if response.ID != "state-1" || response.SessionID != "session-1" {
		t.Fatalf("expected ids to be mapped, got %+v", response)
	}
	if response.Player.Name != "Alice" || response.Player.Gold != 15 {
		t.Fatalf("expected player fields to be mapped, got %+v", response.Player)
	}
	if len(response.Player.Inventory) != 1 || response.Player.Inventory[0].ItemID != "potion" {
		t.Fatalf("expected inventory to be mapped, got %+v", response.Player.Inventory)
	}
	if len(response.Player.Quests) != 1 || response.Player.Quests[0].Status != "active" {
		t.Fatalf("expected quests to be mapped, got %+v", response.Player.Quests)
	}
	if response.CurrentScene != "forest" {
		t.Fatalf("expected current scene %q, got %q", "forest", response.CurrentScene)
	}
}

func TestToPlayerStateMapsDTOToDomainModel(t *testing.T) {
	player := ToPlayerState(PlayerStateDTO{
		Name:  "Alice",
		Level: 3,
		Gold:  25,
		Stats: CharacterStatsDTO{STR: 15, DEX: 14, CON: 13, INT: 12, WIS: 11, CHA: 10},
		Inventory: []InventoryItemDTO{
			{ID: "inv-1", ItemID: "rope", Name: "Rope", Quantity: 1},
		},
		Quests: []QuestProgressDTO{
			{ID: "quest-1", Title: "Escort", Status: "completed", Description: "Escort the merchant"},
		},
	})

	if player.Name != "Alice" || player.Level != 3 || player.Gold != 25 {
		t.Fatalf("expected player basics to be mapped, got %+v", player)
	}
	if player.Stats.STR != 15 || player.Stats.CHA != 10 {
		t.Fatalf("expected stats to be mapped, got %+v", player.Stats)
	}
	if len(player.Inventory) != 1 || player.Inventory[0].ItemID != "rope" {
		t.Fatalf("expected inventory to be mapped, got %+v", player.Inventory)
	}
	if len(player.Quests) != 1 || player.Quests[0].Status != state.QuestStatusCompleted {
		t.Fatalf("expected quests to be mapped, got %+v", player.Quests)
	}
}

func TestToCharacterStatsAndHelpersMapFields(t *testing.T) {
	stats := ToCharacterStats(CharacterStatsDTO{STR: 9, DEX: 8, CON: 7, INT: 6, WIS: 5, CHA: 4})
	item := ToInventoryItem(InventoryItemDTO{ID: "inv-1", ItemID: "bomb", Name: "Bomb", Quantity: 3})
	quest := ToQuestProgress(QuestProgressDTO{ID: "quest-1", Title: "Scout", Status: "failed", Description: "Scout the ruins"})

	if stats.STR != 9 || stats.CHA != 4 {
		t.Fatalf("expected stats helper to map fields, got %+v", stats)
	}
	if item.ItemID != "bomb" || item.Quantity != 3 {
		t.Fatalf("expected item helper to map fields, got %+v", item)
	}
	if quest.Status != state.QuestStatusFailed || quest.Title != "Scout" {
		t.Fatalf("expected quest helper to map fields, got %+v", quest)
	}
}
