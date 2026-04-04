package state

import (
	"testing"
	"time"
)

func TestNewGameStateInitializesState(t *testing.T) {
	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	player := PlayerState{
		Name:  "Alice",
		Level: 1,
		Gold:  10,
		Stats: CharacterStats{STR: 10, DEX: 12, CON: 11, INT: 13, WIS: 14, CHA: 8},
	}

	gameState := NewGameState("state-1", "session-1", player, now)

	if gameState.ID != "state-1" {
		t.Fatalf("expected state id state-1, got %q", gameState.ID)
	}
	if gameState.SessionID != "session-1" {
		t.Fatalf("expected session id session-1, got %q", gameState.SessionID)
	}
	if gameState.Player.Name != "Alice" {
		t.Fatalf("expected player name Alice, got %q", gameState.Player.Name)
	}
	if !gameState.CreatedAt.Equal(now) || !gameState.UpdatedAt.Equal(now) {
		t.Fatalf("expected timestamps %v, got created=%v updated=%v", now, gameState.CreatedAt, gameState.UpdatedAt)
	}
}

func TestAddItemMergesExistingQuantity(t *testing.T) {
	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	gameState := NewGameState("state-1", "session-1", PlayerState{}, now)

	gameState.AddItem(InventoryItem{ID: "inv-1", ItemID: "potion", Name: "Potion", Quantity: 1}, now.Add(time.Minute))
	gameState.AddItem(InventoryItem{ID: "inv-2", ItemID: "potion", Name: "Potion", Quantity: 2}, now.Add(2*time.Minute))

	item, ok := gameState.FindItem("potion")
	if !ok {
		t.Fatal("expected item to exist")
	}
	if item.Quantity != 3 {
		t.Fatalf("expected quantity 3, got %d", item.Quantity)
	}
}

func TestRemoveItemDecrementsAndRemovesWhenZero(t *testing.T) {
	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	gameState := NewGameState("state-1", "session-1", PlayerState{}, now)
	gameState.AddItem(InventoryItem{ID: "inv-1", ItemID: "potion", Name: "Potion", Quantity: 2}, now.Add(time.Minute))

	if !gameState.RemoveItem("potion", 1, now.Add(2*time.Minute)) {
		t.Fatal("expected remove item to succeed")
	}
	item, ok := gameState.FindItem("potion")
	if !ok || item.Quantity != 1 {
		t.Fatalf("expected quantity 1, got %+v", item)
	}

	if !gameState.RemoveItem("potion", 1, now.Add(3*time.Minute)) {
		t.Fatal("expected remove item to succeed")
	}
	if _, ok := gameState.FindItem("potion"); ok {
		t.Fatal("expected item to be removed when quantity reaches zero")
	}
}

func TestGoldOperationsRespectBalance(t *testing.T) {
	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	gameState := NewGameState("state-1", "session-1", PlayerState{Gold: 10}, now)

	gameState.AddGold(5, now.Add(time.Minute))
	if gameState.Player.Gold != 15 {
		t.Fatalf("expected gold 15, got %d", gameState.Player.Gold)
	}
	if !gameState.SpendGold(12, now.Add(2*time.Minute)) {
		t.Fatal("expected spend gold to succeed")
	}
	if gameState.Player.Gold != 3 {
		t.Fatalf("expected gold 3, got %d", gameState.Player.Gold)
	}
	if gameState.SpendGold(4, now.Add(3*time.Minute)) {
		t.Fatal("expected spend gold to fail when balance is insufficient")
	}
}

func TestUpsertQuestProgressUpdatesExistingQuest(t *testing.T) {
	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	gameState := NewGameState("state-1", "session-1", PlayerState{}, now)

	gameState.UpsertQuestProgress(QuestProgress{ID: "quest-1", Title: "Find Key", Status: QuestStatusActive}, now.Add(time.Minute))
	gameState.UpsertQuestProgress(QuestProgress{ID: "quest-1", Title: "Find Key", Status: QuestStatusCompleted}, now.Add(2*time.Minute))

	quest, ok := gameState.FindQuest("quest-1")
	if !ok {
		t.Fatal("expected quest to exist")
	}
	if quest.Status != QuestStatusCompleted {
		t.Fatalf("expected completed quest, got %q", quest.Status)
	}
}

func TestSetCurrentSceneUpdatesScene(t *testing.T) {
	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	gameState := NewGameState("state-1", "session-1", PlayerState{}, now)

	gameState.SetCurrentScene("tavern", now.Add(time.Minute))

	if gameState.CurrentScene != "tavern" {
		t.Fatalf("expected current scene tavern, got %q", gameState.CurrentScene)
	}
}
