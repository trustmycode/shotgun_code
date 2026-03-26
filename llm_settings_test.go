package main

import (
	"testing"
)

func TestNormalizeProviderName(t *testing.T) {
	if got := normalizeProviderName(" OpenAI "); got != LLMProviderOpenAI {
		t.Fatalf("expected openai, got %q", got)
	}
	if got := normalizeProviderName("unknown"); got != "" {
		t.Fatalf("expected empty provider for unknown, got %q", got)
	}
}

func TestLocalProviderDoesNotRequireAPIKey(t *testing.T) {
	app := newTestApp(t)
	app.loadSettings()

	if err := app.SetLlmProvider(LLMProviderOllama); err != nil {
		t.Fatalf("SetLlmProvider(ollama) should work without API key: %v", err)
	}
	if !app.HasActiveLlmKey() {
		t.Fatal("expected HasActiveLlmKey=true for local provider")
	}
	if err := app.SetLlmModel(LLMProviderOllama, "llama3.2"); err != nil {
		t.Fatalf("SetLlmModel failed: %v", err)
	}
	if err := app.SetLlmBaseURL("http://localhost:11434/v1"); err != nil {
		t.Fatalf("SetLlmBaseURL failed: %v", err)
	}

	cfg := buildProviderConfig(app.settings.LLMSettings)
	if cfg.Provider != LLMProviderOllama {
		t.Fatalf("expected provider ollama, got %q", cfg.Provider)
	}
	if cfg.Model != "llama3.2" {
		t.Fatalf("expected model llama3.2, got %q", cfg.Model)
	}
}

func TestRemoteProviderRequiresAPIKey(t *testing.T) {
	app := newTestApp(t)
	app.loadSettings()

	if err := app.SetLlmProvider(LLMProviderOpenAI); err == nil {
		t.Fatal("expected error when setting openai provider without key")
	}
	if err := app.SetLlmApiKey(LLMProviderOpenAI, "test-key"); err != nil {
		t.Fatalf("SetLlmApiKey failed: %v", err)
	}
	if err := app.SetLlmProvider(LLMProviderOpenAI); err != nil {
		t.Fatalf("SetLlmProvider should succeed with key: %v", err)
	}
	if !app.HasActiveLlmKey() {
		t.Fatal("expected active key after setting provider + api key")
	}
}

func TestListLlmModelsIncludesLocalProviders(t *testing.T) {
	app := newTestApp(t)
	models, err := app.ListLlmModels(LLMProviderLMStudio)
	if err != nil {
		t.Fatalf("ListLlmModels failed: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("expected non-empty model catalog for LM Studio")
	}
}

func TestGetOrCreateProviderUsesCache(t *testing.T) {
	app := newTestApp(t)
	cfg := buildProviderConfig(LLMSettings{
		ActiveProvider: LLMProviderOllama,
		Model:          "llama3.2",
		BaseURL:        "http://localhost:11434/v1",
	})

	first, err := app.getOrCreateProvider(cfg)
	if err != nil {
		t.Fatalf("getOrCreateProvider first call failed: %v", err)
	}
	second, err := app.getOrCreateProvider(cfg)
	if err != nil {
		t.Fatalf("getOrCreateProvider second call failed: %v", err)
	}
	if first != second {
		t.Fatal("expected cached provider instance to be reused")
	}

	app.invalidateProviderCache()
	if app.llmCache.instance != nil {
		t.Fatal("expected provider cache to be reset")
	}
}

func TestEstimateTokensAppBinding(t *testing.T) {
	app := newTestApp(t)
	app.settings.LLMSettings.ActiveProvider = LLMProviderOpenAI
	app.settings.LLMSettings.Model = "gpt-4o-mini"

	estimate, err := app.EstimateTokens("", "", "hello world")
	if err != nil {
		t.Fatalf("EstimateTokens failed: %v", err)
	}
	if estimate.Tokens <= 0 {
		t.Fatalf("expected positive token count, got %d", estimate.Tokens)
	}
}
