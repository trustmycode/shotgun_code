package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	openai "github.com/tmc/langchaingo/llms/openai"
)

const defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"

type openRouterProvider struct {
	model  string
	client *openai.LLM
}

func newOpenRouterProvider(cfg Config) (LLMProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("openrouter provider requires an API key")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, errors.New("openrouter provider requires a model")
	}

	baseURL := defaultOpenRouterBaseURL
	if strings.TrimSpace(cfg.BaseURL) != "" {
		baseURL = strings.TrimSpace(cfg.BaseURL)
	}

	opts := []openai.Option{
		openai.WithToken(strings.TrimSpace(cfg.APIKey)),
		openai.WithModel(strings.TrimSpace(cfg.Model)),
		openai.WithBaseURL(baseURL),
	}

	client, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to init openrouter client: %w", err)
	}

	return &openRouterProvider{
		client: client,
		model:  strings.TrimSpace(cfg.Model),
	}, nil
}

func (o *openRouterProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	return ModelCatalog("openrouter")
}

func (o *openRouterProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if o.client == nil {
		return "", errors.New("openrouter client is not configured")
	}
	return llms.GenerateFromSinglePrompt(ctx, o.client, prompt, llms.WithModel(o.model), llms.WithTemperature(0.1))
}
