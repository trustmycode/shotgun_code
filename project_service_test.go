package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	gitignore "github.com/sabhiram/go-gitignore"
)

func TestBuildTreeRecursiveSortsDirectoriesFirst(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "zdir"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "afile.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "bfile.txt"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	nodes, err := buildTreeRecursive(context.Background(), root, root, nil, nil, 0)
	if err != nil {
		t.Fatalf("buildTreeRecursive failed: %v", err)
	}
	if len(nodes) < 3 {
		t.Fatalf("expected at least 3 nodes, got %d", len(nodes))
	}
	if !nodes[0].IsDir {
		t.Fatalf("expected first node to be directory, got file: %+v", nodes[0])
	}
}

func TestListFilesMarksGitAndCustomIgnored(t *testing.T) {
	app := newTestApp(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored.txt\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "custom.txt"), []byte("y"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	app.currentCustomIgnorePatterns = gitignore.CompileIgnoreLines("custom.txt")
	nodes, err := app.ListFiles(root)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}
	if len(nodes) != 1 || len(nodes[0].Children) == 0 {
		t.Fatalf("unexpected tree shape: %+v", nodes)
	}

	var foundGitIgnored, foundCustomIgnored bool
	for _, child := range nodes[0].Children {
		switch child.Name {
		case "ignored.txt":
			foundGitIgnored = child.IsGitignored
		case "custom.txt":
			foundCustomIgnored = child.IsCustomIgnored
		}
	}
	if !foundGitIgnored {
		t.Fatal("expected ignored.txt to be marked as gitignored")
	}
	if !foundCustomIgnored {
		t.Fatal("expected custom.txt to be marked as custom ignored")
	}
}

func TestSaveAndLoadRepoScan(t *testing.T) {
	app := newTestApp(t)
	root := t.TempDir()
	content := "# repo scan\n- item"
	if err := app.SaveRepoScan(root, content); err != nil {
		t.Fatalf("SaveRepoScan failed: %v", err)
	}
	loaded, err := app.LoadRepoScan(root)
	if err != nil {
		t.Fatalf("LoadRepoScan failed: %v", err)
	}
	if loaded != content {
		t.Fatalf("unexpected loaded content: %q", loaded)
	}
}
