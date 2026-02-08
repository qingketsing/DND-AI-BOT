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
}

// Roll 简单的骰子解析 (支持 XdY格式)
// 例如: 1d20, 2d6
func Roll(expression string) (*RollResult, error) {
	expression = strings.ToLower(strings.TrimSpace(expression))
	re := regexp.MustCompile(`^(\d*)d(\d+)$`)
	matches := re.FindStringSubmatch(expression)

	if len(matches) != 3 {
		return nil, fmt.Errorf("骰子格式错误，请使用 [数量]d[面数] 的格式 (例如 d20, 1d20, 2d6)")
	}

	count := 1
	if matches[1] != "" {
		count, _ = strconv.Atoi(matches[1])
	}
	sides, _ := strconv.Atoi(matches[2])

	if count > 100 {
		return nil, fmt.Errorf("too many dice")
	}
	if sides < 1 {
		return nil, fmt.Errorf("invalid sides")
	}

	result := &RollResult{
		Expression: expression,
		Details:    make([]int, 0),
	}

	total := 0
	for i := 0; i < count; i++ {
		val := rand.Intn(sides) + 1
		result.Details = append(result.Details, val)
		total += val
	}
	result.Total = total

	return result, nil
}

func (r *RollResult) String() string {
	detailsStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(r.Details)), "+"), "[]")
	if len(r.Details) == 1 {
		return fmt.Sprintf("投掷 %s: %d", r.Expression, r.Total)
	}
	return fmt.Sprintf("投掷 %s: %s = %d", r.Expression, detailsStr, r.Total)
}
