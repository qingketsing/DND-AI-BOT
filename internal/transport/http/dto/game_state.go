package dto

import (
	"time"

	"DND-AI-BOT/internal/game/state"
)

// CreateGameStateRequest 定义创建游戏进度接口的请求体。
type CreateGameStateRequest struct {
	Player PlayerStateDTO `json:"player"`
}

// UpdateStatsRequest 定义更新六维属性接口的请求体。
type UpdateStatsRequest struct {
	Stats CharacterStatsDTO `json:"stats"`
}

// AddItemRequest 定义添加背包物品接口的请求体。
type AddItemRequest struct {
	Item InventoryItemDTO `json:"item"`
}

// RemoveItemRequest 定义移除背包物品接口的请求体。
type RemoveItemRequest struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

// GoldRequest 定义金币增减接口的请求体。
type GoldRequest struct {
	Amount int `json:"amount"`
}

// SetSceneRequest 定义更新当前场景接口的请求体。
type SetSceneRequest struct {
	Scene string `json:"scene"`
}

// UpsertQuestRequest 定义新增或更新任务接口的请求体。
type UpsertQuestRequest struct {
	Quest QuestProgressDTO `json:"quest"`
}

// GameStateResponse 定义游戏进度接口的统一响应结构。
type GameStateResponse struct {
	ID           string         `json:"id"`
	SessionID    string         `json:"session_id"`
	Player       PlayerStateDTO `json:"player"`
	CurrentScene string         `json:"current_scene"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// PlayerStateDTO 表示玩家当前状态的传输结构。
type PlayerStateDTO struct {
	Name      string             `json:"name"`
	Level     int                `json:"level"`
	Gold      int                `json:"gold"`
	Stats     CharacterStatsDTO  `json:"stats"`
	Inventory []InventoryItemDTO `json:"inventory"`
	Quests    []QuestProgressDTO `json:"quests"`
}

// CharacterStatsDTO 表示六维属性的传输结构。
type CharacterStatsDTO struct {
	STR int `json:"str"`
	DEX int `json:"dex"`
	CON int `json:"con"`
	INT int `json:"int"`
	WIS int `json:"wis"`
	CHA int `json:"cha"`
}

// InventoryItemDTO 表示背包物品的传输结构。
type InventoryItemDTO struct {
	ID       string `json:"id"`
	ItemID   string `json:"item_id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// QuestProgressDTO 表示任务进度的传输结构。
type QuestProgressDTO struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

// ToGameStateResponse 将领域游戏进度模型转换为 HTTP 响应。
func ToGameStateResponse(gameState *state.GameState) GameStateResponse {
	inventory := make([]InventoryItemDTO, len(gameState.Player.Inventory))
	for i, item := range gameState.Player.Inventory {
		inventory[i] = InventoryItemDTO{
			ID:       item.ID,
			ItemID:   item.ItemID,
			Name:     item.Name,
			Quantity: item.Quantity,
		}
	}

	quests := make([]QuestProgressDTO, len(gameState.Player.Quests))
	for i, quest := range gameState.Player.Quests {
		quests[i] = QuestProgressDTO{
			ID:          quest.ID,
			Title:       quest.Title,
			Status:      string(quest.Status),
			Description: quest.Description,
		}
	}

	return GameStateResponse{
		ID:        gameState.ID,
		SessionID: gameState.SessionID,
		Player: PlayerStateDTO{
			Name:  gameState.Player.Name,
			Level: gameState.Player.Level,
			Gold:  gameState.Player.Gold,
			Stats: CharacterStatsDTO{
				STR: gameState.Player.Stats.STR,
				DEX: gameState.Player.Stats.DEX,
				CON: gameState.Player.Stats.CON,
				INT: gameState.Player.Stats.INT,
				WIS: gameState.Player.Stats.WIS,
				CHA: gameState.Player.Stats.CHA,
			},
			Inventory: inventory,
			Quests:    quests,
		},
		CurrentScene: gameState.CurrentScene,
		CreatedAt:    gameState.CreatedAt,
		UpdatedAt:    gameState.UpdatedAt,
	}
}

// ToPlayerState 将请求 DTO 转换为领域玩家状态。
func ToPlayerState(player PlayerStateDTO) state.PlayerState {
	inventory := make([]state.InventoryItem, len(player.Inventory))
	for i, item := range player.Inventory {
		inventory[i] = state.InventoryItem{
			ID:       item.ID,
			ItemID:   item.ItemID,
			Name:     item.Name,
			Quantity: item.Quantity,
		}
	}

	quests := make([]state.QuestProgress, len(player.Quests))
	for i, quest := range player.Quests {
		quests[i] = ToQuestProgress(quest)
	}

	return state.PlayerState{
		Name:      player.Name,
		Level:     player.Level,
		Gold:      player.Gold,
		Stats:     ToCharacterStats(player.Stats),
		Inventory: inventory,
		Quests:    quests,
	}
}

// ToCharacterStats 将请求 DTO 转换为领域六维属性。
func ToCharacterStats(stats CharacterStatsDTO) state.CharacterStats {
	return state.CharacterStats{
		STR: stats.STR,
		DEX: stats.DEX,
		CON: stats.CON,
		INT: stats.INT,
		WIS: stats.WIS,
		CHA: stats.CHA,
	}
}

// ToInventoryItem 将请求 DTO 转换为领域背包物品。
func ToInventoryItem(item InventoryItemDTO) state.InventoryItem {
	return state.InventoryItem{
		ID:       item.ID,
		ItemID:   item.ItemID,
		Name:     item.Name,
		Quantity: item.Quantity,
	}
}

// ToQuestProgress 将请求 DTO 转换为领域任务进度。
func ToQuestProgress(quest QuestProgressDTO) state.QuestProgress {
	return state.QuestProgress{
		ID:          quest.ID,
		Title:       quest.Title,
		Status:      state.QuestStatus(quest.Status),
		Description: quest.Description,
	}
}
