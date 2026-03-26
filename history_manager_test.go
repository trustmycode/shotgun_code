package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

func TestHistoryManagerLoadAddClear(t *testing.T) {
	app := newTestApp(t)
	hm := NewHistoryManager(app)

	if err := hm.LoadHistory(); err != nil {
		t.Fatalf("LoadHistory failed on missing file: %v", err)
	}
	if len(hm.GetItems()) != 0 {
		t.Fatal("expected empty history on first load")
	}

	first := hm.AddItem("task-1", "prompt-1", "response-1", "api-1")
	second := hm.AddItem("task-2", "prompt-2", "response-2", "api-2")
	if first.ID == second.ID {
		t.Fatalf("expected unique IDs, got %q", first.ID)
	}

	items := hm.GetItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 history items, got %d", len(items))
	}
	if items[0].UserTask != "task-2" {
		t.Fatalf("expected newest item first, got %q", items[0].UserTask)
	}

	if err := hm.SaveHistory(); err != nil {
		t.Fatalf("SaveHistory failed: %v", err)
	}
	path, err := hm.getHistoryFilePath()
	if err != nil {
		t.Fatalf("getHistoryFilePath failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected history file to be readable: %v", err)
	}
	var decoded PromptHistory
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("history json should be valid: %v", err)
	}
	if len(decoded.Items) != 2 {
		t.Fatalf("expected 2 persisted items, got %d", len(decoded.Items))
	}

	if err := hm.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if len(hm.GetItems()) != 0 {
		t.Fatal("expected history to be empty after Clear")
	}
}

func TestAppHistoryBindings(t *testing.T) {
	app := newTestApp(t)
	app.historyManager = NewHistoryManager(app)
	_ = app.historyManager.LoadHistory()

	app.historyManager.AddItem("task", "prompt", "response", "api")
	items := app.GetPromptHistory()
	if len(items) != 1 {
		t.Fatalf("expected 1 item from app binding, got %d", len(items))
	}
	if err := app.ClearPromptHistory(); err != nil {
		t.Fatalf("ClearPromptHistory failed: %v", err)
	}
	if len(app.GetPromptHistory()) != 0 {
		t.Fatal("expected app history to be empty after clear")
	}
}

func TestHistoryManagerSaveHistoryTimesOutWhenFileLocked(t *testing.T) {
	app := newTestApp(t)
	hm := NewHistoryManager(app)
	hm.history.Items = []PromptHistoryItem{
		{
			ID:        "1",
			Timestamp: time.Now(),
			UserTask:  "task",
		},
	}

	path, err := hm.getHistoryFilePath()
	if err != nil {
		t.Fatalf("getHistoryFilePath failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	lock := flock.New(path + ".lock")
	locked, err := lock.TryLock()
	if err != nil {
		t.Fatalf("failed to lock history lock file: %v", err)
	}
	if !locked {
		t.Fatal("expected test lock to be acquired")
	}
	defer func() {
		_ = lock.Unlock()
	}()

	started := time.Now()
	saveErr := hm.SaveHistory()
	if saveErr == nil {
		t.Fatal("expected SaveHistory to fail on lock timeout")
	}
	if time.Since(started) < fileLockTimeout {
		t.Fatalf("expected SaveHistory to wait at least %s before failing", fileLockTimeout)
	}
}

func TestHistoryManagerSaveHistoryMergesOnDiskItems(t *testing.T) {
	app := newTestApp(t)
	hm := NewHistoryManager(app)

	now := time.Now()
	memItem := PromptHistoryItem{
		ID:        "mem-id",
		Timestamp: now,
		UserTask:  "mem",
	}
	diskItem := PromptHistoryItem{
		ID:        "disk-id",
		Timestamp: now.Add(-time.Minute),
		UserTask:  "disk",
	}
	hm.history.Items = []PromptHistoryItem{memItem}

	path, err := hm.getHistoryFilePath()
	if err != nil {
		t.Fatalf("getHistoryFilePath failed: %v", err)
	}
	data, err := json.Marshal(PromptHistory{Items: []PromptHistoryItem{diskItem}})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	if err := hm.SaveHistory(); err != nil {
		t.Fatalf("SaveHistory failed: %v", err)
	}

	persisted, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read persisted file failed: %v", err)
	}
	var decoded PromptHistory
	if err := json.Unmarshal(persisted, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(decoded.Items) != 2 {
		t.Fatalf("expected 2 merged items, got %d", len(decoded.Items))
	}
}
