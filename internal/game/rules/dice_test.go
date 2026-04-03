package rules

import "testing"

func TestParseDiceExpression(t *testing.T) {
	count, sides, modifier, err := parseDiceExpression("2d6+3")
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}
	if count != 2 || sides != 6 || modifier != 3 {
		t.Fatalf("unexpected parsed result: count=%d sides=%d modifier=%d", count, sides, modifier)
	}
}

func TestRollDiceNormal(t *testing.T) {
	engine := NewDefaultRuleEngine(&fixedRandomizer{values: []int{9}})

	result, err := engine.RollDice(RollDiceInput{
		Expression: "1d20+5",
		Mode:       RollModeNormal,
	})
	if err != nil {
		t.Fatalf("expected roll to succeed, got %v", err)
	}
	if result.Total != 15 {
		t.Fatalf("expected total 15, got %d", result.Total)
	}
	if len(result.Chosen) != 1 || result.Chosen[0] != 10 {
		t.Fatalf("expected chosen roll 10, got %+v", result.Chosen)
	}
}

func TestRollDiceAdvantage(t *testing.T) {
	engine := NewDefaultRuleEngine(&fixedRandomizer{values: []int{3, 17}})

	result, err := engine.RollDice(RollDiceInput{
		Expression: "1d20",
		Mode:       RollModeAdvantage,
	})
	if err != nil {
		t.Fatalf("expected roll to succeed, got %v", err)
	}
	if len(result.Rolls) != 2 {
		t.Fatalf("expected 2 rolls, got %d", len(result.Rolls))
	}
	if len(result.Chosen) != 1 || result.Chosen[0] != 18 {
		t.Fatalf("expected chosen roll 18, got %+v", result.Chosen)
	}
}

func TestRollDiceDisadvantage(t *testing.T) {
	engine := NewDefaultRuleEngine(&fixedRandomizer{values: []int{3, 17}})

	result, err := engine.RollDice(RollDiceInput{
		Expression: "1d20",
		Mode:       RollModeDisadvantage,
	})
	if err != nil {
		t.Fatalf("expected roll to succeed, got %v", err)
	}
	if len(result.Chosen) != 1 || result.Chosen[0] != 4 {
		t.Fatalf("expected chosen roll 4, got %+v", result.Chosen)
	}
}

func TestRollDiceRejectsInvalidExpression(t *testing.T) {
	engine := NewDefaultRuleEngine(&fixedRandomizer{values: []int{1}})

	_, err := engine.RollDice(RollDiceInput{
		Expression: "abc",
		Mode:       RollModeNormal,
	})
	if err == nil {
		t.Fatal("expected invalid expression error")
	}
}

type fixedRandomizer struct {
	values []int
	index  int
}

func (f *fixedRandomizer) Intn(n int) int {
	value := f.values[f.index%len(f.values)]
	f.index++
	return value
}
