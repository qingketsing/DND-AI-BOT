package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/service"
)

func TestGetGameStateToolCallUsesSessionID(t *testing.T) {
	svc := &fakeGameStateToolService{
		result: newToolGameState(),
	}
	tool := NewGetGameStateTool(svc)

	output, err := tool.Call(context.Background(), CallInput{SessionID: "session-1"})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.getSessionID != "session-1" {
		t.Fatalf("expected session id %q, got %q", "session-1", svc.getSessionID)
	}
	if output.ToolName != "get_game_state" {
		t.Fatalf("expected tool name %q, got %q", "get_game_state", output.ToolName)
	}
}

func TestUpdateStatsToolCallMapsArgs(t *testing.T) {
	svc := &fakeGameStateToolService{result: newToolGameState()}
	tool := NewUpdateStatsTool(svc)
	now := time.Date(2026, 4, 6, 14, 10, 0, 0, time.UTC)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"str":15,"dex":14,"con":13,"int":12,"wis":11,"cha":10}`),
		Now:       now,
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.updateStatsInput.SessionID != "session-1" || svc.updateStatsInput.Stats.STR != 15 || !svc.updateStatsNow.Equal(now) {
		t.Fatalf("expected update stats input to be mapped, got %+v at %v", svc.updateStatsInput, svc.updateStatsNow)
	}
}

func TestAddItemToolCallMapsArgs(t *testing.T) {
	svc := &fakeGameStateToolService{result: newToolGameState()}
	tool := NewAddItemTool(svc)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"id":"inv-1","item_id":"potion","name":"Potion","quantity":2}`),
		Now:       time.Date(2026, 4, 6, 14, 20, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.addItemInput.Item.ItemID != "potion" || svc.addItemInput.Item.Quantity != 2 {
		t.Fatalf("expected add item input to be mapped, got %+v", svc.addItemInput)
	}
}

func TestRemoveItemToolCallMapsArgs(t *testing.T) {
	svc := &fakeGameStateToolService{result: newToolGameState()}
	tool := NewRemoveItemTool(svc)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"item_id":"potion","quantity":1}`),
		Now:       time.Date(2026, 4, 6, 14, 25, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.removeItemInput.ItemID != "potion" || svc.removeItemInput.Quantity != 1 {
		t.Fatalf("expected remove item input to be mapped, got %+v", svc.removeItemInput)
	}
}

func TestAddGoldToolCallMapsArgs(t *testing.T) {
	svc := &fakeGameStateToolService{result: newToolGameState()}
	tool := NewAddGoldTool(svc)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"amount":8}`),
		Now:       time.Date(2026, 4, 6, 14, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.addGoldInput.Amount != 8 {
		t.Fatalf("expected add gold amount 8, got %+v", svc.addGoldInput)
	}
}

func TestSpendGoldToolCallMapsArgs(t *testing.T) {
	svc := &fakeGameStateToolService{result: newToolGameState()}
	tool := NewSpendGoldTool(svc)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"amount":3}`),
		Now:       time.Date(2026, 4, 6, 14, 35, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.spendGoldInput.Amount != 3 {
		t.Fatalf("expected spend gold amount 3, got %+v", svc.spendGoldInput)
	}
}

func TestSetSceneToolCallMapsArgs(t *testing.T) {
	svc := &fakeGameStateToolService{result: newToolGameState()}
	tool := NewSetSceneTool(svc)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"scene":"forest"}`),
		Now:       time.Date(2026, 4, 6, 14, 40, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.setSceneInput.Scene != "forest" {
		t.Fatalf("expected set scene input to be mapped, got %+v", svc.setSceneInput)
	}
}

func TestUpsertQuestToolCallMapsArgs(t *testing.T) {
	svc := &fakeGameStateToolService{result: newToolGameState()}
	tool := NewUpsertQuestTool(svc)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"id":"quest-1","title":"Find Key","status":"active","description":"Find the hidden key"}`),
		Now:       time.Date(2026, 4, 6, 14, 45, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if svc.upsertQuestInput.Quest.ID != "quest-1" || svc.upsertQuestInput.Quest.Status != state.QuestStatusActive {
		t.Fatalf("expected upsert quest input to be mapped, got %+v", svc.upsertQuestInput)
	}
}

func TestGameStateToolsRejectInvalidInput(t *testing.T) {
	tool := NewAddGoldTool(&fakeGameStateToolService{})

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"amount":"bad"}`),
	})
	if !errors.Is(err, ErrInvalidToolInput) {
		t.Fatalf("expected ErrInvalidToolInput, got %v", err)
	}
}

type fakeGameStateToolService struct {
	result           *state.GameState
	err              error
	getSessionID     string
	updateStatsInput service.UpdateStatsInput
	updateStatsNow   time.Time
	addItemInput     service.AddItemInput
	addItemNow       time.Time
	removeItemInput  service.RemoveItemInput
	removeItemNow    time.Time
	addGoldInput     service.AddGoldInput
	addGoldNow       time.Time
	spendGoldInput   service.SpendGoldInput
	spendGoldNow     time.Time
	setSceneInput    service.SetSceneInput
	setSceneNow      time.Time
	upsertQuestInput service.UpsertQuestInput
	upsertQuestNow   time.Time
}

func newToolGameState() *state.GameState {
	return state.NewGameState("state-1", "session-1", state.PlayerState{
		Name:  "Alice",
		Level: 1,
		Gold:  10,
		Stats: state.CharacterStats{STR: 10, DEX: 12, CON: 11, INT: 13, WIS: 14, CHA: 8},
	}, time.Date(2026, 4, 6, 14, 0, 0, 0, time.UTC))
}

func (f *fakeGameStateToolService) Create(ctx context.Context, input service.CreateGameStateInput, now time.Time) (*state.GameState, error) {
	_ = ctx
	_ = input
	_ = now
	return f.result, f.err
}

func (f *fakeGameStateToolService) GetBySessionID(ctx context.Context, sessionID string) (*state.GameState, error) {
	_ = ctx
	f.getSessionID = sessionID
	return f.result, f.err
}

func (f *fakeGameStateToolService) UpdateStats(ctx context.Context, input service.UpdateStatsInput, now time.Time) (*state.GameState, error) {
	_ = ctx
	f.updateStatsInput = input
	f.updateStatsNow = now
	return f.result, f.err
}

func (f *fakeGameStateToolService) AddItem(ctx context.Context, input service.AddItemInput, now time.Time) (*state.GameState, error) {
	_ = ctx
	f.addItemInput = input
	f.addItemNow = now
	return f.result, f.err
}

func (f *fakeGameStateToolService) RemoveItem(ctx context.Context, input service.RemoveItemInput, now time.Time) (*state.GameState, error) {
	_ = ctx
	f.removeItemInput = input
	f.removeItemNow = now
	return f.result, f.err
}

func (f *fakeGameStateToolService) AddGold(ctx context.Context, input service.AddGoldInput, now time.Time) (*state.GameState, error) {
	_ = ctx
	f.addGoldInput = input
	f.addGoldNow = now
	return f.result, f.err
}

func (f *fakeGameStateToolService) SpendGold(ctx context.Context, input service.SpendGoldInput, now time.Time) (*state.GameState, error) {
	_ = ctx
	f.spendGoldInput = input
	f.spendGoldNow = now
	return f.result, f.err
}

func (f *fakeGameStateToolService) SetScene(ctx context.Context, input service.SetSceneInput, now time.Time) (*state.GameState, error) {
	_ = ctx
	f.setSceneInput = input
	f.setSceneNow = now
	return f.result, f.err
}

func (f *fakeGameStateToolService) UpsertQuest(ctx context.Context, input service.UpsertQuestInput, now time.Time) (*state.GameState, error) {
	_ = ctx
	f.upsertQuestInput = input
	f.upsertQuestNow = now
	return f.result, f.err
}
