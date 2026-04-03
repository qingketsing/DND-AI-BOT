package rules

import "testing"

func TestAbilityCheckSuccess(t *testing.T) {
	engine := NewDefaultRuleEngine(&fixedRandomizer{values: []int{11}})

	result, err := engine.AbilityCheck(AbilityCheckInput{
		Ability:  "dex",
		DC:       12,
		Modifier: 2,
		Mode:     RollModeNormal,
	})
	if err != nil {
		t.Fatalf("expected check to succeed, got %v", err)
	}
	if !result.Success {
		t.Fatal("expected ability check to succeed")
	}
	if result.Total != 14 {
		t.Fatalf("expected total 14, got %d", result.Total)
	}
}

func TestAbilityCheckFailure(t *testing.T) {
	engine := NewDefaultRuleEngine(&fixedRandomizer{values: []int{4}})

	result, err := engine.AbilityCheck(AbilityCheckInput{
		Ability:  "str",
		DC:       10,
		Modifier: 0,
		Mode:     RollModeNormal,
	})
	if err != nil {
		t.Fatalf("expected check to succeed, got %v", err)
	}
	if result.Success {
		t.Fatal("expected ability check to fail")
	}
}

func TestAbilityCheckRejectsInvalidAbility(t *testing.T) {
	engine := NewDefaultRuleEngine(&fixedRandomizer{values: []int{1}})

	_, err := engine.AbilityCheck(AbilityCheckInput{
		Ability:  "luck",
		DC:       10,
		Modifier: 0,
		Mode:     RollModeNormal,
	})
	if err == nil {
		t.Fatal("expected invalid ability error")
	}
}

func TestSkillCheckMapsSkillToAbility(t *testing.T) {
	engine := NewDefaultRuleEngine(&fixedRandomizer{values: []int{14}})

	result, err := engine.SkillCheck(SkillCheckInput{
		Skill:    "stealth",
		DC:       12,
		Modifier: 3,
		Mode:     RollModeNormal,
	})
	if err != nil {
		t.Fatalf("expected check to succeed, got %v", err)
	}
	if result.Ability != "dex" {
		t.Fatalf("expected mapped ability dex, got %q", result.Ability)
	}
	if !result.Success {
		t.Fatal("expected skill check to succeed")
	}
}

func TestSkillCheckRejectsInvalidSkill(t *testing.T) {
	engine := NewDefaultRuleEngine(&fixedRandomizer{values: []int{1}})

	_, err := engine.SkillCheck(SkillCheckInput{
		Skill:    "cooking",
		DC:       10,
		Modifier: 0,
		Mode:     RollModeNormal,
	})
	if err == nil {
		t.Fatal("expected invalid skill error")
	}
}
