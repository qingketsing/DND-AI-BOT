package dto

import (
	"time"

	"DND-AI-BOT/internal/game/combat"
)

// CreateEncounterRequest 定义创建战斗接口的请求体。
type CreateEncounterRequest struct {
	Combatants []CombatantDTO `json:"combatants"`
}

// ApplyDamageRequest 定义造成伤害接口的请求体。
type ApplyDamageRequest struct {
	TargetID string `json:"target_id"`
	Amount   int    `json:"amount"`
}

// HealRequest 定义治疗接口的请求体。
type HealRequest struct {
	TargetID string `json:"target_id"`
	Amount   int    `json:"amount"`
}

// AddEffectRequest 定义添加状态效果接口的请求体。
type AddEffectRequest struct {
	TargetID string          `json:"target_id"`
	Effect   StatusEffectDTO `json:"effect"`
}

// RemoveEffectRequest 定义移除状态效果接口的请求体。
type RemoveEffectRequest struct {
	TargetID string `json:"target_id"`
	EffectID string `json:"effect_id"`
}

// CanActRequest 定义检查是否可行动接口的请求体。
type CanActRequest struct {
	TargetID string `json:"target_id"`
}

// CanActResponse 定义行动能力检查接口的响应体。
type CanActResponse struct {
	CanAct bool `json:"can_act"`
}

// EncounterResponse 定义战斗接口的统一响应结构。
type EncounterResponse struct {
	ID         string         `json:"id"`
	SessionID  string         `json:"session_id"`
	Combatants []CombatantDTO `json:"combatants"`
	Round      int            `json:"round"`
	TurnIndex  int            `json:"turn_index"`
	StartedAt  time.Time      `json:"started_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// CombatantDTO 表示战斗单位的传输结构。
type CombatantDTO struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Side       string            `json:"side"`
	CurrentHP  int               `json:"current_hp"`
	MaxHP      int               `json:"max_hp"`
	ArmorClass int               `json:"armor_class"`
	Initiative int               `json:"initiative"`
	Status     string            `json:"status"`
	Effects    []StatusEffectDTO `json:"effects"`
}

// StatusEffectDTO 表示战斗状态效果的传输结构。
type StatusEffectDTO struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Source   string `json:"source"`
	Duration int    `json:"duration"`
}

// ToEncounterResponse 将领域战斗模型转换为 HTTP 响应。
func ToEncounterResponse(encounter *combat.Encounter) EncounterResponse {
	combatants := make([]CombatantDTO, len(encounter.Combatants))
	for i, combatantItem := range encounter.Combatants {
		combatants[i] = ToCombatantDTO(combatantItem)
	}

	return EncounterResponse{
		ID:         encounter.ID,
		SessionID:  encounter.SessionID,
		Combatants: combatants,
		Round:      encounter.Round,
		TurnIndex:  encounter.TurnIndex,
		StartedAt:  encounter.StartedAt,
		UpdatedAt:  encounter.UpdatedAt,
	}
}

// ToCombatantDTO 将领域战斗单位转换为 HTTP 响应项。
func ToCombatantDTO(combatantItem combat.Combatant) CombatantDTO {
	effects := make([]StatusEffectDTO, len(combatantItem.Effects))
	for i, effect := range combatantItem.Effects {
		effects[i] = StatusEffectDTO{
			ID:       effect.ID,
			Type:     string(effect.Type),
			Source:   effect.Source,
			Duration: effect.Duration,
		}
	}

	return CombatantDTO{
		ID:         combatantItem.ID,
		Name:       combatantItem.Name,
		Side:       string(combatantItem.Side),
		CurrentHP:  combatantItem.CurrentHP,
		MaxHP:      combatantItem.MaxHP,
		ArmorClass: combatantItem.ArmorClass,
		Initiative: combatantItem.Initiative,
		Status:     string(combatantItem.Status),
		Effects:    effects,
	}
}

// ToCombatants 将请求 DTO 转换为领域战斗单位列表。
func ToCombatants(combatants []CombatantDTO) []combat.Combatant {
	result := make([]combat.Combatant, len(combatants))
	for i, combatantItem := range combatants {
		effects := make([]combat.StatusEffect, len(combatantItem.Effects))
		for j, effect := range combatantItem.Effects {
			effects[j] = ToStatusEffect(effect)
		}

		result[i] = combat.Combatant{
			ID:         combatantItem.ID,
			Name:       combatantItem.Name,
			Side:       combat.CombatSide(combatantItem.Side),
			CurrentHP:  combatantItem.CurrentHP,
			MaxHP:      combatantItem.MaxHP,
			ArmorClass: combatantItem.ArmorClass,
			Initiative: combatantItem.Initiative,
			Status:     combat.CombatStatus(combatantItem.Status),
			Effects:    effects,
		}
	}

	return result
}

// ToStatusEffect 将请求 DTO 转换为领域状态效果。
func ToStatusEffect(effect StatusEffectDTO) combat.StatusEffect {
	return combat.StatusEffect{
		ID:       effect.ID,
		Type:     combat.EffectType(effect.Type),
		Source:   effect.Source,
		Duration: effect.Duration,
	}
}
