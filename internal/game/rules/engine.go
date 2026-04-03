package rules

import (
	"math/rand"
	"time"
)

// RuleEngine 定义规则引擎对外暴露的最小能力。
type RuleEngine interface {
	RollDice(input RollDiceInput) (RollDiceResult, error)
	AbilityCheck(input AbilityCheckInput) (CheckResult, error)
	SkillCheck(input SkillCheckInput) (CheckResult, error)
}

// Randomizer 抽象随机数生成器，便于测试控制骰值。
type Randomizer interface {
	Intn(n int) int
}

// DefaultRuleEngine 是最小规则引擎的默认实现。
type DefaultRuleEngine struct {
	rng Randomizer
}

// NewDefaultRuleEngine 创建默认规则引擎。
func NewDefaultRuleEngine(rng Randomizer) *DefaultRuleEngine {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	return &DefaultRuleEngine{rng: rng}
}
