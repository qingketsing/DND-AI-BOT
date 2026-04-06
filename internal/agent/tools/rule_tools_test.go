package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"DND-AI-BOT/internal/game/rules"
)

func TestRollDiceToolCallMapsArgs(t *testing.T) {
	engine := &fakeRuleEngine{
		rollDiceResult: rules.RollDiceResult{Expression: "1d20", Total: 14},
	}
	tool := NewRollDiceTool(engine)

	output, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"expression":"1d20","mode":"normal"}`),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if engine.rollDiceInput.Expression != "1d20" || engine.rollDiceInput.Mode != rules.RollModeNormal {
		t.Fatalf("expected roll dice input to be mapped, got %+v", engine.rollDiceInput)
	}
	result, ok := output.Content.(rules.RollDiceResult)
	if !ok || result.Total != 14 {
		t.Fatalf("expected roll dice result, got %#v", output.Content)
	}
}

func TestAbilityCheckToolCallMapsArgs(t *testing.T) {
	engine := &fakeRuleEngine{
		checkResult: rules.CheckResult{Kind: "ability_check", Success: true},
	}
	tool := NewAbilityCheckTool(engine)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"ability":"str","dc":15,"modifier":2,"mode":"advantage"}`),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if engine.abilityCheckInput.Ability != "str" || engine.abilityCheckInput.Mode != rules.RollModeAdvantage {
		t.Fatalf("expected ability check input to be mapped, got %+v", engine.abilityCheckInput)
	}
}

func TestSkillCheckToolCallMapsArgs(t *testing.T) {
	engine := &fakeRuleEngine{
		checkResult: rules.CheckResult{Kind: "skill_check", Success: true},
	}
	tool := NewSkillCheckTool(engine)

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"skill":"stealth","dc":15,"modifier":4,"mode":"disadvantage"}`),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if engine.skillCheckInput.Skill != "stealth" || engine.skillCheckInput.Mode != rules.RollModeDisadvantage {
		t.Fatalf("expected skill check input to be mapped, got %+v", engine.skillCheckInput)
	}
}

func TestRuleToolsRejectInvalidInput(t *testing.T) {
	tool := NewSkillCheckTool(&fakeRuleEngine{})

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"dc":"bad"}`),
	})
	if !errors.Is(err, ErrInvalidToolInput) {
		t.Fatalf("expected ErrInvalidToolInput, got %v", err)
	}
}

type fakeRuleEngine struct {
	rollDiceInput     rules.RollDiceInput
	abilityCheckInput rules.AbilityCheckInput
	skillCheckInput   rules.SkillCheckInput
	rollDiceResult    rules.RollDiceResult
	checkResult       rules.CheckResult
	err               error
}

func (f *fakeRuleEngine) RollDice(input rules.RollDiceInput) (rules.RollDiceResult, error) {
	f.rollDiceInput = input
	return f.rollDiceResult, f.err
}

func (f *fakeRuleEngine) AbilityCheck(input rules.AbilityCheckInput) (rules.CheckResult, error) {
	f.abilityCheckInput = input
	return f.checkResult, f.err
}

func (f *fakeRuleEngine) SkillCheck(input rules.SkillCheckInput) (rules.CheckResult, error) {
	f.skillCheckInput = input
	return f.checkResult, f.err
}
