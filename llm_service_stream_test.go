package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"shotgun_code/internal/llm/provider"
)

type fakeLLMProvider struct {
	streamFn func(context.Context, string, func(string)) (string, string, error)
}

func (f *fakeLLMProvider) ListModels(context.Context) ([]provider.ModelInfo, error) {
	return nil, nil
}

func (f *fakeLLMProvider) Generate(context.Context, string) (string, string, error) {
	return "", "", nil
}

func (f *fakeLLMProvider) GenerateStream(ctx context.Context, prompt string, onChunk func(string)) (string, string, error) {
	if f.streamFn != nil {
		return f.streamFn(ctx, prompt, onChunk)
	}
	return "ok", "api", nil
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func TestStartPromptStreamCompletesAndUntracksRequest(t *testing.T) {
	app := newTestApp(t)
	app.settings.LLMSettings.ActiveProvider = LLMProviderOllama
	app.settings.LLMSettings.Model = "llama3.2"

	fake := &fakeLLMProvider{
		streamFn: func(_ context.Context, _ string, onChunk func(string)) (string, string, error) {
			onChunk("he")
			onChunk("llo")
			return "hello", "api-call", nil
		},
	}
	cfg := buildProviderConfig(app.settings.LLMSettings)
	app.llmCache = cachedProvider{cfg: cfg, instance: fake}

	svc := NewLLMService(app)
	requestID, err := svc.StartPromptStream("task", "prompt")
	if err != nil {
		t.Fatalf("StartPromptStream failed: %v", err)
	}
	if requestID == "" {
		t.Fatal("expected non-empty request ID")
	}

	waitForCondition(t, time.Second, func() bool {
		svc.mu.Lock()
		defer svc.mu.Unlock()
		_, exists := svc.streamMap[requestID]
		return !exists
	})

	if len(app.historyManager.GetItems()) == 0 {
		t.Fatal("expected history item to be saved on stream end")
	}
}

func TestCancelPromptStreamCancelsRequest(t *testing.T) {
	app := newTestApp(t)
	app.settings.LLMSettings.ActiveProvider = LLMProviderOllama
	app.settings.LLMSettings.Model = "llama3.2"

	started := make(chan struct{})
	var once sync.Once
	fake := &fakeLLMProvider{
		streamFn: func(ctx context.Context, _ string, _ func(string)) (string, string, error) {
			once.Do(func() { close(started) })
			<-ctx.Done()
			return "", "api-call", ctx.Err()
		},
	}
	cfg := buildProviderConfig(app.settings.LLMSettings)
	app.llmCache = cachedProvider{cfg: cfg, instance: fake}

	svc := NewLLMService(app)
	requestID, err := svc.StartPromptStream("task", "prompt")
	if err != nil {
		t.Fatalf("StartPromptStream failed: %v", err)
	}
	<-started

	if err := svc.CancelPromptStream(requestID); err != nil {
		t.Fatalf("CancelPromptStream failed: %v", err)
	}

	waitForCondition(t, time.Second, func() bool {
		svc.mu.Lock()
		defer svc.mu.Unlock()
		_, exists := svc.streamMap[requestID]
		return !exists
	})
}

func TestCancelPromptStreamValidation(t *testing.T) {
	svc := NewLLMService(newTestApp(t))
	if err := svc.CancelPromptStream(""); err == nil {
		t.Fatal("expected validation error for empty request ID")
	}
	if err := svc.CancelPromptStream("missing"); err == nil {
		t.Fatal("expected not found error for unknown request ID")
	}
}

func TestExecutePromptSyncValidatesPrompt(t *testing.T) {
	svc := NewLLMService(newTestApp(t))
	_, err := svc.ExecutePromptSync("task", "")
	if err == nil {
		t.Fatal("expected validation error for empty prompt")
	}
}

func TestAppCancelLLMPromptStreamWithoutService(t *testing.T) {
	app := &App{}
	err := app.CancelLLMPromptStream("req")
	if err == nil {
		t.Fatal("expected error when llm service is nil")
	}
	if err.Error() != "LLM service is not initialized" {
		t.Fatalf("unexpected error: %v", err)
	}
}
