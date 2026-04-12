package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/repository"
)

var (
	// ErrInvalidGameState 表示创建或更新游戏进度时传入了不合法的参数。
	ErrInvalidGameState = errors.New("invalid game state")
	// ErrInsufficientGold 表示当前金币不足以完成扣减操作。
	ErrInsufficientGold = errors.New("insufficient gold")
	// ErrInsufficientItemQuantity 表示当前物品数量不足以完成扣减操作。
	ErrInsufficientItemQuantity = errors.New("insufficient item quantity")
)

// GameStateService 负责编排游戏进度的读取与更新流程。
type GameStateService struct {
	repository repository.GameStateRepository
}

// CreateGameStateInput 定义创建游戏进度时需要的最小输入。
type CreateGameStateInput struct {
	ID        string
	SessionID string
	Player    state.PlayerState
}

// CreateCharacterInput 定义一次性初始化角色与场景所需的输入。
type CreateCharacterInput struct {
	SessionID         string
	Name              string
	Race              string
	Class             string
	BackgroundSummary string
	Level             int
	Stats             state.CharacterStats
	Inventory         []state.InventoryItem
	Scene             string
}

// UpdateStatsInput 定义整体更新六维属性所需的输入。
type UpdateStatsInput struct {
	SessionID string
	Stats     state.CharacterStats
}

// AddItemInput 定义向背包添加物品所需的输入。
type AddItemInput struct {
	SessionID string
	Item      state.InventoryItem
}

// RemoveItemInput 定义从背包移除物品所需的输入。
type RemoveItemInput struct {
	SessionID string
	ItemID    string
	Quantity  int
}

// AddGoldInput 定义增加金币所需的输入。
type AddGoldInput struct {
	SessionID string
	Amount    int
}

// SpendGoldInput 定义消耗金币所需的输入。
type SpendGoldInput struct {
	SessionID string
	Amount    int
}

// SetSceneInput 定义更新当前场景所需的输入。
type SetSceneInput struct {
	SessionID string
	Scene     string
}

// UpsertQuestInput 定义新增或更新任务进度所需的输入。
type UpsertQuestInput struct {
	SessionID string
	Quest     state.QuestProgress
}

// NewGameStateService 创建游戏进度服务。
func NewGameStateService(repository repository.GameStateRepository) *GameStateService {
	return &GameStateService{repository: repository}
}

// Create 创建并保存一份新的游戏进度。
func (s *GameStateService) Create(ctx context.Context, input CreateGameStateInput, now time.Time) (*state.GameState, error) {
	if strings.TrimSpace(input.ID) == "" || strings.TrimSpace(input.SessionID) == "" {
		return nil, ErrInvalidGameState
	}

	gameState := state.NewGameState(strings.TrimSpace(input.ID), strings.TrimSpace(input.SessionID), input.Player, now)
	if err := s.repository.Save(ctx, gameState); err != nil {
		return nil, err
	}

	return gameState, nil
}

// GetBySessionID 按会话 ID 读取游戏进度。
func (s *GameStateService) GetBySessionID(ctx context.Context, sessionID string) (*state.GameState, error) {
	return s.repository.LoadBySessionID(ctx, strings.TrimSpace(sessionID))
}

// CreateCharacter 一次性初始化当前会话的角色和起始场景；若状态不存在则自动创建。
func (s *GameStateService) CreateCharacter(ctx context.Context, input CreateCharacterInput, now time.Time) (*state.GameState, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Class) == "" {
		return nil, ErrInvalidGameState
	}

	gameState, err := s.repository.LoadBySessionID(ctx, sessionID)
	if err != nil {
		if !errors.Is(err, repository.ErrGameStateNotFound) {
			return nil, err
		}

		gameState = state.NewGameState("state-"+sessionID, sessionID, state.PlayerState{}, now)
	}

	gameState.Player = state.PlayerState{
		Name:              strings.TrimSpace(input.Name),
		Race:              strings.TrimSpace(input.Race),
		Class:             strings.TrimSpace(input.Class),
		BackgroundSummary: strings.TrimSpace(input.BackgroundSummary),
		Level:             input.Level,
		Gold:              gameState.Player.Gold,
		Stats:             input.Stats,
		Inventory:         append([]state.InventoryItem(nil), input.Inventory...),
		Quests:            append([]state.QuestProgress(nil), gameState.Player.Quests...),
	}
	gameState.SetCurrentScene(strings.TrimSpace(input.Scene), now)
	gameState.UpdatedAt = now
	if err := s.repository.Save(ctx, gameState); err != nil {
		return nil, err
	}

	return gameState, nil
}

// UpdateStats 整体替换玩家当前六维属性。
func (s *GameStateService) UpdateStats(ctx context.Context, input UpdateStatsInput, now time.Time) (*state.GameState, error) {
	gameState, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	gameState.Player.Stats = input.Stats
	gameState.UpdatedAt = now
	if err := s.repository.Save(ctx, gameState); err != nil {
		return nil, err
	}

	return gameState, nil
}

// AddItem 向背包添加指定物品。
func (s *GameStateService) AddItem(ctx context.Context, input AddItemInput, now time.Time) (*state.GameState, error) {
	gameState, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	gameState.AddItem(input.Item, now)
	if err := s.repository.Save(ctx, gameState); err != nil {
		return nil, err
	}

	return gameState, nil
}

// RemoveItem 从背包移除指定数量的物品。
func (s *GameStateService) RemoveItem(ctx context.Context, input RemoveItemInput, now time.Time) (*state.GameState, error) {
	gameState, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	if ok := gameState.RemoveItem(strings.TrimSpace(input.ItemID), input.Quantity, now); !ok {
		return nil, ErrInsufficientItemQuantity
	}
	if err := s.repository.Save(ctx, gameState); err != nil {
		return nil, err
	}

	return gameState, nil
}

// AddGold 增加玩家当前金币数量。
func (s *GameStateService) AddGold(ctx context.Context, input AddGoldInput, now time.Time) (*state.GameState, error) {
	gameState, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	gameState.AddGold(input.Amount, now)
	if err := s.repository.Save(ctx, gameState); err != nil {
		return nil, err
	}

	return gameState, nil
}

// SpendGold 消耗玩家当前金币数量。
func (s *GameStateService) SpendGold(ctx context.Context, input SpendGoldInput, now time.Time) (*state.GameState, error) {
	gameState, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	if ok := gameState.SpendGold(input.Amount, now); !ok {
		return nil, ErrInsufficientGold
	}
	if err := s.repository.Save(ctx, gameState); err != nil {
		return nil, err
	}

	return gameState, nil
}

// SetScene 更新当前场景。
func (s *GameStateService) SetScene(ctx context.Context, input SetSceneInput, now time.Time) (*state.GameState, error) {
	gameState, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	gameState.SetCurrentScene(strings.TrimSpace(input.Scene), now)
	if err := s.repository.Save(ctx, gameState); err != nil {
		return nil, err
	}

	return gameState, nil
}

// UpsertQuest 新增或更新任务进度。
func (s *GameStateService) UpsertQuest(ctx context.Context, input UpsertQuestInput, now time.Time) (*state.GameState, error) {
	gameState, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	gameState.UpsertQuestProgress(input.Quest, now)
	if err := s.repository.Save(ctx, gameState); err != nil {
		return nil, err
	}

	return gameState, nil
}
