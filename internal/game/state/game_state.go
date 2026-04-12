package state

import "time"

// PlayerState 表示玩家当前可持久化的核心进度。
type PlayerState struct {
	Name              string
	Race              string
	Class             string
	BackgroundSummary string
	Level             int
	Gold              int
	Stats             CharacterStats
	Inventory         []InventoryItem
	Quests            []QuestProgress
}

// GameState 表示一局游戏的当前最小状态。
type GameState struct {
	ID           string
	SessionID    string
	Player       PlayerState
	CurrentScene string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewGameState 创建新的游戏状态。
func NewGameState(id string, sessionID string, player PlayerState, now time.Time) *GameState {
	if player.Inventory == nil {
		player.Inventory = make([]InventoryItem, 0)
	}
	if player.Quests == nil {
		player.Quests = make([]QuestProgress, 0)
	}

	return &GameState{
		ID:        id,
		SessionID: sessionID,
		Player:    player,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddItem 将物品加入背包；同一 ItemID 的数量会累加。
func (g *GameState) AddItem(item InventoryItem, now time.Time) {
	if existing, ok := g.FindItem(item.ItemID); ok {
		existing.Quantity += item.Quantity
		g.UpdatedAt = now
		return
	}

	g.Player.Inventory = append(g.Player.Inventory, item)
	g.UpdatedAt = now
}

// RemoveItem 从背包中移除指定数量的物品。
func (g *GameState) RemoveItem(itemID string, quantity int, now time.Time) bool {
	if quantity <= 0 {
		return false
	}

	for i := range g.Player.Inventory {
		if g.Player.Inventory[i].ItemID != itemID {
			continue
		}
		if g.Player.Inventory[i].Quantity < quantity {
			return false
		}

		g.Player.Inventory[i].Quantity -= quantity
		if g.Player.Inventory[i].Quantity == 0 {
			g.Player.Inventory = append(g.Player.Inventory[:i], g.Player.Inventory[i+1:]...)
		}
		g.UpdatedAt = now
		return true
	}

	return false
}

// AddGold 增加玩家金币数量。
func (g *GameState) AddGold(amount int, now time.Time) {
	g.Player.Gold += amount
	g.UpdatedAt = now
}

// SpendGold 消耗玩家金币；余额不足时返回 false。
func (g *GameState) SpendGold(amount int, now time.Time) bool {
	if amount <= 0 || g.Player.Gold < amount {
		return false
	}

	g.Player.Gold -= amount
	g.UpdatedAt = now
	return true
}

// UpsertQuestProgress 新增或更新任务进度。
func (g *GameState) UpsertQuestProgress(quest QuestProgress, now time.Time) {
	if existing, ok := g.FindQuest(quest.ID); ok {
		*existing = quest
		g.UpdatedAt = now
		return
	}

	g.Player.Quests = append(g.Player.Quests, quest)
	g.UpdatedAt = now
}

// SetCurrentScene 更新当前场景。
func (g *GameState) SetCurrentScene(scene string, now time.Time) {
	g.CurrentScene = scene
	g.UpdatedAt = now
}

// FindItem 按物品模板 ID 查找背包物品。
func (g *GameState) FindItem(itemID string) (*InventoryItem, bool) {
	for i := range g.Player.Inventory {
		if g.Player.Inventory[i].ItemID == itemID {
			return &g.Player.Inventory[i], true
		}
	}

	return nil, false
}

// FindQuest 按任务 ID 查找任务进度。
func (g *GameState) FindQuest(questID string) (*QuestProgress, bool) {
	for i := range g.Player.Quests {
		if g.Player.Quests[i].ID == questID {
			return &g.Player.Quests[i], true
		}
	}

	return nil, false
}
