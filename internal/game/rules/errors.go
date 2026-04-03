package rules

import "errors"

var (
	// ErrInvalidDiceExpression 表示骰子表达式格式不合法。
	ErrInvalidDiceExpression = errors.New("invalid dice expression")
	// ErrInvalidRollMode 表示掷骰模式不合法。
	ErrInvalidRollMode = errors.New("invalid roll mode")
	// ErrInvalidAbility 表示属性缩写不合法。
	ErrInvalidAbility = errors.New("invalid ability")
	// ErrInvalidSkill 表示技能名不合法。
	ErrInvalidSkill = errors.New("invalid skill")
	// ErrInvalidDC 表示难度等级不合法。
	ErrInvalidDC = errors.New("invalid difficulty class")
)
