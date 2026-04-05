package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"DND-AI-BOT/internal/game/combat"
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

// NewEncounterService 创建战斗服务。
func NewEncounterService(repository repository.EncounterRepository) *EncounterService {
	return &EncounterService{repository: repository}
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
