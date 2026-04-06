package tools

import (
	"context"
	"time"

	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/service"
)

type encounterToolService interface {
	GetBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error)
	ApplyDamage(ctx context.Context, input service.ApplyDamageInput, now time.Time) (*combat.Encounter, error)
	Heal(ctx context.Context, input service.HealInput, now time.Time) (*combat.Encounter, error)
	AdvanceTurn(ctx context.Context, input service.AdvanceTurnInput, now time.Time) (*combat.Encounter, error)
	AddEffect(ctx context.Context, input service.AddEffectInput, now time.Time) (*combat.Encounter, error)
	RemoveEffect(ctx context.Context, input service.RemoveEffectInput, now time.Time) (*combat.Encounter, error)
	CanAct(ctx context.Context, input service.CanActInput) (bool, error)
}

type applyDamageArgs struct {
	TargetID string `json:"target_id"`
	Amount   int    `json:"amount"`
}

type healArgs struct {
	TargetID string `json:"target_id"`
	Amount   int    `json:"amount"`
}

type addEffectArgs struct {
	TargetID string `json:"target_id"`
	EffectID string `json:"effect_id"`
	Type     string `json:"type"`
	Source   string `json:"source"`
	Duration int    `json:"duration"`
}

type removeEffectArgs struct {
	TargetID string `json:"target_id"`
	EffectID string `json:"effect_id"`
}

type canActArgs struct {
	TargetID string `json:"target_id"`
}

// CanActResult 表示检查行动能力工具的结构化结果。
type CanActResult struct {
	CanAct bool `json:"can_act"`
}

// GetEncounterTool 用于读取当前战斗状态。
type GetEncounterTool struct{ service encounterToolService }

// NewGetEncounterTool 创建战斗状态读取工具。
func NewGetEncounterTool(service encounterToolService) *GetEncounterTool {
	return &GetEncounterTool{service: service}
}

// Spec 返回战斗状态读取工具的元信息描述。
func (t *GetEncounterTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "get_encounter",
		Description: "读取当前会话关联的战斗状态",
		InputSchema: objectSchema(map[string]any{}),
	}
}

// Call 读取当前战斗状态。
func (t *GetEncounterTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	encounter, err := t.service.GetBySessionID(ctx, input.SessionID)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, encounter), nil
}

// ApplyDamageTool 用于对目标单位造成伤害。
type ApplyDamageTool struct{ service encounterToolService }

// NewApplyDamageTool 创建造成伤害工具。
func NewApplyDamageTool(service encounterToolService) *ApplyDamageTool {
	return &ApplyDamageTool{service: service}
}

// Spec 返回造成伤害工具的元信息描述。
func (t *ApplyDamageTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "apply_damage",
		Description: "对当前战斗中的目标单位造成伤害",
		InputSchema: objectSchema(map[string]any{
			"target_id": map[string]any{"type": "string"},
			"amount":    map[string]any{"type": "integer"},
		}, "target_id", "amount"),
	}
}

// Call 解析参数并对目标单位造成伤害。
func (t *ApplyDamageTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args applyDamageArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	encounter, err := t.service.ApplyDamage(ctx, service.ApplyDamageInput{
		SessionID: input.SessionID,
		TargetID:  args.TargetID,
		Amount:    args.Amount,
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, encounter), nil
}

// HealTool 用于对目标单位进行治疗。
type HealTool struct{ service encounterToolService }

// NewHealTool 创建治疗工具。
func NewHealTool(service encounterToolService) *HealTool { return &HealTool{service: service} }

// Spec 返回治疗工具的元信息描述。
func (t *HealTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "heal",
		Description: "对当前战斗中的目标单位进行治疗",
		InputSchema: objectSchema(map[string]any{
			"target_id": map[string]any{"type": "string"},
			"amount":    map[string]any{"type": "integer"},
		}, "target_id", "amount"),
	}
}

// Call 解析参数并治疗目标单位。
func (t *HealTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args healArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	encounter, err := t.service.Heal(ctx, service.HealInput{
		SessionID: input.SessionID,
		TargetID:  args.TargetID,
		Amount:    args.Amount,
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, encounter), nil
}

// AdvanceTurnTool 用于推进当前战斗回合。
type AdvanceTurnTool struct{ service encounterToolService }

// NewAdvanceTurnTool 创建回合推进工具。
func NewAdvanceTurnTool(service encounterToolService) *AdvanceTurnTool {
	return &AdvanceTurnTool{service: service}
}

// Spec 返回回合推进工具的元信息描述。
func (t *AdvanceTurnTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "advance_turn",
		Description: "推进当前战斗到下一名单位行动",
		InputSchema: objectSchema(map[string]any{}),
	}
}

// Call 推进当前战斗回合。
func (t *AdvanceTurnTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	encounter, err := t.service.AdvanceTurn(ctx, service.AdvanceTurnInput{
		SessionID: input.SessionID,
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, encounter), nil
}

// AddEffectTool 用于为目标单位附加状态效果。
type AddEffectTool struct{ service encounterToolService }

// NewAddEffectTool 创建状态效果添加工具。
func NewAddEffectTool(service encounterToolService) *AddEffectTool {
	return &AddEffectTool{service: service}
}

// Spec 返回状态效果添加工具的元信息描述。
func (t *AddEffectTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "add_effect",
		Description: "为当前战斗中的目标单位添加状态效果",
		InputSchema: objectSchema(map[string]any{
			"target_id": map[string]any{"type": "string"},
			"effect_id": map[string]any{"type": "string"},
			"type":      map[string]any{"type": "string"},
			"source":    map[string]any{"type": "string"},
			"duration":  map[string]any{"type": "integer"},
		}, "target_id", "effect_id", "type", "source", "duration"),
	}
}

// Call 解析参数并附加状态效果。
func (t *AddEffectTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args addEffectArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	encounter, err := t.service.AddEffect(ctx, service.AddEffectInput{
		SessionID: input.SessionID,
		TargetID:  args.TargetID,
		Effect: combat.StatusEffect{
			ID:       args.EffectID,
			Type:     combat.EffectType(args.Type),
			Source:   args.Source,
			Duration: args.Duration,
		},
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, encounter), nil
}

// RemoveEffectTool 用于移除目标单位的状态效果。
type RemoveEffectTool struct{ service encounterToolService }

// NewRemoveEffectTool 创建状态效果移除工具。
func NewRemoveEffectTool(service encounterToolService) *RemoveEffectTool {
	return &RemoveEffectTool{service: service}
}

// Spec 返回状态效果移除工具的元信息描述。
func (t *RemoveEffectTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "remove_effect",
		Description: "移除当前战斗中目标单位的指定状态效果",
		InputSchema: objectSchema(map[string]any{
			"target_id": map[string]any{"type": "string"},
			"effect_id": map[string]any{"type": "string"},
		}, "target_id", "effect_id"),
	}
}

// Call 解析参数并移除状态效果。
func (t *RemoveEffectTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args removeEffectArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	encounter, err := t.service.RemoveEffect(ctx, service.RemoveEffectInput{
		SessionID: input.SessionID,
		TargetID:  args.TargetID,
		EffectID:  args.EffectID,
	}, input.Now)
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, encounter), nil
}

// CanActTool 用于判断目标单位是否可行动。
type CanActTool struct{ service encounterToolService }

// NewCanActTool 创建行动能力检查工具。
func NewCanActTool(service encounterToolService) *CanActTool { return &CanActTool{service: service} }

// Spec 返回行动能力检查工具的元信息描述。
func (t *CanActTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "can_act",
		Description: "判断当前战斗中的目标单位是否还能行动",
		InputSchema: objectSchema(map[string]any{
			"target_id": map[string]any{"type": "string"},
		}, "target_id"),
	}
}

// Call 解析参数并返回目标单位是否可行动。
func (t *CanActTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args canActArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	canAct, err := t.service.CanAct(ctx, service.CanActInput{
		SessionID: input.SessionID,
		TargetID:  args.TargetID,
	})
	if err != nil {
		return CallOutput{}, err
	}
	return newToolOutput(t.Spec().Name, CanActResult{CanAct: canAct}), nil
}
