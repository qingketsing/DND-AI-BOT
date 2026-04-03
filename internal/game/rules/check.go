package rules

import "strings"

// AbilityCheck 执行一次基础属性检定。
func (e *DefaultRuleEngine) AbilityCheck(input AbilityCheckInput) (CheckResult, error) {
	ability := normalizeKey(input.Ability)
	if err := validateAbility(ability); err != nil {
		return CheckResult{}, err
	}
	if err := validateDC(input.DC); err != nil {
		return CheckResult{}, err
	}

	rollResult, err := e.RollDice(RollDiceInput{
		Expression: "1d20",
		Mode:       input.Mode,
	})
	if err != nil {
		return CheckResult{}, err
	}

	return buildCheckResult("ability_check", ability, "", input.DC, input.Modifier, input.Mode, rollResult), nil
}

// SkillCheck 执行一次基础技能检定。
func (e *DefaultRuleEngine) SkillCheck(input SkillCheckInput) (CheckResult, error) {
	skill := normalizeKey(input.Skill)
	if err := validateSkill(skill); err != nil {
		return CheckResult{}, err
	}
	if err := validateDC(input.DC); err != nil {
		return CheckResult{}, err
	}

	rollResult, err := e.RollDice(RollDiceInput{
		Expression: "1d20",
		Mode:       input.Mode,
	})
	if err != nil {
		return CheckResult{}, err
	}

	return buildCheckResult("skill_check", SkillAbilityMap[skill], skill, input.DC, input.Modifier, input.Mode, rollResult), nil
}

func validateAbility(ability string) error {
	if _, ok := ValidAbilities[ability]; !ok {
		return ErrInvalidAbility
	}
	return nil
}

func validateSkill(skill string) error {
	if _, ok := SkillAbilityMap[skill]; !ok {
		return ErrInvalidSkill
	}
	return nil
}

func validateDC(dc int) error {
	if dc <= 0 {
		return ErrInvalidDC
	}
	return nil
}

func normalizeKey(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func buildCheckResult(kind string, ability string, skill string, dc int, modifier int, mode RollMode, rollResult RollDiceResult) CheckResult {
	chosen := 0
	if len(rollResult.Chosen) > 0 {
		chosen = rollResult.Chosen[0]
	}

	total := chosen + modifier
	return CheckResult{
		Kind:     kind,
		Ability:  ability,
		Skill:    skill,
		DC:       dc,
		Mode:     mode,
		Rolls:    rollResult.Rolls,
		Chosen:   chosen,
		Modifier: modifier,
		Total:    total,
		Success:  total >= dc,
	}
}
