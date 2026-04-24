package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/game/rules"
	"DND-AI-BOT/internal/repository"
)

var (
	// ErrInvalidEncounter 表示创建或更新战斗时传入了不合法的参数。
	ErrInvalidEncounter = errors.New("invalid encounter")
	// ErrEffectNotFound 表示目标单位身上不存在指定效果。
	ErrEffectNotFound = errors.New("effect not found")
)

// EncounterService 负责编排战斗状态的读取与更新流程。
type EncounterService struct {
	repository repository.EncounterRepository
	ruleEngine rules.RuleEngine
}

// EncounterServiceOption 定义战斗服务可选依赖。
type EncounterServiceOption func(*EncounterService)

// WithEncounterRuleEngine 注入战斗服务使用的规则引擎。
func WithEncounterRuleEngine(engine rules.RuleEngine) EncounterServiceOption {
	return func(service *EncounterService) {
		if engine != nil {
			service.ruleEngine = engine
		}
	}
}

// CreateEncounterInput 定义创建战斗时需要的最小输入。
type CreateEncounterInput struct {
	ID         string
	SessionID  string
	Combatants []combat.Combatant
}

// ApplyDamageInput 定义造成伤害所需的输入。
type ApplyDamageInput struct {
	SessionID string
	TargetID  string
	Amount    int
}

// HealInput 定义治疗所需的输入。
type HealInput struct {
	SessionID string
	TargetID  string
	Amount    int
}

// AdvanceTurnInput 定义推进回合所需的输入。
type AdvanceTurnInput struct {
	SessionID string
}

// AddEffectInput 定义增加状态效果所需的输入。
type AddEffectInput struct {
	SessionID string
	TargetID  string
	Effect    combat.StatusEffect
}

// RemoveEffectInput 定义移除状态效果所需的输入。
type RemoveEffectInput struct {
	SessionID string
	TargetID  string
	EffectID  string
}

// CanActInput 定义检查目标单位是否可行动所需的输入。
type CanActInput struct {
	SessionID string
	TargetID  string
}

// ResolveAttackActionInput 定义一次原子化攻击结算所需输入。
type ResolveAttackActionInput struct {
	SessionID   string
	AttackerID  string
	TargetID    string
	AttackBonus int
	DamageDice  string
	DamageBonus int
	AttackMode  rules.RollMode
	AdvanceTurn bool
}

// ResolveAttackActionResult 表示一次攻击结算的结构化结果。
type ResolveAttackActionResult struct {
	EncounterExists bool             `json:"encounter_exists"`
	ActionResolved  bool             `json:"action_resolved"`
	AttackerCanAct  bool             `json:"attacker_can_act"`
	AttackRoll      int              `json:"attack_roll,omitempty"`
	AttackTotal     int              `json:"attack_total,omitempty"`
	TargetAC        int              `json:"target_ac,omitempty"`
	Hit             bool             `json:"hit"`
	DamageRoll      int              `json:"damage_roll,omitempty"`
	DamageTotal     int              `json:"damage_total,omitempty"`
	TargetHP        int              `json:"target_hp,omitempty"`
	TargetMaxHP     int              `json:"target_max_hp,omitempty"`
	TargetDown      bool             `json:"target_down"`
	TargetDead      bool             `json:"target_dead"`
	Round           int              `json:"round,omitempty"`
	TurnIndex       int              `json:"turn_index,omitempty"`
	Message         string           `json:"message,omitempty"`
	Encounter       *combat.Encounter `json:"encounter,omitempty"`
}

// NewEncounterService 创建战斗服务。
func NewEncounterService(repository repository.EncounterRepository, options ...EncounterServiceOption) *EncounterService {
	service := &EncounterService{repository: repository}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

// Create 创建并保存一场新的战斗。
func (s *EncounterService) Create(ctx context.Context, input CreateEncounterInput, now time.Time) (*combat.Encounter, error) {
	if strings.TrimSpace(input.ID) == "" || strings.TrimSpace(input.SessionID) == "" {
		return nil, ErrInvalidEncounter
	}

	encounter := combat.NewEncounter(strings.TrimSpace(input.ID), strings.TrimSpace(input.SessionID), input.Combatants, now)
	if err := s.repository.Save(ctx, encounter); err != nil {
		return nil, err
	}

	return encounter, nil
}

// GetBySessionID 按会话 ID 读取战斗状态。
func (s *EncounterService) GetBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error) {
	return s.repository.LoadBySessionID(ctx, strings.TrimSpace(sessionID))
}

// ApplyDamage 对目标单位造成伤害并保存更新后的战斗状态。
func (s *EncounterService) ApplyDamage(ctx context.Context, input ApplyDamageInput, now time.Time) (*combat.Encounter, error) {
	encounter, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	if err := encounter.ApplyDamage(strings.TrimSpace(input.TargetID), input.Amount, now); err != nil {
		return nil, err
	}
	if err := s.repository.Save(ctx, encounter); err != nil {
		return nil, err
	}

	return encounter, nil
}

// Heal 对目标单位进行治疗并保存更新后的战斗状态。
func (s *EncounterService) Heal(ctx context.Context, input HealInput, now time.Time) (*combat.Encounter, error) {
	encounter, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	if err := encounter.Heal(strings.TrimSpace(input.TargetID), input.Amount, now); err != nil {
		return nil, err
	}
	if err := s.repository.Save(ctx, encounter); err != nil {
		return nil, err
	}

	return encounter, nil
}

// AdvanceTurn 推进回合并保存更新后的战斗状态。
func (s *EncounterService) AdvanceTurn(ctx context.Context, input AdvanceTurnInput, now time.Time) (*combat.Encounter, error) {
	encounter, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	encounter.AdvanceTurn(now)
	if err := s.repository.Save(ctx, encounter); err != nil {
		return nil, err
	}

	return encounter, nil
}

// AddEffect 为目标单位增加状态效果并保存更新后的战斗状态。
func (s *EncounterService) AddEffect(ctx context.Context, input AddEffectInput, now time.Time) (*combat.Encounter, error) {
	encounter, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	target, err := encounter.FindCombatant(strings.TrimSpace(input.TargetID))
	if err != nil {
		return nil, err
	}
	target.AddEffect(input.Effect)
	encounter.UpdatedAt = now
	if err := s.repository.Save(ctx, encounter); err != nil {
		return nil, err
	}

	return encounter, nil
}

// RemoveEffect 从目标单位移除状态效果并保存更新后的战斗状态。
func (s *EncounterService) RemoveEffect(ctx context.Context, input RemoveEffectInput, now time.Time) (*combat.Encounter, error) {
	encounter, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	target, err := encounter.FindCombatant(strings.TrimSpace(input.TargetID))
	if err != nil {
		return nil, err
	}
	if ok := target.RemoveEffect(strings.TrimSpace(input.EffectID)); !ok {
		return nil, ErrEffectNotFound
	}
	encounter.UpdatedAt = now
	if err := s.repository.Save(ctx, encounter); err != nil {
		return nil, err
	}

	return encounter, nil
}

// CanAct 判断目标单位当前是否还能行动。
func (s *EncounterService) CanAct(ctx context.Context, input CanActInput) (bool, error) {
	encounter, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return false, err
	}

	return encounter.CanAct(strings.TrimSpace(input.TargetID))
}

// ResolveAttackAction 以一次领域工具调用完成攻击检定、伤害结算和状态推进。
func (s *EncounterService) ResolveAttackAction(ctx context.Context, input ResolveAttackActionInput, now time.Time) (ResolveAttackActionResult, error) {
	encounter, err := s.repository.LoadBySessionID(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return ResolveAttackActionResult{}, err
	}
	result := ResolveAttackActionResult{
		EncounterExists: true,
		Encounter:       encounter,
		Round:           encounter.Round,
		TurnIndex:       encounter.TurnIndex,
	}
	if s.ruleEngine == nil {
		result.Message = "攻击结算失败：规则引擎不可用。"
		return result, ErrInvalidEncounter
	}

	current, ok := encounter.CurrentCombatant()
	if !ok {
		result.Message = "当前战斗中没有可行动单位。"
		return result, nil
	}
	if current.ID != strings.TrimSpace(input.AttackerID) {
		result.Message = fmt.Sprintf("当前行动单位是 %s，不是请求攻击的单位。", current.Name)
		return result, nil
	}

	canAct, err := encounter.CanAct(strings.TrimSpace(input.AttackerID))
	if err != nil {
		if errors.Is(err, combat.ErrCombatantNotFound) {
			result.Message = "攻击者不存在于当前战斗中。"
			return result, nil
		}
		return result, err
	}
	result.AttackerCanAct = canAct
	if !canAct {
		result.Message = "当前攻击者无法行动。"
		return result, nil
	}

	target, err := encounter.FindCombatant(strings.TrimSpace(input.TargetID))
	if err != nil {
		if errors.Is(err, combat.ErrCombatantNotFound) {
			result.Message = "目标不存在于当前战斗中。"
			return result, nil
		}
		return result, err
	}
	result.TargetAC = target.ArmorClass
	result.TargetHP = target.CurrentHP
	result.TargetMaxHP = target.MaxHP

	attackRoll, err := s.ruleEngine.RollDice(rules.RollDiceInput{
		Expression: fmt.Sprintf("1d20+%d", input.AttackBonus),
		Mode:       input.AttackMode,
	})
	if err != nil {
		return result, err
	}
	result.AttackRoll = sumInts(attackRoll.Chosen)
	result.AttackTotal = attackRoll.Total
	result.Hit = attackRoll.Total >= target.ArmorClass

	if result.Hit {
		damageRoll, err := s.ruleEngine.RollDice(rules.RollDiceInput{
			Expression: appendModifier(input.DamageDice, input.DamageBonus),
			Mode:       rules.RollModeNormal,
		})
		if err != nil {
			return result, err
		}
		result.DamageRoll = sumInts(damageRoll.Chosen)
		result.DamageTotal = damageRoll.Total
		if err := encounter.ApplyDamage(strings.TrimSpace(input.TargetID), damageRoll.Total, now); err != nil {
			return result, err
		}
	}

	if input.AdvanceTurn {
		encounter.AdvanceTurn(now)
	}
	if err := s.repository.Save(ctx, encounter); err != nil {
		return result, err
	}

	target, err = encounter.FindCombatant(strings.TrimSpace(input.TargetID))
	if err != nil {
		return result, err
	}
	result.ActionResolved = true
	result.TargetHP = target.CurrentHP
	result.TargetMaxHP = target.MaxHP
	result.TargetDown = target.Status == combat.CombatStatusDown
	result.TargetDead = target.Status == combat.CombatStatusDead
	result.Round = encounter.Round
	result.TurnIndex = encounter.TurnIndex
	result.Encounter = encounter
	return result, nil
}

func appendModifier(dice string, modifier int) string {
	dice = strings.TrimSpace(dice)
	switch {
	case modifier > 0:
		return fmt.Sprintf("%s+%d", dice, modifier)
	case modifier < 0:
		return fmt.Sprintf("%s%d", dice, modifier)
	default:
		return dice
	}
}

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}
