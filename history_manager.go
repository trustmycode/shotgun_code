package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type PromptHistoryItem struct {
	ID                string    `json:"id"`
	Timestamp         time.Time `json:"timestamp"`
	UserTask          string    `json:"userTask"`
	ConstructedPrompt string    `json:"constructedPrompt"`
	Response          string    `json:"response"`
	APICall           string    `json:"apiCall,omitempty"`
}

type PromptHistory struct {
	Items []PromptHistoryItem `json:"items"`
}

type HistoryManager struct {
	app         *App
	historyPath string
	history     PromptHistory
	mu          sync.Mutex
}

func NewHistoryManager(app *App) *HistoryManager {
	return &HistoryManager{
		app:     app,
		history: PromptHistory{Items: []PromptHistoryItem{}},
	}
}

func (hm *HistoryManager) getHistoryFilePath() (string, error) {
	if hm.historyPath != "" {
		return hm.historyPath, nil
	}
	// Use the same directory as settings.json
	if hm.app.configPath != "" {
		dir := filepath.Dir(hm.app.configPath)
		hm.historyPath = filepath.Join(dir, "prompt_history.json")
		return hm.historyPath, nil
	}
	return "", errors.New("config path not initialized in App")
}

func (hm *HistoryManager) LoadHistory() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	path, err := hm.getHistoryFilePath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No history yet, start empty
			hm.history = PromptHistory{Items: []PromptHistoryItem{}}
			return nil
		}
		return err
	}

	err = json.Unmarshal(data, &hm.history)
	if err != nil {
		wailsRuntime.LogErrorf(hm.app.ctx, "Error unmarshalling history: %v", err)
		return err
	}
	return nil
}

func (hm *HistoryManager) SaveHistory() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	path, err := hm.getHistoryFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(hm.history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (hm *HistoryManager) AddItem(userTask, constructedPrompt, response, apiCall string) PromptHistoryItem {
	hm.mu.Lock()
	// Generate simple ID based on timestamp
	now := time.Now()
	item := PromptHistoryItem{
		ID:                fmt.Sprintf("%d", now.UnixNano()),
		Timestamp:         now,
		UserTask:          userTask,
		ConstructedPrompt: constructedPrompt,
		Response:          response,
		APICall:           apiCall,
	}
	// Prepend to keep newest first
	hm.history.Items = append([]PromptHistoryItem{item}, hm.history.Items...)
	hm.mu.Unlock()

	// Save asynchronously to avoid blocking UI too much
	go func() {
		if err := hm.SaveHistory(); err != nil {
			wailsRuntime.LogError(hm.app.ctx, "Failed to save history: "+err.Error())
		}
	}()

	return item
}

func (hm *HistoryManager) GetItems() []PromptHistoryItem {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	// Return a copy to avoid race conditions if modified elsewhere
	items := make([]PromptHistoryItem, len(hm.history.Items))
	copy(items, hm.history.Items)
	return items
}

func (hm *HistoryManager) Clear() error {
	hm.mu.Lock()
	hm.history.Items = []PromptHistoryItem{}
	hm.mu.Unlock()
	return hm.SaveHistory()
}

// --- App Methods Binding ---

func (a *App) ExecuteLLMPrompt(userTask, finalPrompt string) (PromptHistoryItem, error) {
	if !a.HasActiveLlmKey() {
		return PromptHistoryItem{}, errors.New("no active LLM configuration found")
	}

	cfg := buildProviderConfig(a.settings.LLMSettings)
	providerInstance, err := a.getOrCreateProvider(cfg)
	if err != nil {
		return PromptHistoryItem{}, fmt.Errorf("failed to create provider: %w", err)
	}

	wailsRuntime.LogInfof(a.ctx, "Executing LLM prompt via %s (%s)...", cfg.Provider, cfg.Model)

	// Use provider.Generate. Note: we don't have streaming here yet, so it waits for full response.
	response, apiCall, err := providerInstance.Generate(a.ctx, finalPrompt)

	var historyItem PromptHistoryItem
	if a.historyManager != nil {
		historyResponse := response
		if err != nil {
			historyResponse = fmt.Sprintf("ERROR during prompt execution: %v", err)
		}
		historyItem = a.historyManager.AddItem(userTask, finalPrompt, historyResponse, apiCall)
	}

	if err != nil {
		return PromptHistoryItem{}, fmt.Errorf("LLM generation failed: %w", err)
	}

	return historyItem, nil
}

func (a *App) GetPromptHistory() []PromptHistoryItem {
	if a.historyManager == nil {
		return []PromptHistoryItem{}
	}
	return a.historyManager.GetItems()
}

func (a *App) ClearPromptHistory() error {
	if a.historyManager == nil {
		return nil
	}
	return a.historyManager.Clear()
}
