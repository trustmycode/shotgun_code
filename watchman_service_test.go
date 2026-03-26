package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	gitignore "github.com/sabhiram/go-gitignore"
)

func TestWatchmanStartStopViaAppBindings(t *testing.T) {
	app := newTestApp(t)
	app.fileWatcher = NewWatchman(app)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if err := app.StartFileWatcher(root); err != nil {
		t.Fatalf("StartFileWatcher failed: %v", err)
	}
	if app.fileWatcher.fsWatcher == nil {
		t.Fatal("expected fsWatcher to be initialized")
	}

	// Trigger a couple of fsnotify events so run() path executes.
	_ = os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("y"), 0o644)
	_ = os.MkdirAll(filepath.Join(root, "newdir"), 0o755)
	time.Sleep(100 * time.Millisecond)

	if err := app.fileWatcher.RefreshIgnoresAndRescan(); err != nil {
		t.Fatalf("RefreshIgnoresAndRescan failed: %v", err)
	}

	if err := app.StopFileWatcher(); err != nil {
		t.Fatalf("StopFileWatcher failed: %v", err)
	}
}

func TestWatchmanStartWithEmptyRootIsNoop(t *testing.T) {
	app := newTestApp(t)
	w := NewWatchman(app)
	if err := w.Start(""); err != nil {
		t.Fatalf("expected no error for empty root: %v", err)
	}
}

func TestWatchmanAddPathsToWatcherRecursive(t *testing.T) {
	app := newTestApp(t)
	w := NewWatchman(app)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer watcher.Close()

	w.mu.Lock()
	w.rootDir = root
	w.fsWatcher = watcher
	w.watchedDirs = make(map[string]bool)
	w.mu.Unlock()

	w.addPathsToWatcherRecursive(root)
	w.mu.Lock()
	count := len(w.watchedDirs)
	w.mu.Unlock()
	if count == 0 {
		t.Fatal("expected at least one watched directory after recursive add")
	}

	app.notifyFileChange(root)
}

func TestMatchesIgnoreDirectoryTrailingSlashPattern(t *testing.T) {
	ign := gitignore.CompileIgnoreLines("node_modules/")
	if matchesIgnore(ign, "node_modules", true) != true {
		t.Fatal("expected directory match for node_modules/ pattern")
	}
	if matchesIgnore(ign, "node_modules/package.json", false) != true {
		t.Fatal("expected nested file to be matched by node_modules/ pattern")
	}
	if matchesIgnore(ign, "src", true) != false {
		t.Fatal("did not expect src to match node_modules/ pattern")
	}
}

func TestNormalizeWatchmanRelPath(t *testing.T) {
	if got := normalizeWatchmanRelPath(`.\src\pkg`); got != "src/pkg" {
		t.Fatalf("unexpected normalized path: %q", got)
	}
}
