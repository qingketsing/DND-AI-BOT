package dice

import (
	"testing"
)

// === 合法表达式解析测试 ===

func TestRoll_BasicFormat(t *testing.T) {
	// 1d20 标准格式
	r, err := Roll("1d20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Details) != 1 {
		t.Errorf("expected 1 die, got %d", len(r.Details))
	}
	if r.Details[0] < 1 || r.Details[0] > 20 {
		t.Errorf("d20 result %d out of range [1,20]", r.Details[0])
	}
	if r.Modifier != 0 {
		t.Errorf("expected modifier 0, got %d", r.Modifier)
	}
}

func TestRoll_ShorthandD20(t *testing.T) {
	// d20 省略数量，等价于 1d20
	r, err := Roll("d20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Details) != 1 {
		t.Errorf("expected 1 die, got %d", len(r.Details))
	}
	if r.Details[0] < 1 || r.Details[0] > 20 {
		t.Errorf("d20 result %d out of range [1,20]", r.Details[0])
	}
}

func TestRoll_MultipleDice(t *testing.T) {
	r, err := Roll("4d6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Details) != 4 {
		t.Errorf("expected 4 dice, got %d", len(r.Details))
	}
	sum := 0
	for _, v := range r.Details {
		if v < 1 || v > 6 {
			t.Errorf("d6 result %d out of range [1,6]", v)
		}
		sum += v
	}
	if r.Total != sum {
		t.Errorf("total %d != sum of details %d", r.Total, sum)
	}
}

func TestRoll_PositiveModifier(t *testing.T) {
	r, err := Roll("1d20+5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Modifier != 5 {
		t.Errorf("expected modifier 5, got %d", r.Modifier)
	}
	if r.Total != r.Details[0]+5 {
		t.Errorf("total %d != roll %d + modifier 5", r.Total, r.Details[0])
	}
}

func TestRoll_NegativeModifier(t *testing.T) {
	r, err := Roll("2d6-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Modifier != -1 {
		t.Errorf("expected modifier -1, got %d", r.Modifier)
	}
	sum := 0
	for _, v := range r.Details {
		sum += v
	}
	if r.Total != sum-1 {
		t.Errorf("total %d != sum %d - 1", r.Total, sum)
	}
}

func TestRoll_ShorthandWithModifier(t *testing.T) {
	// d20+3 省略数量 + 修饰符
	r, err := Roll("d20+3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Details) != 1 {
		t.Errorf("expected 1 die, got %d", len(r.Details))
	}
	if r.Modifier != 3 {
		t.Errorf("expected modifier 3, got %d", r.Modifier)
	}
	if r.Total != r.Details[0]+3 {
		t.Errorf("total mismatch")
	}
}

func TestRoll_CaseInsensitive(t *testing.T) {
	r, err := Roll("1D20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Details) != 1 {
		t.Errorf("expected 1 die, got %d", len(r.Details))
	}
}

func TestRoll_Whitespace(t *testing.T) {
	r, err := Roll("  1d20+2  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Modifier != 2 {
		t.Errorf("expected modifier 2, got %d", r.Modifier)
	}
}

func TestRoll_D6(t *testing.T) {
	r, err := Roll("1d6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Details[0] < 1 || r.Details[0] > 6 {
		t.Errorf("d6 result %d out of range", r.Details[0])
	}
}

// === 非法表达式测试 ===

func TestRoll_InvalidFormat(t *testing.T) {
	invalidExprs := []string{
		"abc",
		"XdY",
		"hello",
		"20",
		"d",
		"dd20",
		"1d20+",
		"1d20+abc",
		"1d20*2",
		"",
	}
	for _, expr := range invalidExprs {
		_, err := Roll(expr)
		if err == nil {
			t.Errorf("expected error for %q, got nil", expr)
		}
	}
}

func TestRoll_TooManyDice(t *testing.T) {
	_, err := Roll("101d6")
	if err == nil {
		t.Error("expected error for 101d6")
	}
}

// === 数值范围统计测试（大量投骰验证分布合理性）===

func TestRoll_RangeCheck(t *testing.T) {
	for i := 0; i < 1000; i++ {
		r, err := Roll("1d6+2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 1d6+2 的范围：3~8
		if r.Total < 3 || r.Total > 8 {
			t.Errorf("1d6+2 result %d out of expected range [3,8]", r.Total)
		}
	}
}

// === String() 输出格式测试 ===

func TestRollResult_String_NoModifier(t *testing.T) {
	r := &RollResult{
		Expression: "1d20",
		Details:    []int{15},
		Total:      15,
		Modifier:   0,
	}
	s := r.String()
	expected := "🎲 1d20: [15] = 15"
	if s != expected {
		t.Errorf("got %q, want %q", s, expected)
	}
}

func TestRollResult_String_PositiveModifier(t *testing.T) {
	r := &RollResult{
		Expression: "1d20+5",
		Details:    []int{12},
		Total:      17,
		Modifier:   5,
	}
	s := r.String()
	expected := "🎲 1d20+5: [12] + 5 = 17"
	if s != expected {
		t.Errorf("got %q, want %q", s, expected)
	}
}

func TestRollResult_String_NegativeModifier(t *testing.T) {
	r := &RollResult{
		Expression: "2d6-1",
		Details:    []int{3, 5},
		Total:      7,
		Modifier:   -1,
	}
	s := r.String()
	expected := "🎲 2d6-1: [3, 5] - 1 = 7"
	if s != expected {
		t.Errorf("got %q, want %q", s, expected)
	}
}

func TestRollResult_String_MultipleDice(t *testing.T) {
	r := &RollResult{
		Expression: "3d6",
		Details:    []int{2, 4, 6},
		Total:      12,
		Modifier:   0,
	}
	s := r.String()
	expected := "🎲 3d6: [2, 4, 6] = 12"
	if s != expected {
		t.Errorf("got %q, want %q", s, expected)
	}
}
