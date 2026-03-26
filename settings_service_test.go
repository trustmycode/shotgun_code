package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

func TestLoadSettingsCreatesDefaultFile(t *testing.T) {
	app := newTestApp(t)
	app.loadSettings()

	if strings.TrimSpace(app.settings.CustomIgnoreRules) == "" {
		t.Fatal("expected default custom ignore rules to be loaded")
	}
	if got := app.GetCustomPromptRules(); strings.TrimSpace(got) == "" {
		t.Fatal("expected default custom prompt rules to be available")
	}
	if _, err := os.Stat(app.configPath); err != nil {
		t.Fatalf("expected settings file to be created: %v", err)
	}
}

func TestSetAndGetCustomPromptRules(t *testing.T) {
	app := newTestApp(t)
	app.loadSettings()

	rules := "be concise\nprefer explicit file paths"
	if err := app.SetCustomPromptRules(rules); err != nil {
		t.Fatalf("SetCustomPromptRules failed: %v", err)
	}
	if got := app.GetCustomPromptRules(); got != rules {
		t.Fatalf("unexpected prompt rules: %q", got)
	}
}

func TestSetCustomIgnoreRulesCompilesAndPersists(t *testing.T) {
	app := newTestApp(t)
	app.loadSettings()

	rules := "node_modules\n*.tmp"
	if err := app.SetCustomIgnoreRules(rules); err != nil {
		t.Fatalf("SetCustomIgnoreRules failed: %v", err)
	}
	if app.currentCustomIgnorePatterns == nil {
		t.Fatal("expected currentCustomIgnorePatterns to be compiled")
	}

	reloaded := &App{ctx: app.ctx, configPath: app.configPath}
	reloaded.loadSettings()
	if !strings.Contains(reloaded.GetCustomIgnoreRules(), "node_modules") {
		t.Fatalf("expected persisted ignore rules, got: %q", reloaded.GetCustomIgnoreRules())
	}
}

func TestSetUseIgnoreFlagsWithoutWatcher(t *testing.T) {
	app := newTestApp(t)
	if err := app.SetUseGitignore(false); err != nil {
		t.Fatalf("SetUseGitignore failed: %v", err)
	}
	if app.useGitignore {
		t.Fatal("expected useGitignore=false")
	}
	if err := app.SetUseCustomIgnore(false); err != nil {
		t.Fatalf("SetUseCustomIgnore failed: %v", err)
	}
	if app.useCustomIgnore {
		t.Fatal("expected useCustomIgnore=false")
	}
}

func TestCompileCustomIgnorePatternsEmpty(t *testing.T) {
	app := newTestApp(t)
	app.settings.CustomIgnoreRules = ""
	if err := app.compileCustomIgnorePatterns(); err != nil {
		t.Fatalf("compileCustomIgnorePatterns failed: %v", err)
	}
	if app.currentCustomIgnorePatterns != nil {
		t.Fatal("expected no custom patterns when rules are empty")
	}
}

func TestSaveSettingsCreatesDirectory(t *testing.T) {
	app := newTestApp(t)
	app.configPath = filepath.Join(t.TempDir(), "nested", "settings.json")
	app.settings.CustomIgnoreRules = "*.log"
	app.settings.CustomPromptRules = "no additional rules"
	if err := app.saveSettings(); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}
	if _, err := os.Stat(app.configPath); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
}

func TestSaveSettingsTimesOutWhenFileLocked(t *testing.T) {
	app := newTestApp(t)
	app.settings.CustomIgnoreRules = "*.log"
	app.settings.CustomPromptRules = "no additional rules"

	lock := flock.New(app.configPath + ".lock")
	locked, err := lock.TryLock()
	if err != nil {
		t.Fatalf("failed to lock settings lock file: %v", err)
	}
	if !locked {
		t.Fatal("expected test lock to be acquired")
	}
	defer func() {
		_ = lock.Unlock()
	}()

	started := time.Now()
	saveErr := app.saveSettings()
	if saveErr == nil {
		t.Fatal("expected saveSettings to fail on lock timeout")
	}
	if time.Since(started) < fileLockTimeout {
		t.Fatalf("expected saveSettings to wait at least %s before failing", fileLockTimeout)
	}
}
