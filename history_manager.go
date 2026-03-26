package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
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

func mergeHistoryItems(first, second []PromptHistoryItem) []PromptHistoryItem {
	combined := make([]PromptHistoryItem, 0, len(first)+len(second))
	combined = append(combined, first...)
	combined = append(combined, second...)

	seen := make(map[string]bool, len(combined))
	merged := make([]PromptHistoryItem, 0, len(combined))
	for _, item := range combined {
		id := item.ID
		if id == "" {
			id = fmt.Sprintf("%d|%s|%s", item.Timestamp.UnixNano(), item.UserTask, item.ConstructedPrompt)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		merged = append(merged, item)
	}

	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].Timestamp.After(merged[j].Timestamp)
	})
	return merged
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
	lockPath := path + ".lock"

	return withFileLock(lockPath, true, func() error {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				hm.history = PromptHistory{Items: []PromptHistoryItem{}}
				return nil
			}
			return err
		}

		err = json.Unmarshal(data, &hm.history)
		if err != nil {
			safeLogErrorf(hm.app.ctx, "Error unmarshalling history: %v", err)
			return err
		}
		return nil
	})
}

func (hm *HistoryManager) SaveHistory() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	return hm.saveHistoryLocked(true)
}

func (hm *HistoryManager) saveHistoryLocked(mergeWithOnDisk bool) error {
	path, err := hm.getHistoryFilePath()
	if err != nil {
		return err
	}

	if err := ensureParentDir(path); err != nil {
		return err
	}
	lockPath := path + ".lock"

	return withFileLock(lockPath, false, func() error {
		if mergeWithOnDisk {
			var onDisk PromptHistory
			existingData, err := os.ReadFile(path)
			if err == nil {
				if unmarshalErr := json.Unmarshal(existingData, &onDisk); unmarshalErr != nil {
					safeLogWarningf(hm.app.ctx, "Failed to parse existing history, overwriting with in-memory history: %v", unmarshalErr)
				}
			} else if !os.IsNotExist(err) {
				return err
			}

			hm.history.Items = mergeHistoryItems(hm.history.Items, onDisk.Items)
		}

		data, err := json.MarshalIndent(hm.history, "", "  ")
		if err != nil {
			return err
		}

		return os.WriteFile(path, data, 0o644)
	})
}

func (hm *HistoryManager) AddItem(userTask, constructedPrompt, response, apiCall string) PromptHistoryItem {
	hm.mu.Lock()
	now := time.Now()
	item := PromptHistoryItem{
		ID:                fmt.Sprintf("%d", now.UnixNano()),
		Timestamp:         now,
		UserTask:          userTask,
		ConstructedPrompt: constructedPrompt,
		Response:          response,
		APICall:           apiCall,
	}
	hm.history.Items = append([]PromptHistoryItem{item}, hm.history.Items...)
	hm.mu.Unlock()

	go func() {
		if err := hm.SaveHistory(); err != nil {
			safeLogError(hm.app.ctx, "Failed to save history: "+err.Error())
		}
	}()

	return item
}

func (hm *HistoryManager) GetItems() []PromptHistoryItem {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	items := make([]PromptHistoryItem, len(hm.history.Items))
	copy(items, hm.history.Items)
	return items
}

func (hm *HistoryManager) Clear() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.history.Items = []PromptHistoryItem{}
	return hm.saveHistoryLocked(false)
}

func (a *App) ExecuteLLMPrompt(userTask, finalPrompt string) (PromptHistoryItem, error) {
	if a.llmService == nil {
		a.llmService = NewLLMService(a)
	}
	return a.llmService.ExecutePromptSync(userTask, finalPrompt)
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
