package main

import (
	"context"
	"strings"
	"testing"
)

func TestGetPathFromDiffHeader(t *testing.T) {
	path := getPathFromDiffHeader("diff --git a/frontend/src/main.js b/frontend/src/main.js")
	if path != "a/frontend/src/main.js" {
		t.Fatalf("unexpected parsed path: %s", path)
	}
}

func TestSplitShotgunDiffHandlesPlainTextInput(t *testing.T) {
	app := &App{ctx: context.Background()}
	parts, err := app.SplitShotgunDiff("line1\nline2", 10)
	if err != nil {
		t.Fatalf("SplitShotgunDiff failed: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("expected single part, got %d", len(parts))
	}
}

func TestSplitShotgunDiffSplitsLargeBlocksByHunks(t *testing.T) {
	app := &App{ctx: context.Background()}
	diff := strings.Join([]string{
		"diff --git a/a.txt b/a.txt",
		"index 111..222 100644",
		"--- a/a.txt",
		"+++ b/a.txt",
		"@@ -1,1 +1,2 @@",
		"-a",
		"+a",
		"+b",
		"@@ -10,1 +10,2 @@",
		"-x",
		"+x",
		"+y",
		"diff --git a/b.txt b/b.txt",
		"index 333..444 100644",
		"--- a/b.txt",
		"+++ b/b.txt",
		"@@ -1,1 +1,1 @@",
		"-c",
		"+d",
	}, "\n")

	parts, err := app.SplitShotgunDiff(diff, 6)
	if err != nil {
		t.Fatalf("SplitShotgunDiff failed: %v", err)
	}
	if len(parts) < 2 {
		t.Fatalf("expected multiple parts for tight limit, got %d", len(parts))
	}
	for i, p := range parts {
		if strings.TrimSpace(p) == "" {
			t.Fatalf("split part %d is empty", i)
		}
	}
}

func TestSplitShotgunDiffSkipsMergeWhenLimitNonPositive(t *testing.T) {
	app := &App{ctx: context.Background()}
	diff := strings.Join([]string{
		"diff --git a/a.txt b/a.txt",
		"@@ -1 +1 @@",
		"-a",
		"+b",
		"diff --git a/b.txt b/b.txt",
		"@@ -1 +1 @@",
		"-c",
		"+d",
	}, "\n")
	parts, err := app.SplitShotgunDiff(diff, 0)
	if err != nil {
		t.Fatalf("SplitShotgunDiff failed: %v", err)
	}
	if len(parts) == 0 {
		t.Fatal("expected at least one split")
	}
}
