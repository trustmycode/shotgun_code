package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"shotgun_code/internal/llm/provider"
)

type LLMService struct {
	app           *App
	streamEnabled bool

	mu        sync.Mutex
	streamMap map[string]context.CancelFunc
	counter   uint64
}

func NewLLMService(app *App) *LLMService {
	return &LLMService{
		app:           app,
		streamEnabled: isLLMStreamEnabled(),
		streamMap:     make(map[string]context.CancelFunc),
	}
}

func isLLMStreamEnabled() bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("LLM_STREAM_ENABLED")))
	if raw == "" {
		return true
	}
	switch raw {
	case "0", "false", "off", "no", "disabled":
		return false
	default:
		return true
	}
}

func (s *LLMService) ExecutePromptSync(userTask, finalPrompt string) (PromptHistoryItem, error) {
	if strings.TrimSpace(finalPrompt) == "" {
		return PromptHistoryItem{}, errors.New("prompt is required")
	}

	cfg, providerInstance, err := s.prepareProvider()
	if err != nil {
		return PromptHistoryItem{}, err
	}

	safeLogInfof(s.app.ctx, "Executing LLM prompt via %s (%s)...", cfg.Provider, cfg.Model)
	response, apiCall, err := providerInstance.Generate(s.app.ctx, finalPrompt)

	var historyItem PromptHistoryItem
	if s.app.historyManager != nil {
		historyResponse := response
		if err != nil {
			historyResponse = fmt.Sprintf("ERROR during prompt execution: %v", err)
		}
		historyItem = s.app.historyManager.AddItem(userTask, finalPrompt, historyResponse, apiCall)
	}

	if err != nil {
		return PromptHistoryItem{}, fmt.Errorf("LLM generation failed: %w", err)
	}

	return historyItem, nil
}

func (s *LLMService) StartPromptStream(userTask, finalPrompt string) (string, error) {
	if strings.TrimSpace(finalPrompt) == "" {
		return "", errors.New("prompt is required")
	}
	if !s.streamEnabled {
		return "", errors.New("streaming is disabled by LLM_STREAM_ENABLED")
	}

	cfg, providerInstance, err := s.prepareProvider()
	if err != nil {
		return "", err
	}

	requestID := s.nextRequestID()
	ctx, cancel := context.WithCancel(s.app.ctx)
	s.trackRequest(requestID, cancel)

	safeEventsEmit(s.app.ctx, "llmPromptStreamStart", map[string]any{
		"requestId": requestID,
		"provider":  cfg.Provider,
		"model":     cfg.Model,
		"startedAt": time.Now().UTC().Format(time.RFC3339Nano),
	})

	go s.runPromptStream(ctx, requestID, userTask, finalPrompt, providerInstance)
	return requestID, nil
}

func (s *LLMService) CancelPromptStream(requestID string) error {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return errors.New("requestId is required")
	}

	s.mu.Lock()
	cancel, ok := s.streamMap[requestID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("stream request %s was not found", requestID)
	}
	cancel()
	return nil
}

func (s *LLMService) runPromptStream(ctx context.Context, requestID, userTask, finalPrompt string, providerInstance interface {
	GenerateStream(context.Context, string, func(string)) (string, string, error)
}) {
	defer s.untrackRequest(requestID)

	var chunksMu sync.Mutex
	var totalChars int
	var streamed strings.Builder

	onChunk := func(chunk string) {
		if chunk == "" {
			return
		}
		chunksMu.Lock()
		streamed.WriteString(chunk)
		totalChars += len(chunk)
		currentTotal := totalChars
		chunksMu.Unlock()

		safeEventsEmit(s.app.ctx, "llmPromptStreamChunk", map[string]any{
			"requestId":  requestID,
			"chunk":      chunk,
			"totalChars": currentTotal,
		})
	}

	response, apiCall, err := providerInstance.GenerateStream(ctx, finalPrompt, onChunk)
	partial := strings.TrimSpace(streamed.String())
	if err != nil {
		if partial == "" && response != "" {
			partial = response
		}
		if s.app.historyManager != nil {
			errText := fmt.Sprintf("ERROR during prompt stream execution: %v", err)
			if partial != "" {
				errText = partial + "\n\n" + errText
			}
			s.app.historyManager.AddItem(userTask, finalPrompt, errText, apiCall)
		}

		safeEventsEmit(s.app.ctx, "llmPromptStreamError", map[string]any{
			"requestId":       requestID,
			"message":         err.Error(),
			"partialResponse": partial,
		})
		return
	}

	finalResponse := response
	if strings.TrimSpace(finalResponse) == "" {
		finalResponse = partial
	}
	if partial == "" && finalResponse != "" {
		onChunk(finalResponse)
	}

	var historyItem PromptHistoryItem
	if s.app.historyManager != nil {
		historyItem = s.app.historyManager.AddItem(userTask, finalPrompt, finalResponse, apiCall)
	}

	safeEventsEmit(s.app.ctx, "llmPromptStreamEnd", map[string]any{
		"requestId":     requestID,
		"response":      finalResponse,
		"apiCall":       apiCall,
		"historyItemId": historyItem.ID,
		"finishedAt":    time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *LLMService) prepareProvider() (provider.Config, provider.LLMProvider, error) {
	if !s.app.HasActiveLlmKey() {
		return provider.Config{}, nil, errors.New("no active LLM configuration found")
	}

	realCfg := buildProviderConfig(s.app.settings.LLMSettings)
	instance, err := s.app.getOrCreateProvider(realCfg)
	if err != nil {
		return provider.Config{}, nil, fmt.Errorf("failed to create provider: %w", err)
	}
	return realCfg, instance, nil
}

func (s *LLMService) trackRequest(requestID string, cancel context.CancelFunc) {
	s.mu.Lock()
	s.streamMap[requestID] = cancel
	s.mu.Unlock()
}

func (s *LLMService) untrackRequest(requestID string) {
	s.mu.Lock()
	delete(s.streamMap, requestID)
	s.mu.Unlock()
}

func (s *LLMService) nextRequestID() string {
	n := atomic.AddUint64(&s.counter, 1)
	return fmt.Sprintf("llm-%d-%d", time.Now().UnixNano(), n)
}

func (a *App) ExecuteLLMPromptStream(userTask, finalPrompt string) (string, error) {
	if a.llmService == nil {
		a.llmService = NewLLMService(a)
	}
	return a.llmService.StartPromptStream(userTask, finalPrompt)
}

func (a *App) CancelLLMPromptStream(requestID string) error {
	if a.llmService == nil {
		return errors.New("LLM service is not initialized")
	}
	return a.llmService.CancelPromptStream(requestID)
}
