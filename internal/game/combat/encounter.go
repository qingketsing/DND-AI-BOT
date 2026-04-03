package combat

import "time"

// Encounter 表示一场最小可运行的战斗。
type Encounter struct {
	ID         string
	Combatants []Combatant
	Round      int
	TurnIndex  int
	StartedAt  time.Time
	UpdatedAt  time.Time
}

// NewEncounter 创建一场新的战斗。
func NewEncounter(id string, combatants []Combatant, now time.Time) *Encounter {
	return &Encounter{
		ID:         id,
		Combatants: combatants,
		Round:      1,
		TurnIndex:  0,
		StartedAt:  now,
		UpdatedAt:  now,
	}
}

// CurrentCombatant 返回当前行动单位。
func (e *Encounter) CurrentCombatant() (Combatant, bool) {
	if len(e.Combatants) == 0 {
		return Combatant{}, false
	}

	return e.Combatants[e.TurnIndex], true
}

// AdvanceTurn 推进到下一个单位行动。
func (e *Encounter) AdvanceTurn(now time.Time) {
	if len(e.Combatants) == 0 {
		return
	}

	e.TurnIndex++
	if e.TurnIndex >= len(e.Combatants) {
		e.TurnIndex = 0
		e.Round++
	}
	e.UpdatedAt = now
}

// ApplyDamage 对目标单位造成伤害，并在生命值为 0 时设置为倒地。
func (e *Encounter) ApplyDamage(targetID string, amount int, now time.Time) error {
	if amount <= 0 {
		return ErrInvalidDamage
	}

	target, err := e.FindCombatant(targetID)
	if err != nil {
		return err
	}

	target.CurrentHP -= amount
	if target.CurrentHP <= 0 {
		target.CurrentHP = 0
		target.Status = CombatStatusDown
	}
	e.UpdatedAt = now
	return nil
}

// Heal 对目标单位进行治疗，并在恢复生命值后重置为可行动状态。
func (e *Encounter) Heal(targetID string, amount int, now time.Time) error {
	if amount <= 0 {
		return ErrInvalidHeal
	}

	target, err := e.FindCombatant(targetID)
	if err != nil {
		return err
	}

	target.CurrentHP += amount
	if target.CurrentHP > target.MaxHP {
		target.CurrentHP = target.MaxHP
	}
	if target.CurrentHP > 0 && target.Status == CombatStatusDown {
		target.Status = CombatStatusActive
	}
	e.UpdatedAt = now
	return nil
}

// FindCombatant 按 ID 查找战斗单位。
func (e *Encounter) FindCombatant(targetID string) (*Combatant, error) {
	for i := range e.Combatants {
		if e.Combatants[i].ID == targetID {
			return &e.Combatants[i], nil
		}
	}

	return nil, ErrCombatantNotFound
}

// CanAct 判断目标单位当前是否还能行动。
func (e *Encounter) CanAct(targetID string) (bool, error) {
	target, err := e.FindCombatant(targetID)
	if err != nil {
		return false, err
	}

	if target.Status == CombatStatusDown || target.Status == CombatStatusDead {
		return false, nil
	}
	if target.HasEffect(EffectStunned) {
		return false, nil
	}

	return true, nil
}
