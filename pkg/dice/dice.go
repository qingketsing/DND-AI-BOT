package dice

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RollResult 投掷结果
type RollResult struct {
	Expression string
	Total      int
	Details    []int
	Modifier   int
}

// Roll 简单的骰子解析 (支持 XdY格式)
// 例如: 1d20, 2d6
func Roll(expression string) (*RollResult, error) {
	expression = strings.ToLower(strings.TrimSpace(expression))
	re := regexp.MustCompile(`^(\d*)d(\d+)([+-]\d+)?$`)
	matches := re.FindStringSubmatch(expression)

	if matches == nil {
		return nil, fmt.Errorf("骰子格式错误，请使用 [数量]d[面数][+/-修饰符] 的格式 (例如 d20, 1d20+5, 2d6-1)")
	}

	count := 1
	if matches[1] != "" {
		var err error
		count, err = strconv.Atoi(matches[1])
		if err != nil || count <= 0 {
			return nil, fmt.Errorf("骰子数量必须为正整数")
		}
	}
	// 解析面数
	sides, err := strconv.Atoi(matches[2])
	if err != nil || sides <= 0 {
		return nil, fmt.Errorf("骰子面数必须为正整数")
	}

	if count > 100 {
		return nil, fmt.Errorf("too many dice")
	}

	modifier := 0
	if matches[3] != "" {
		modifier, err = strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("无效的修饰符")
		}
	}
	rolls := make([]int, count)
	total := 0
	for i := 0; i < count; i++ {
		val := rand.Intn(sides) + 1
		rolls[i] = val
		total += val
	}
	total += modifier

	return &RollResult{
		Expression: expression,
		Total:      total,
		Details:    rolls,
		Modifier:   modifier,
	}, nil

	// result := &RollResult{
	// 	Expression: expression,
	// 	Details:    make([]int, 0),
	// }

	// total := 0
	// for i := 0; i < count; i++ {
	// 	val := rand.Intn(sides) + 1
	// 	result.Details = append(result.Details, val)
	// 	total += val
	// }
	// result.Total = total

	// return result, nil
}

func (r *RollResult) String() string {
	rollStrs := make([]string, len(r.Details))
	for i, v := range r.Details {
		rollStrs[i] = strconv.Itoa(v)
	}

	result := fmt.Sprintf("🎲 %s: [%s]", r.Expression, strings.Join(rollStrs, ", "))

	if r.Modifier != 0 {
		if r.Modifier > 0 {
			result += fmt.Sprintf(" + %d", r.Modifier)
		} else {
			result += fmt.Sprintf(" - %d", -r.Modifier)
		}
	}

	result += fmt.Sprintf(" = %d", r.Total)
	return result
}
