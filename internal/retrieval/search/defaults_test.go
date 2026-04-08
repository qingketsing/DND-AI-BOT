package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveProjectRootPrefersConfiguredRoot(t *testing.T) {
	t.Setenv("DND_AI_BOT_ROOT", "/tmp/dnd-root")

	got := resolveProjectRoot("/does/not/matter")
	want := "/tmp/dnd-root"
	if got != want {
		t.Fatalf("expected configured project root %q, got %q", want, got)
	}
}

func TestResolveProjectRootFindsGoModInAncestor(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "internal", "retrieval", "search")
	t.Setenv("DND_AI_BOT_ROOT", "")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("expected temp directory tree to be created, got %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("expected temp go.mod to be created, got %v", err)
	}

	got := resolveProjectRoot(subdir)
	if got != root {
		t.Fatalf("expected discovered project root %q, got %q", root, got)
	}
}

func TestBuildChunksPathUsesResolvedProjectRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("expected temp go.mod to be created, got %v", err)
	}

	got := buildChunksPath(root, KnowledgeBaseRules)
	want := filepath.Join(root, "data", "chunks", "rules", "chunks.jsonl")
	if got != want {
		t.Fatalf("expected chunks path %q, got %q", want, got)
	}
}
