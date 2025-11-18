package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
	"encoding/json"
)

type geminiProvider struct {
	model  string
	client *googleai.GoogleAI
}

func newGeminiProvider(cfg Config) (LLMProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("gemini provider requires an API key")
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gemini-1.5-flash"
	}

	client, err := googleai.New(
		context.Background(),
		googleai.WithAPIKey(strings.TrimSpace(cfg.APIKey)),
		googleai.WithDefaultModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init gemini client: %w", err)
	}

	return &geminiProvider{
		client: client,
		model:  model,
	}, nil
}

func (g *geminiProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	return ModelCatalog("gemini")
}

func (g *geminiProvider) Generate(ctx context.Context, prompt string) (string, string, error) {
	if g.client == nil {
		return "", "", errors.New("gemini client is not configured")
	}
	output, err := llms.GenerateFromSinglePrompt(ctx, g.client, prompt, llms.WithModel(g.model), llms.WithTemperature(0.1))

	debug := map[string]any{
		"provider": "gemini",
		"model":    g.model,
		"sdk":      "langchaingo/llms.googleai",
		"call":     "llms.GenerateFromSinglePrompt",
		"input":    "[request_text]",
	}

	data, mErr := json.MarshalIndent(debug, "", "  ")
	debugString := ""
	if mErr == nil {
		debugString = string(data)
	}

	if err != nil {
		return "", debugString, err
	}
	return output, debugString, nil
}
