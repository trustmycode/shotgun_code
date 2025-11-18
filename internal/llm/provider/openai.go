package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	openai "github.com/tmc/langchaingo/llms/openai"
)

type openAIProvider struct {
	model  string
	client *openai.LLM
}

func newOpenAIProvider(cfg Config) (LLMProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("openai provider requires an API key")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, errors.New("openai provider requires a model")
	}

	opts := []openai.Option{
		openai.WithToken(strings.TrimSpace(cfg.APIKey)),
		openai.WithModel(strings.TrimSpace(cfg.Model)),
	}
	if base := strings.TrimSpace(cfg.BaseURL); base != "" {
		opts = append(opts, openai.WithBaseURL(base))
	}

	client, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to init openai client: %w", err)
	}

	return &openAIProvider{
		client: client,
		model:  strings.TrimSpace(cfg.Model),
	}, nil
}

func (o *openAIProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	return ModelCatalog("openai")
}

func (o *openAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if o.client == nil {
		return "", errors.New("openai client is not configured")
	}
	return llms.GenerateFromSinglePrompt(ctx, o.client, prompt, llms.WithModel(o.model), llms.WithTemperature(0.1))
}
