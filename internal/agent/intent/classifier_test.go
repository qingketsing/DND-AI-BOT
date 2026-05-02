package intent

import "testing"

func TestKeywordClassifierClassifiesStatusQuery(t *testing.T) {
	classifier := NewKeywordClassifier()

	result := classifier.Classify("它还有多少血量？")

	if result.Kind != KindStatusQuery {
		t.Fatalf("expected kind %q, got %q", KindStatusQuery, result.Kind)
	}
	if result.Confidence <= 0 {
		t.Fatalf("expected positive confidence, got %f", result.Confidence)
	}
	if result.Reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestKeywordClassifierClassifiesSessionRecall(t *testing.T) {
	classifier := NewKeywordClassifier()

	result := classifier.Classify("我们刚才说到哪里了？")

	if result.Kind != KindSessionRecall {
		t.Fatalf("expected kind %q, got %q", KindSessionRecall, result.Kind)
	}
}

func TestKeywordClassifierClassifiesCharacterDraft(t *testing.T) {
	classifier := NewKeywordClassifier()

	result := classifier.Classify("使用标准数组")

	if result.Kind != KindCharacterDraft {
		t.Fatalf("expected kind %q, got %q", KindCharacterDraft, result.Kind)
	}
}

func TestKeywordClassifierClassifiesCombatAction(t *testing.T) {
	classifier := NewKeywordClassifier()

	result := classifier.Classify("继续攻击并进行近战攻击检定")

	if result.Kind != KindCombatAction {
		t.Fatalf("expected kind %q, got %q", KindCombatAction, result.Kind)
	}
}

func TestKeywordClassifierClassifiesLoreQueryBeforeExploration(t *testing.T) {
	classifier := NewKeywordClassifier()

	result := classifier.Classify("查看一下 the city 设定")

	if result.Kind != KindLoreQuery {
		t.Fatalf("expected kind %q, got %q", KindLoreQuery, result.Kind)
	}
}
