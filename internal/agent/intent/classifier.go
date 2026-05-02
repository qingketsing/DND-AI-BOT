package intent

import "strings"

// Kind 表示用户输入的高层意图类型。
type Kind string

const (
	KindUnknown           Kind = "unknown"
	KindStatusQuery       Kind = "status_query"
	KindSessionRecall     Kind = "session_recall"
	KindCharacterCreation Kind = "character_creation"
	KindCharacterDraft    Kind = "character_draft"
	KindCombatAction      Kind = "combat_action"
	KindExplorationAction Kind = "exploration_action"
	KindRulesQuery        Kind = "rules_query"
	KindLoreQuery         Kind = "lore_query"
	KindNarrative         Kind = "narrative"
)

// Result 表示一次意图分类结果。
type Result struct {
	Kind       Kind
	Confidence float64
	Reason     string
}

// Classifier 定义用户输入意图分类能力。
type Classifier interface {
	Classify(message string) Result
}

// KeywordClassifier 使用确定性关键词规则进行低成本分类。
type KeywordClassifier struct{}

// NewKeywordClassifier 创建关键词分类器。
func NewKeywordClassifier() *KeywordClassifier {
	return &KeywordClassifier{}
}

// Classify 返回用户输入的高层意图。
func (c *KeywordClassifier) Classify(message string) Result {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return Result{Kind: KindUnknown, Reason: "empty message"}
	}

	if containsAny(normalized, "多少血", "血量", "hp", "生命值", "状态", "当前位置", "任务列表", "还剩多少", "还有多少") {
		return matched(KindStatusQuery, "matched status keyword")
	}
	if containsAny(normalized, "说到哪", "说到哪里", "刚才发生", "前面聊", "总结一下当前", "我们在哪") {
		return matched(KindSessionRecall, "matched session recall keyword")
	}
	if containsAny(normalized, "创建角色", "创建一个", "我要扮演", "设定一个角色") {
		return matched(KindCharacterCreation, "matched character creation keyword")
	}
	if containsAny(normalized, "标准数组", "标准点数", "使用这个", "就这个", "名字叫", "职业是", "种族是") {
		return matched(KindCharacterDraft, "matched character draft keyword")
	}
	if containsAny(normalized, "攻击", "施法", "检定", "鉴定", "先攻", "伤害", "击杀", "防御", "闪避", "护盾术", "睡眠术", "冷冻射线") {
		return matched(KindCombatAction, "matched combat keyword")
	}
	if containsAny(normalized, "规则", "dnd", "法术说明", "职业规则", "属性怎么计算", "熟练加值") {
		return matched(KindRulesQuery, "matched rules keyword")
	}
	if containsAny(normalized, "设定", "the city", "世界观", "知识库", "背景", "城市") {
		return matched(KindLoreQuery, "matched lore keyword")
	}
	if containsAny(normalized, "查看", "观察", "聆听", "靠近", "打开门", "前进", "后退", "询问", "移动") {
		return matched(KindExplorationAction, "matched exploration keyword")
	}
	if len([]rune(normalized)) > 80 {
		return matched(KindNarrative, "long free-form message")
	}

	return Result{Kind: KindUnknown, Confidence: 0.2, Reason: "no rule matched"}
}

func matched(kind Kind, reason string) Result {
	return Result{Kind: kind, Confidence: 0.9, Reason: reason}
}

func containsAny(message string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(message, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}
