package rules

import (
	"strconv"
	"strings"
)

// RollDice 执行一次通用掷骰。
func (e *DefaultRuleEngine) RollDice(input RollDiceInput) (RollDiceResult, error) {
	if err := validateRollMode(input.Mode); err != nil {
		return RollDiceResult{}, err
	}

	count, sides, modifier, err := parseDiceExpression(input.Expression)
	if err != nil {
		return RollDiceResult{}, err
	}

	rollCount := count
	if sides == 20 && count == 1 && (input.Mode == RollModeAdvantage || input.Mode == RollModeDisadvantage) {
		rollCount = 2
	}

	rolls := rollMany(e.rng, rollCount, sides)
	chosen, err := applyRollMode(rolls, count, sides, input.Mode)
	if err != nil {
		return RollDiceResult{}, err
	}

	return RollDiceResult{
		Expression: input.Expression,
		Mode:       input.Mode,
		Rolls:      rolls,
		Chosen:     chosen,
		Modifier:   modifier,
		Total:      sumInts(chosen) + modifier,
	}, nil
}

func parseDiceExpression(expr string) (count int, sides int, modifier int, err error) {
	expr = strings.TrimSpace(strings.ToLower(expr))
	parts := strings.SplitN(expr, "d", 2)
	if len(parts) != 2 {
		return 0, 0, 0, ErrInvalidDiceExpression
	}

	count, err = strconv.Atoi(parts[0])
	if err != nil || count < 1 {
		return 0, 0, 0, ErrInvalidDiceExpression
	}

	sidesPart := parts[1]
	modifierIndex := strings.IndexAny(sidesPart, "+-")
	if modifierIndex == -1 {
		sides, err = strconv.Atoi(sidesPart)
		if err != nil || sides < 2 {
			return 0, 0, 0, ErrInvalidDiceExpression
		}
		return count, sides, 0, nil
	}

	sides, err = strconv.Atoi(sidesPart[:modifierIndex])
	if err != nil || sides < 2 {
		return 0, 0, 0, ErrInvalidDiceExpression
	}

	modifier, err = strconv.Atoi(sidesPart[modifierIndex:])
	if err != nil {
		return 0, 0, 0, ErrInvalidDiceExpression
	}

	return count, sides, modifier, nil
}

func validateRollMode(mode RollMode) error {
	switch mode {
	case RollModeNormal, RollModeAdvantage, RollModeDisadvantage:
		return nil
	default:
		return ErrInvalidRollMode
	}
}

func rollMany(rng Randomizer, count int, sides int) []int {
	rolls := make([]int, count)
	for i := 0; i < count; i++ {
		rolls[i] = rng.Intn(sides) + 1
	}

	return rolls
}

func applyRollMode(rolls []int, count int, sides int, mode RollMode) ([]int, error) {
	switch mode {
	case RollModeNormal:
		chosen := make([]int, len(rolls))
		copy(chosen, rolls)
		return chosen, nil
	case RollModeAdvantage:
		if count != 1 || sides != 20 || len(rolls) != 2 {
			return nil, ErrInvalidRollMode
		}
		return []int{maxInt(rolls[0], rolls[1])}, nil
	case RollModeDisadvantage:
		if count != 1 || sides != 20 || len(rolls) != 2 {
			return nil, ErrInvalidRollMode
		}
		return []int{minInt(rolls[0], rolls[1])}, nil
	default:
		return nil, ErrInvalidRollMode
	}
}

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}

	return total
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
