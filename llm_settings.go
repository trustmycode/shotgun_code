package main

import (
	"errors"
	"fmt"
	"strings"

	"shotgun_code/internal/llm/provider"
)

type cachedProvider struct {
	cfg      provider.Config
	instance provider.LLMProvider
}

func normalizeProviderName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case LLMProviderOpenAI:
		return LLMProviderOpenAI
	case LLMProviderOpenRouter:
		return LLMProviderOpenRouter
	case LLMProviderGemini:
		return LLMProviderGemini
	case LLMProviderOllama:
		return LLMProviderOllama
	case LLMProviderLMStudio:
		return LLMProviderLMStudio
	default:
		return ""
	}
}

func defaultModelForProvider(providerName string) string {
	switch providerName {
	case LLMProviderOpenAI:
		return "gpt-5"
	case LLMProviderGemini:
		return "gemini-2.5-pro"
	case LLMProviderOpenRouter:
		return "openai/gpt-5"
	case LLMProviderOllama:
		return "llama3.2"
	case LLMProviderLMStudio:
		return "local-model"
	default:
		return ""
	}
}

func (l LLMSettings) keyForProvider(providerName string) string {
	switch normalizeProviderName(providerName) {
	case LLMProviderOpenAI:
		return strings.TrimSpace(l.OpenAIKey)
	case LLMProviderOpenRouter:
		return strings.TrimSpace(l.OpenRouterKey)
	case LLMProviderGemini:
		return strings.TrimSpace(l.GeminiKey)
	case LLMProviderOllama:
		return strings.TrimSpace(l.OllamaKey)
	case LLMProviderLMStudio:
		return strings.TrimSpace(l.LMStudioKey)
	default:
		return ""
	}
}

func providerRequiresAPIKey(providerName string) bool {
	switch providerName {
	case LLMProviderOpenAI, LLMProviderOpenRouter, LLMProviderGemini:
		return true
	default:
		return false
	}
}

func (a *App) ensureLLMSettingsDefaults() {
	settings := &a.settings.LLMSettings
	settings.ActiveProvider = normalizeProviderName(settings.ActiveProvider)
	settings.Model = strings.TrimSpace(settings.Model)
	settings.BaseURL = strings.TrimSpace(settings.BaseURL)
	settings.OpenAIKey = strings.TrimSpace(settings.OpenAIKey)
	settings.OpenRouterKey = strings.TrimSpace(settings.OpenRouterKey)
	settings.GeminiKey = strings.TrimSpace(settings.GeminiKey)
	settings.OllamaKey = strings.TrimSpace(settings.OllamaKey)
	settings.LMStudioKey = strings.TrimSpace(settings.LMStudioKey)

	if settings.ActiveProvider != "" &&
		providerRequiresAPIKey(settings.ActiveProvider) &&
		settings.keyForProvider(settings.ActiveProvider) == "" {
		safeLogWarning(a.ctx, "Active LLM provider is missing an API key; disabling auto-context.")
		settings.ActiveProvider = ""
		settings.Model = ""
	}
	if settings.ActiveProvider != "" && settings.Model == "" {
		settings.Model = defaultModelForProvider(settings.ActiveProvider)
	}
}

func (a *App) HasActiveLlmKey() bool {
	settings := a.settings.LLMSettings
	if settings.ActiveProvider == "" {
		return false
	}
	if providerRequiresAPIKey(settings.ActiveProvider) {
		return settings.keyForProvider(settings.ActiveProvider) != ""
	}
	return true
}

func (a *App) GetLlmSettings() LLMSettings {
	return a.settings.LLMSettings
}

func (a *App) SetLlmApiKey(providerName, apiKey string) error {
	providerName = normalizeProviderName(providerName)
	if providerName == "" {
		return errors.New("unknown provider")
	}
	apiKey = strings.TrimSpace(apiKey)
	switch providerName {
	case LLMProviderOpenAI:
		a.settings.LLMSettings.OpenAIKey = apiKey
	case LLMProviderOpenRouter:
		a.settings.LLMSettings.OpenRouterKey = apiKey
	case LLMProviderGemini:
		a.settings.LLMSettings.GeminiKey = apiKey
	case LLMProviderOllama:
		a.settings.LLMSettings.OllamaKey = apiKey
	case LLMProviderLMStudio:
		a.settings.LLMSettings.LMStudioKey = apiKey
	}
	if a.settings.LLMSettings.ActiveProvider == providerName && strings.TrimSpace(a.settings.LLMSettings.Model) == "" {
		a.settings.LLMSettings.Model = defaultModelForProvider(providerName)
	}
	a.ensureLLMSettingsDefaults()
	if err := a.saveSettings(); err != nil {
		return fmt.Errorf("failed to persist API key: %w", err)
	}
	a.invalidateProviderCache()
	return nil
}

func (a *App) SetLlmProvider(providerName string) error {
	providerName = normalizeProviderName(providerName)
	if providerName == "" {
		a.settings.LLMSettings.ActiveProvider = ""
		a.settings.LLMSettings.Model = ""
		a.invalidateProviderCache()
		return a.saveSettings()
	}
	if providerRequiresAPIKey(providerName) && a.settings.LLMSettings.keyForProvider(providerName) == "" {
		return fmt.Errorf("set API key for %s before activating it", providerName)
	}
	a.settings.LLMSettings.ActiveProvider = providerName
	if strings.TrimSpace(a.settings.LLMSettings.Model) == "" {
		a.settings.LLMSettings.Model = defaultModelForProvider(providerName)
	}
	a.ensureLLMSettingsDefaults()
	if err := a.saveSettings(); err != nil {
		return fmt.Errorf("failed to save provider: %w", err)
	}
	a.invalidateProviderCache()
	return nil
}

func (a *App) SetLlmModel(providerName, model string) error {
	providerName = normalizeProviderName(providerName)
	if providerName == "" {
		return errors.New("unknown provider")
	}
	if providerRequiresAPIKey(providerName) && a.settings.LLMSettings.keyForProvider(providerName) == "" {
		return fmt.Errorf("set API key for %s before selecting a model", providerName)
	}
	if strings.TrimSpace(model) == "" {
		return errors.New("model name is required")
	}
	a.settings.LLMSettings.ActiveProvider = providerName
	a.settings.LLMSettings.Model = strings.TrimSpace(model)
	a.ensureLLMSettingsDefaults()

	if err := a.saveSettings(); err != nil {
		return fmt.Errorf("failed to save model selection: %w", err)
	}
	a.invalidateProviderCache()
	return nil
}

func (a *App) SetLlmBaseURL(baseURL string) error {
	a.settings.LLMSettings.BaseURL = strings.TrimSpace(baseURL)
	if err := a.saveSettings(); err != nil {
		return fmt.Errorf("failed to save base URL: %w", err)
	}
	a.invalidateProviderCache()
	return nil
}

func (a *App) ListLlmModels(providerName string) ([]provider.ModelInfo, error) {
	providerName = normalizeProviderName(providerName)
	if providerName == "" {
		return nil, errors.New("unknown provider")
	}
	return provider.ModelCatalog(providerName)
}

func (a *App) invalidateProviderCache() {
	a.llmCache = cachedProvider{}
}

func (a *App) getOrCreateProvider(cfg provider.Config) (provider.LLMProvider, error) {
	if cfg.Provider == "" || cfg.Model == "" {
		return nil, errors.New("incomplete provider configuration")
	}
	if providerRequiresAPIKey(cfg.Provider) && cfg.APIKey == "" {
		return nil, errors.New("incomplete provider configuration")
	}
	if a.llmCache.instance != nil && a.llmCache.cfg == cfg {
		return a.llmCache.instance, nil
	}
	instance, err := provider.Factory(cfg)
	if err != nil {
		return nil, err
	}
	a.llmCache = cachedProvider{
		cfg:      cfg,
		instance: instance,
	}
	return instance, nil
}
