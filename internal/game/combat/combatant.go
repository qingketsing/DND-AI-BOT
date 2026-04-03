package combat

// Combatant 表示可参与战斗的最小单位。
type Combatant struct {
	ID         string
	Name       string
	Side       CombatSide
	CurrentHP  int
	MaxHP      int
	ArmorClass int
	Initiative int
	Status     CombatStatus
	Effects    []StatusEffect
}

// NewCombatant 创建一个带基础生命值和护甲等级的战斗单位。
func NewCombatant(id string, name string, side CombatSide, maxHP int, armorClass int, initiative int) Combatant {
	return Combatant{
		ID:         id,
		Name:       name,
		Side:       side,
		CurrentHP:  maxHP,
		MaxHP:      maxHP,
		ArmorClass: armorClass,
		Initiative: initiative,
		Status:     CombatStatusActive,
		Effects:    make([]StatusEffect, 0),
	}
}

// AddEffect 为单位添加状态效果；同类型效果会被最新效果覆盖。
func (c *Combatant) AddEffect(effect StatusEffect) {
	for i := range c.Effects {
		if c.Effects[i].Type == effect.Type {
			c.Effects[i] = effect
			return
		}
	}

	c.Effects = append(c.Effects, effect)
}

// RemoveEffect 按效果 ID 移除状态效果。
func (c *Combatant) RemoveEffect(effectID string) bool {
	for i := range c.Effects {
		if c.Effects[i].ID == effectID {
			c.Effects = append(c.Effects[:i], c.Effects[i+1:]...)
			return true
		}
	}

	return false
}

// HasEffect 判断单位是否拥有指定类型的状态效果。
func (c *Combatant) HasEffect(effectType EffectType) bool {
	_, ok := c.GetEffect(effectType)
	return ok
}

// GetEffect 获取指定类型的状态效果。
func (c *Combatant) GetEffect(effectType EffectType) (StatusEffect, bool) {
	for _, effect := range c.Effects {
		if effect.Type == effectType {
			return effect, true
		}
	}

	return StatusEffect{}, false
}

// TickEffects 推进单位身上所有状态效果的持续时间，并移除已到期效果。
func (c *Combatant) TickEffects() {
	kept := make([]StatusEffect, 0, len(c.Effects))
	for _, effect := range c.Effects {
		if effect.Duration > 0 {
			effect.Duration--
		}
		if effect.Duration != 0 {
			kept = append(kept, effect)
		}
	}

	c.Effects = kept
}
