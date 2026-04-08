package bootstrap

import "testing"

func TestBuildSearchRuntimeBuildsDefaultSearchers(t *testing.T) {
	deps, err := BuildSearchRuntime()
	if err != nil {
		t.Fatalf("expected search runtime build to succeed, got %v", err)
	}
	if deps == nil {
		t.Fatal("expected search runtime dependencies to be created")
	}
	if deps.RuleSearcher == nil {
		t.Fatal("expected rule searcher to be created")
	}
	if deps.LoreSearcher == nil {
		t.Fatal("expected lore searcher to be created")
	}
}
