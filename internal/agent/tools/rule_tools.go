package tools

import (
	"context"

	"DND-AI-BOT/internal/game/rules"
)

type ruleToolEngine interface {
	RollDice(input rules.RollDiceInput) (rules.RollDiceResult, error)
	AbilityCheck(input rules.AbilityCheckInput) (rules.CheckResult, error)
	SkillCheck(input rules.SkillCheckInput) (rules.CheckResult, error)
}

type rollDiceArgs struct {
	Expression string `json:"expression"`
	Mode       string `json:"mode"`
}

type abilityCheckArgs struct {
	Ability  string `json:"ability"`
	DC       int    `json:"dc"`
	Modifier int    `json:"modifier"`
	Mode     string `json:"mode"`
}

type skillCheckArgs struct {
	Skill    string `json:"skill"`
	DC       int    `json:"dc"`
	Modifier int    `json:"modifier"`
	Mode     string `json:"mode"`
}

// RollDiceTool 用于执行通用掷骰。
type RollDiceTool struct{ engine ruleToolEngine }

// NewRollDiceTool 创建通用掷骰工具。
func NewRollDiceTool(engine ruleToolEngine) *RollDiceTool { return &RollDiceTool{engine: engine} }

// Spec 返回通用掷骰工具的元信息描述。
func (t *RollDiceTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "roll_dice",
		Description: "执行一次通用掷骰表达式计算",
		InputSchema: objectSchema(map[string]any{
			"expression": map[string]any{"type": "string"},
			"mode":       map[string]any{"type": "string"},
		}, "expression"),
	}
}

// Call 解析参数并执行通用掷骰。
func (t *RollDiceTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	_ = ctx
	var args rollDiceArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	result, err := t.engine.RollDice(rules.RollDiceInput{
		Expression: args.Expression,
		Mode:       parseRollMode(args.Mode),
	})
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, result), nil
}

// AbilityCheckTool 用于执行属性检定。
type AbilityCheckTool struct{ engine ruleToolEngine }

// NewAbilityCheckTool 创建属性检定工具。
func NewAbilityCheckTool(engine ruleToolEngine) *AbilityCheckTool {
	return &AbilityCheckTool{engine: engine}
}

// Spec 返回属性检定工具的元信息描述。
func (t *AbilityCheckTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "ability_check",
		Description: "执行一次属性检定",
		InputSchema: objectSchema(map[string]any{
			"ability":  map[string]any{"type": "string"},
			"dc":       map[string]any{"type": "integer"},
			"modifier": map[string]any{"type": "integer"},
			"mode":     map[string]any{"type": "string"},
		}, "ability", "dc", "modifier"),
	}
}

// Call 解析参数并执行属性检定。
func (t *AbilityCheckTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	_ = ctx
	var args abilityCheckArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	result, err := t.engine.AbilityCheck(rules.AbilityCheckInput{
		Ability:  args.Ability,
		DC:       args.DC,
		Modifier: args.Modifier,
		Mode:     parseRollMode(args.Mode),
	})
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, result), nil
}

// SkillCheckTool 用于执行技能检定。
type SkillCheckTool struct{ engine ruleToolEngine }

// NewSkillCheckTool 创建技能检定工具。
func NewSkillCheckTool(engine ruleToolEngine) *SkillCheckTool { return &SkillCheckTool{engine: engine} }

// Spec 返回技能检定工具的元信息描述。
func (t *SkillCheckTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "skill_check",
		Description: "执行一次技能检定",
		InputSchema: objectSchema(map[string]any{
			"skill":    map[string]any{"type": "string"},
			"dc":       map[string]any{"type": "integer"},
			"modifier": map[string]any{"type": "integer"},
			"mode":     map[string]any{"type": "string"},
		}, "skill", "dc", "modifier"),
	}
}

// Call 解析参数并执行技能检定。
func (t *SkillCheckTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	_ = ctx
	var args skillCheckArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	result, err := t.engine.SkillCheck(rules.SkillCheckInput{
		Skill:    args.Skill,
		DC:       args.DC,
		Modifier: args.Modifier,
		Mode:     parseRollMode(args.Mode),
	})
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, result), nil
}
