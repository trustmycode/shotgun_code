package main

import (
	"context"
	"path/filepath"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	configDir := t.TempDir()
	app := &App{
		ctx:             context.Background(),
		configPath:      filepath.Join(configDir, "settings.json"),
		useGitignore:    true,
		useCustomIgnore: true,
	}
	app.historyManager = NewHistoryManager(app)
	app.llmService = NewLLMService(app)
	app.tokenService = NewTokenService()
	return app
}
