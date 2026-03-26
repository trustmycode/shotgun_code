package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestNormalizeRelativePath(t *testing.T) {
	if got := normalizeRelativePath(" ./frontend/src "); got != "frontend/src" {
		t.Fatalf("unexpected normalized path: %s", got)
	}
	if got := normalizeRelativePath("."); got != "" {
		t.Fatalf("expected empty for '.', got %q", got)
	}
}

func TestNormalizeCandidateForRoot(t *testing.T) {
	root := "/tmp/shotgun_code"
	if got := normalizeCandidateForRoot(root, "shotgun_code/frontend/main.go"); got != "frontend/main.go" {
		t.Fatalf("unexpected candidate normalization: %s", got)
	}
}

func TestResolveLLMSelection(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "b", "f1.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "f2.txt"), []byte("y"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	selected, err := resolveLLMSelection(root, []string{"a"})
	if err != nil {
		t.Fatalf("resolve selection failed: %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("expected 2 files selected from dir, got %d (%v)", len(selected), selected)
	}
}

func TestBuildAutoContextTreeHonorsExcludedMap(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "keep.go"), []byte("package main"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "skip.go"), []byte("package main"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	tree, err := buildAutoContextTree(root, map[string]bool{"src/skip.go": true})
	if err != nil {
		t.Fatalf("build auto context tree failed: %v", err)
	}
	if tree == "" {
		t.Fatal("expected non-empty tree")
	}
	if contains := filepath.Base(root) + string(os.PathSeparator); len(tree) < len(contains) {
		t.Fatalf("unexpected short tree output: %q", tree)
	}
	if match := "skip.go"; strings.Contains(tree, match) {
		t.Fatalf("expected excluded file %s not to appear in tree: %q", match, tree)
	}
}

func TestParseAutoContextJSONWithMarkdownFence(t *testing.T) {
	result, err := parseAutoContextJSON("```json\n{\"files\":[\"./src/main.go\"],\"reasoning\":\"needed\"}\n```")
	if err != nil {
		t.Fatalf("parseAutoContextJSON failed: %v", err)
	}
	if len(result.Files) != 1 || result.Files[0] != "src/main.go" {
		t.Fatalf("unexpected parsed files: %+v", result.Files)
	}
}

func TestParseAutoContextJSONRejectsUnknownFields(t *testing.T) {
	_, err := parseAutoContextJSON("{\"files\":[\"a.go\"],\"extra\":true}")
	if err == nil {
		t.Fatal("expected parse error for unknown field")
	}
}

func TestAutoContextServiceBuildPrompt(t *testing.T) {
	svc := NewAutoContextService()
	prompt, err := svc.BuildPrompt("repo/\n└── main.go\n", "fix bug", "current notes")
	if err != nil {
		t.Fatalf("BuildPrompt failed: %v", err)
	}
	if !strings.Contains(prompt, "fix bug") {
		t.Fatalf("expected task in prompt, got: %q", prompt)
	}
	if !strings.Contains(prompt, "\"files\"") {
		t.Fatalf("expected format instructions in prompt, got: %q", prompt)
	}
}

func TestBuildAutoContextTreeTooLarge(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 2200; i++ {
		name := filepath.Join(root, "dir", "file_"+strings.Repeat("a", 20)+strconv.Itoa(i)+".txt")
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}
	_, err := buildAutoContextTree(root, map[string]bool{})
	if err == nil {
		t.Fatal("expected size error")
	}
	if !errors.Is(err, errAutoContextTreeTooLarge) {
		t.Fatalf("expected errAutoContextTreeTooLarge, got %v", err)
	}
}

func TestBuildProviderConfigFallbackModel(t *testing.T) {
	cfg := buildProviderConfig(LLMSettings{
		ActiveProvider: LLMProviderOpenRouter,
		Model:          "",
		OpenRouterKey:  "k",
		BaseURL:        "http://localhost",
	})
	if cfg.Provider != LLMProviderOpenRouter {
		t.Fatalf("unexpected provider: %q", cfg.Provider)
	}
	if cfg.Model == "" {
		t.Fatal("expected fallback model to be set")
	}
	if cfg.APIKey != "k" {
		t.Fatalf("unexpected api key mapping: %q", cfg.APIKey)
	}
}
