package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryRetentionAndPermissions(t *testing.T) {
	app := NewApp()
	app.configPath = filepath.Join(t.TempDir(), "shotgun-code", "settings.json")
	hm := NewHistoryManager(app)
	for i := 0; i < maxPromptHistoryItems+10; i++ {
		hm.history.Items = append(hm.history.Items, PromptHistoryItem{ID: fmt.Sprint(i)})
	}

	if err := hm.SaveHistory(); err != nil {
		t.Fatalf("SaveHistory() error = %v", err)
	}
	path, err := hm.getHistoryFilePath()
	if err != nil {
		t.Fatalf("getHistoryFilePath() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != privateFileMode {
		t.Fatalf("history mode = %o, want %o", got, privateFileMode)
	}

	loaded := NewHistoryManager(app)
	if err := loaded.LoadHistory(); err != nil {
		t.Fatalf("LoadHistory() error = %v", err)
	}
	if got := len(loaded.GetItems()); got != maxPromptHistoryItems {
		t.Fatalf("history items = %d, want %d", got, maxPromptHistoryItems)
	}
}

func TestTrimPromptHistoryEnforcesByteLimit(t *testing.T) {
	large := string(make([]byte, maxPromptHistoryBytes/2+1))
	items := []PromptHistoryItem{
		{ID: "newest", ConstructedPrompt: large},
		{ID: "older", ConstructedPrompt: large},
	}
	trimmed := trimPromptHistory(items)
	if len(trimmed) != 1 || trimmed[0].ID != "newest" {
		t.Fatalf("trimPromptHistory() kept %#v, want only newest item", trimmed)
	}
}
