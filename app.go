package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	gitignore "github.com/sabhiram/go-gitignore"

	"shotgun_code/internal/labgradient"
)

const maxOutputSizeBytes = 10_000_000 // 10MB

var ErrContextTooLong = errors.New("context is too long")

//go:embed ignore.glob
var defaultCustomIgnoreRulesContent string

const defaultCustomPromptRulesContent = "no additional rules"

const (
	LLMProviderOpenAI     = "openai"
	LLMProviderOpenRouter = "openrouter"
	LLMProviderGemini     = "gemini"
	LLMProviderOllama     = "ollama"
	LLMProviderLMStudio   = "lmstudio"
)

type LLMSettings struct {
	ActiveProvider string `json:"activeProvider"`
	Model          string `json:"model"`
	OpenAIKey      string `json:"openAIKey"`
	OpenRouterKey  string `json:"openRouterKey"`
	GeminiKey      string `json:"geminiKey"`
	OllamaKey      string `json:"ollamaKey"`
	LMStudioKey    string `json:"lmStudioKey"`
	BaseURL        string `json:"baseURL"`
}

type AppSettings struct {
	CustomIgnoreRules string      `json:"customIgnoreRules"`
	CustomPromptRules string      `json:"customPromptRules"`
	LLMSettings       LLMSettings `json:"llmSettings"`
}

type App struct {
	ctx                         context.Context
	contextGenerator            *ContextGenerator
	fileWatcher                 *Watchman
	settings                    AppSettings
	currentCustomIgnorePatterns *gitignore.GitIgnore
	configPath                  string
	useGitignore                bool
	useCustomIgnore             bool
	projectGitignore            *gitignore.GitIgnore
	autoContextService          *AutoContextService
	historyManager              *HistoryManager
	llmCache                    cachedProvider
	autoContextButtonTexture    string
	llmService                  *LLMService
	tokenService                *TokenService
	settingsMu                  sync.Mutex
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.contextGenerator = NewContextGenerator(a)
	a.autoContextService = NewAutoContextService()
	a.historyManager = NewHistoryManager(a)
	a.fileWatcher = NewWatchman(a)
	a.llmService = NewLLMService(a)
	a.tokenService = NewTokenService()
	a.useGitignore = true
	a.useCustomIgnore = true

	configFilePath, err := xdg.ConfigFile("shotgun-code/settings.json")
	if err != nil {
		safeLogErrorf(a.ctx, "Error getting config file path: %v. Using defaults and will attempt to save later if rules are modified.", err)
	}
	a.configPath = configFilePath

	a.loadSettings()
	if err := a.historyManager.LoadHistory(); err != nil {
		safeLogWarningf(a.ctx, "Failed to load prompt history: %v", err)
	}

	if strings.TrimSpace(a.settings.CustomPromptRules) == "" {
		a.settings.CustomPromptRules = defaultCustomPromptRulesContent
	}

	a.initAutoContextButtonTexture()
}

func (a *App) initAutoContextButtonTexture() {
	params := labgradient.SliceParams{
		L:               80.0,
		Radius:          60.0,
		VerticalSpanDeg: 30.0,
	}

	texture, err := labgradient.GeneratePanoramicTexture(3072, 64, params)
	if err != nil {
		safeLogErrorf(a.ctx, "failed to generate auto-context LAB texture: %v", err)
		return
	}

	a.autoContextButtonTexture = texture
	safeLogDebug(a.ctx, "auto-context LAB texture generated successfully")
}

func (a *App) GetAutoContextButtonTexture() string {
	return a.autoContextButtonTexture
}

func (a *App) RequestAutoContextSelection(rootDir string, excludedPaths []string, userTask string) ([]string, error) {
	if a.autoContextService == nil {
		return nil, errors.New("auto-context service is not initialized")
	}
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		return nil, errors.New("project root is required")
	}
	if !a.HasActiveLlmKey() {
		return nil, errors.New("no active LLM configuration found")
	}

	excludedMap := make(map[string]bool)
	for _, p := range excludedPaths {
		excludedMap[normalizeRelativePath(p)] = true
	}

	tree, err := buildAutoContextTree(rootDir, excludedMap)
	if err != nil {
		a.emitAutoContextError(fmt.Sprintf("failed to build project tree: %v", err))
		return nil, err
	}

	task := strings.TrimSpace(userTask)
	prompt, err := a.autoContextService.BuildPrompt(tree, task, "")
	if err != nil {
		a.emitAutoContextError(fmt.Sprintf("failed to render auto-context prompt: %v", err))
		return nil, err
	}

	cfg := buildProviderConfig(a.settings.LLMSettings)
	providerInstance, err := a.getOrCreateProvider(cfg)
	if err != nil {
		a.emitAutoContextError(fmt.Sprintf("failed to configure provider: %v", err))
		return nil, err
	}

	raw, apiCall, err := providerInstance.Generate(a.ctx, prompt)

	if a.historyManager != nil {
		historyLabel := "AUTO CONTEXT"
		if task != "" {
			const maxLabelRunes = 80
			runes := []rune(task)
			if len(runes) > maxLabelRunes {
				historyLabel = "AUTO CONTEXT: " + string(runes[:maxLabelRunes]) + "…"
			} else {
				historyLabel = "AUTO CONTEXT: " + task
			}
		}

		responseForHistory := raw
		if err != nil {
			responseForHistory = fmt.Sprintf("ERROR during auto-context LLM call: %v", err)
		}
		a.historyManager.AddItem(historyLabel, prompt, responseForHistory, apiCall)
	}

	if err != nil {
		a.emitAutoContextError(fmt.Sprintf("provider error: %v", err))
		return nil, err
	}

	parsed, err := a.autoContextService.ParseResponse(raw)
	if err != nil {
		a.emitAutoContextError(fmt.Sprintf("failed to parse LLM response: %v", err))
		return nil, err
	}

	selected, err := resolveLLMSelection(rootDir, parsed.Files)
	if err != nil {
		a.emitAutoContextError(fmt.Sprintf("unable to match LLM selection to files: %v", err))
		return nil, err
	}

	safeLogInfof(a.ctx, "Auto-context selected %d files via %s (%s)", len(selected), cfg.Provider, cfg.Model)
	return selected, nil
}

func (a *App) emitAutoContextError(message string) {
	safeLogError(a.ctx, message)
	safeEventsEmit(a.ctx, "autoContextError", message)
}
