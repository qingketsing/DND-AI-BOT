package rules

// RollMode 表示掷骰时采用的模式。
type RollMode string

const (
	RollModeNormal       RollMode = "normal"
	RollModeAdvantage    RollMode = "advantage"
	RollModeDisadvantage RollMode = "disadvantage"
)

// RollDiceInput 定义通用掷骰输入。
type RollDiceInput struct {
	Expression string
	Mode       RollMode
}

// RollDiceResult 表示一次掷骰的完整结果。
type RollDiceResult struct {
	Expression string
	Mode       RollMode
	Rolls      []int
	Chosen     []int
	Modifier   int
	Total      int
}

// AbilityCheckInput 定义属性检定输入。
type AbilityCheckInput struct {
	Ability  string
	DC       int
	Modifier int
	Mode     RollMode
}

// SkillCheckInput 定义技能检定输入。
type SkillCheckInput struct {
	Skill    string
	DC       int
	Modifier int
	Mode     RollMode
}

// CheckResult 表示属性或技能检定的结构化结果。
type CheckResult struct {
	Kind     string
	Ability  string
	Skill    string
	DC       int
	Mode     RollMode
	Rolls    []int
	Chosen   int
	Modifier int
	Total    int
	Success  bool
}
