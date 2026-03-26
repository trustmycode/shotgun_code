package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

func (a *App) compileCustomIgnorePatterns() error {
	if strings.TrimSpace(a.settings.CustomIgnoreRules) == "" {
		a.currentCustomIgnorePatterns = nil
		safeLogDebug(a.ctx, "Custom ignore rules are empty, no patterns compiled.")
		return nil
	}

	lines := strings.Split(strings.ReplaceAll(a.settings.CustomIgnoreRules, "\r\n", "\n"), "\n")
	var validLines []string
	for _, line := range lines {
		validLines = append(validLines, line)
	}

	ign := gitignore.CompileIgnoreLines(validLines...)
	a.currentCustomIgnorePatterns = ign
	safeLogInfo(a.ctx, "Successfully compiled custom ignore patterns.")
	return nil
}

func (a *App) loadSettings() {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	a.loadSettingsLocked()
}

func (a *App) loadSettingsLocked() {
	a.settings.CustomIgnoreRules = defaultCustomIgnoreRulesContent

	if a.configPath == "" {
		safeLogWarningf(a.ctx, "Config path is empty, using default custom ignore rules (embedded).")
		_ = a.compileCustomIgnorePatterns()
		return
	}

	lockPath := a.configPath + ".lock"
	err := withFileLock(lockPath, true, func() error {
		data, readErr := os.ReadFile(a.configPath)
		if readErr != nil {
			return readErr
		}
		if unmarshalErr := json.Unmarshal(data, &a.settings); unmarshalErr != nil {
			return fmt.Errorf("failed to parse settings json: %w", unmarshalErr)
		}
		safeLogInfo(a.ctx, "Successfully loaded custom ignore rules from config.")
		if strings.TrimSpace(a.settings.CustomIgnoreRules) == "" && strings.TrimSpace(defaultCustomIgnoreRulesContent) != "" {
			safeLogInfo(a.ctx, "Loaded custom ignore rules are empty, falling back to default embedded rules.")
			a.settings.CustomIgnoreRules = defaultCustomIgnoreRulesContent
		}
		if strings.TrimSpace(a.settings.CustomPromptRules) == "" {
			safeLogInfo(a.ctx, "Custom prompt rules are empty or missing, using default.")
			a.settings.CustomPromptRules = defaultCustomPromptRulesContent
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			safeLogInfo(a.ctx, "Settings file not found. Using default custom ignore rules (embedded) and attempting to save them.")
			if errSave := a.saveSettingsLocked(); errSave != nil {
				safeLogErrorf(a.ctx, "Failed to save default settings: %v", errSave)
			}
		} else {
			safeLogErrorf(a.ctx, "Error reading settings file %s: %v. Using default custom ignore rules (embedded).", a.configPath, err)
		}
	}

	a.ensureLLMSettingsDefaults()
	_ = a.compileCustomIgnorePatterns()
}

func (a *App) saveSettings() error {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	return a.saveSettingsLocked()
}

func (a *App) saveSettingsLocked() error {
	if a.configPath == "" {
		err := errors.New("config path is not set, cannot save settings")
		safeLogError(a.ctx, err.Error())
		return err
	}

	data, err := json.MarshalIndent(a.settings, "", "  ")
	if err != nil {
		safeLogErrorf(a.ctx, "Error marshalling settings: %v", err)
		return err
	}

	configDir := filepath.Dir(a.configPath)
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		safeLogErrorf(a.ctx, "Error creating config directory %s: %v", configDir, err)
		return err
	}

	lockPath := a.configPath + ".lock"
	if err := withFileLock(lockPath, false, func() error {
		return os.WriteFile(a.configPath, data, 0o644)
	}); err != nil {
		safeLogErrorf(a.ctx, "Error writing settings to %s: %v", a.configPath, err)
		return err
	}

	safeLogInfo(a.ctx, "Settings saved successfully.")
	return nil
}

func (a *App) GetCustomIgnoreRules() string {
	return a.settings.CustomIgnoreRules
}

func (a *App) SetCustomIgnoreRules(rules string) error {
	a.settings.CustomIgnoreRules = rules
	compileErr := a.compileCustomIgnorePatterns()

	saveErr := a.saveSettings()
	if saveErr != nil {
		return fmt.Errorf("failed to save settings: %w (compile error: %v)", saveErr, compileErr)
	}
	if compileErr != nil {
		return fmt.Errorf("rules saved, but failed to compile custom ignore patterns: %w", compileErr)
	}

	if a.fileWatcher != nil && a.fileWatcher.rootDir != "" {
		return a.fileWatcher.RefreshIgnoresAndRescan()
	}
	return nil
}

func (a *App) GetCustomPromptRules() string {
	if strings.TrimSpace(a.settings.CustomPromptRules) == "" {
		return defaultCustomPromptRulesContent
	}
	return a.settings.CustomPromptRules
}

func (a *App) SetCustomPromptRules(rules string) error {
	a.settings.CustomPromptRules = rules
	if err := a.saveSettings(); err != nil {
		return fmt.Errorf("failed to save custom prompt rules: %w", err)
	}
	safeLogInfo(a.ctx, "Custom prompt rules saved successfully.")
	return nil
}

func (a *App) SetUseGitignore(enabled bool) error {
	a.useGitignore = enabled
	safeLogInfof(a.ctx, "App setting useGitignore changed to: %v", enabled)
	if a.fileWatcher != nil && a.fileWatcher.rootDir != "" {
		return a.fileWatcher.RefreshIgnoresAndRescan()
	}
	return nil
}

func (a *App) SetUseCustomIgnore(enabled bool) error {
	a.useCustomIgnore = enabled
	safeLogInfof(a.ctx, "App setting useCustomIgnore changed to: %v", enabled)
	if a.fileWatcher != nil && a.fileWatcher.rootDir != "" {
		return a.fileWatcher.RefreshIgnoresAndRescan()
	}
	return nil
}
