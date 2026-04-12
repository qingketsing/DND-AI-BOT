package tools

import (
	"context"
	"errors"
	"time"

	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/service"
)

type gameStateToolService interface {
	GetBySessionID(ctx context.Context, sessionID string) (*state.GameState, error)
	CreateCharacter(ctx context.Context, input service.CreateCharacterInput, now time.Time) (*state.GameState, error)
	UpdateStats(ctx context.Context, input service.UpdateStatsInput, now time.Time) (*state.GameState, error)
	AddItem(ctx context.Context, input service.AddItemInput, now time.Time) (*state.GameState, error)
	RemoveItem(ctx context.Context, input service.RemoveItemInput, now time.Time) (*state.GameState, error)
	AddGold(ctx context.Context, input service.AddGoldInput, now time.Time) (*state.GameState, error)
	SpendGold(ctx context.Context, input service.SpendGoldInput, now time.Time) (*state.GameState, error)
	SetScene(ctx context.Context, input service.SetSceneInput, now time.Time) (*state.GameState, error)
	UpsertQuest(ctx context.Context, input service.UpsertQuestInput, now time.Time) (*state.GameState, error)
}

type updateStatsArgs struct {
	STR int `json:"str"`
	DEX int `json:"dex"`
	CON int `json:"con"`
	INT int `json:"int"`
	WIS int `json:"wis"`
	CHA int `json:"cha"`
}

type addItemArgs struct {
	ID       string `json:"id"`
	ItemID   string `json:"item_id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type removeItemArgs struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

type addGoldArgs struct {
	Amount int `json:"amount"`
}

type spendGoldArgs struct {
	Amount int `json:"amount"`
}

type setSceneArgs struct {
	Scene string `json:"scene"`
}

type createCharacterArgs struct {
	Name              string          `json:"name"`
	Race              string          `json:"race"`
	Class             string          `json:"class"`
	BackgroundSummary string          `json:"background_summary"`
	Level             int             `json:"level"`
	Stats             updateStatsArgs `json:"stats"`
	Inventory         []addItemArgs   `json:"inventory"`
	Scene             string          `json:"scene"`
}

type upsertQuestArgs struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

// GetGameStateTool 用于读取当前游戏进度。
type GetGameStateTool struct{ service gameStateToolService }

// NewGetGameStateTool 创建读取游戏进度工具。
func NewGetGameStateTool(service gameStateToolService) *GetGameStateTool {
	return &GetGameStateTool{service: service}
}

// Spec 返回读取游戏进度工具的元信息描述。
func (t *GetGameStateTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "get_game_state",
		Description: "读取当前会话关联的游戏进度",
		InputSchema: objectSchema(map[string]any{}),
	}
}

// Call 读取并返回当前游戏进度。
func (t *GetGameStateTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	gameState, err := t.service.GetBySessionID(ctx, input.SessionID)
	if err != nil {
		if !errors.Is(err, repository.ErrGameStateNotFound) {
			return CallOutput{}, err
		}

		gameState = state.NewGameState("state-"+input.SessionID, input.SessionID, state.PlayerState{}, input.Now)
	}
	return newToolOutput(t.Spec().Name, gameState), nil
}

// CreateCharacterTool 用于一次性初始化角色和起始场景。
type CreateCharacterTool struct{ service gameStateToolService }

// NewCreateCharacterTool 创建角色初始化工具。
func NewCreateCharacterTool(service gameStateToolService) *CreateCharacterTool {
	return &CreateCharacterTool{service: service}
}

// Spec 返回角色初始化工具的元信息描述。
func (t *CreateCharacterTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "create_character",
		Description: "初始化当前会话的角色身份、属性、初始装备和起始场景",
		InputSchema: objectSchema(map[string]any{
			"name":               map[string]any{"type": "string"},
			"race":               map[string]any{"type": "string"},
			"class":              map[string]any{"type": "string"},
			"background_summary": map[string]any{"type": "string"},
			"level":              map[string]any{"type": "integer"},
			"stats":              map[string]any{"type": "object"},
			"inventory":          map[string]any{"type": "array"},
			"scene":              map[string]any{"type": "string"},
		}, "name", "class", "level", "stats"),
	}
}

// Call 解析角色参数并初始化当前会话的角色状态。
func (t *CreateCharacterTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args createCharacterArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}

	inventory := make([]state.InventoryItem, 0, len(args.Inventory))
	for _, item := range args.Inventory {
		inventory = append(inventory, state.InventoryItem{
			ID:       item.ID,
			ItemID:   item.ItemID,
			Name:     item.Name,
			Quantity: item.Quantity,
		})
	}

	gameState, err := t.service.CreateCharacter(ctx, service.CreateCharacterInput{
		SessionID:         input.SessionID,
		Name:              args.Name,
		Race:              args.Race,
		Class:             args.Class,
		BackgroundSummary: args.BackgroundSummary,
		Level:             args.Level,
		Stats: state.CharacterStats{
			STR: args.Stats.STR,
			DEX: args.Stats.DEX,
			CON: args.Stats.CON,
			INT: args.Stats.INT,
			WIS: args.Stats.WIS,
			CHA: args.Stats.CHA,
		},
		Inventory: inventory,
		Scene:     args.Scene,
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, gameState), nil
}

// UpdateStatsTool 用于整体更新六维属性。
type UpdateStatsTool struct{ service gameStateToolService }

// NewUpdateStatsTool 创建六维属性更新工具。
func NewUpdateStatsTool(service gameStateToolService) *UpdateStatsTool {
	return &UpdateStatsTool{service: service}
}

// Spec 返回六维属性更新工具的元信息描述。
func (t *UpdateStatsTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "update_stats",
		Description: "整体更新当前角色的六维属性",
		InputSchema: objectSchema(map[string]any{
			"str": map[string]any{"type": "integer"},
			"dex": map[string]any{"type": "integer"},
			"con": map[string]any{"type": "integer"},
			"int": map[string]any{"type": "integer"},
			"wis": map[string]any{"type": "integer"},
			"cha": map[string]any{"type": "integer"},
		}, "str", "dex", "con", "int", "wis", "cha"),
	}
}

// Call 解析六维参数并更新当前角色属性。
func (t *UpdateStatsTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args updateStatsArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}

	gameState, err := t.service.UpdateStats(ctx, service.UpdateStatsInput{
		SessionID: input.SessionID,
		Stats: state.CharacterStats{
			STR: args.STR,
			DEX: args.DEX,
			CON: args.CON,
			INT: args.INT,
			WIS: args.WIS,
			CHA: args.CHA,
		},
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, gameState), nil
}

// AddItemTool 用于向背包添加物品。
type AddItemTool struct{ service gameStateToolService }

// NewAddItemTool 创建添加物品工具。
func NewAddItemTool(service gameStateToolService) *AddItemTool { return &AddItemTool{service: service} }

// Spec 返回添加物品工具的元信息描述。
func (t *AddItemTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "add_item",
		Description: "向当前角色背包中添加一项物品",
		InputSchema: objectSchema(map[string]any{
			"id":       map[string]any{"type": "string"},
			"item_id":  map[string]any{"type": "string"},
			"name":     map[string]any{"type": "string"},
			"quantity": map[string]any{"type": "integer"},
		}, "id", "item_id", "name", "quantity"),
	}
}

// Call 解析物品参数并向背包添加物品。
func (t *AddItemTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args addItemArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	gameState, err := t.service.AddItem(ctx, service.AddItemInput{
		SessionID: input.SessionID,
		Item: state.InventoryItem{
			ID:       args.ID,
			ItemID:   args.ItemID,
			Name:     args.Name,
			Quantity: args.Quantity,
		},
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, gameState), nil
}

// RemoveItemTool 用于从背包移除物品。
type RemoveItemTool struct{ service gameStateToolService }

// NewRemoveItemTool 创建移除物品工具。
func NewRemoveItemTool(service gameStateToolService) *RemoveItemTool {
	return &RemoveItemTool{service: service}
}

// Spec 返回移除物品工具的元信息描述。
func (t *RemoveItemTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "remove_item",
		Description: "从当前角色背包中移除指定数量的物品",
		InputSchema: objectSchema(map[string]any{
			"item_id":  map[string]any{"type": "string"},
			"quantity": map[string]any{"type": "integer"},
		}, "item_id", "quantity"),
	}
}

// Call 解析参数并从背包中移除物品。
func (t *RemoveItemTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args removeItemArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	gameState, err := t.service.RemoveItem(ctx, service.RemoveItemInput{
		SessionID: input.SessionID,
		ItemID:    args.ItemID,
		Quantity:  args.Quantity,
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, gameState), nil
}

// AddGoldTool 用于增加金币。
type AddGoldTool struct{ service gameStateToolService }

// NewAddGoldTool 创建增加金币工具。
func NewAddGoldTool(service gameStateToolService) *AddGoldTool { return &AddGoldTool{service: service} }

// Spec 返回增加金币工具的元信息描述。
func (t *AddGoldTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "add_gold",
		Description: "增加当前角色的金币数量",
		InputSchema: objectSchema(map[string]any{"amount": map[string]any{"type": "integer"}}, "amount"),
	}
}

// Call 解析参数并增加金币。
func (t *AddGoldTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args addGoldArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	gameState, err := t.service.AddGold(ctx, service.AddGoldInput{
		SessionID: input.SessionID,
		Amount:    args.Amount,
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, gameState), nil
}

// SpendGoldTool 用于消耗金币。
type SpendGoldTool struct{ service gameStateToolService }

// NewSpendGoldTool 创建消耗金币工具。
func NewSpendGoldTool(service gameStateToolService) *SpendGoldTool {
	return &SpendGoldTool{service: service}
}

// Spec 返回消耗金币工具的元信息描述。
func (t *SpendGoldTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "spend_gold",
		Description: "消耗当前角色的金币数量",
		InputSchema: objectSchema(map[string]any{"amount": map[string]any{"type": "integer"}}, "amount"),
	}
}

// Call 解析参数并消耗金币。
func (t *SpendGoldTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args spendGoldArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	gameState, err := t.service.SpendGold(ctx, service.SpendGoldInput{
		SessionID: input.SessionID,
		Amount:    args.Amount,
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, gameState), nil
}

// SetSceneTool 用于更新当前场景。
type SetSceneTool struct{ service gameStateToolService }

// NewSetSceneTool 创建场景更新工具。
func NewSetSceneTool(service gameStateToolService) *SetSceneTool {
	return &SetSceneTool{service: service}
}

// Spec 返回场景更新工具的元信息描述。
func (t *SetSceneTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "set_scene",
		Description: "更新当前游戏所处场景",
		InputSchema: objectSchema(map[string]any{"scene": map[string]any{"type": "string"}}, "scene"),
	}
}

// Call 解析参数并更新场景。
func (t *SetSceneTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args setSceneArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	gameState, err := t.service.SetScene(ctx, service.SetSceneInput{
		SessionID: input.SessionID,
		Scene:     args.Scene,
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, gameState), nil
}

// UpsertQuestTool 用于新增或更新任务进度。
type UpsertQuestTool struct{ service gameStateToolService }

// NewUpsertQuestTool 创建任务更新工具。
func NewUpsertQuestTool(service gameStateToolService) *UpsertQuestTool {
	return &UpsertQuestTool{service: service}
}

// Spec 返回任务更新工具的元信息描述。
func (t *UpsertQuestTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "upsert_quest",
		Description: "新增或更新当前角色的任务进度",
		InputSchema: objectSchema(map[string]any{
			"id":          map[string]any{"type": "string"},
			"title":       map[string]any{"type": "string"},
			"status":      map[string]any{"type": "string"},
			"description": map[string]any{"type": "string"},
		}, "id", "title", "status", "description"),
	}
}

// Call 解析参数并写入任务进度。
func (t *UpsertQuestTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args upsertQuestArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	gameState, err := t.service.UpsertQuest(ctx, service.UpsertQuestInput{
		SessionID: input.SessionID,
		Quest: state.QuestProgress{
			ID:          args.ID,
			Title:       args.Title,
			Status:      state.QuestStatus(args.Status),
			Description: args.Description,
		},
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, gameState), nil
}
