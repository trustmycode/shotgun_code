package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/tmc/langchaingo/llms"
	openai "github.com/tmc/langchaingo/llms/openai"
)

const defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"

type openRouterProvider struct {
	model   string
	client  *openai.LLM
	apiKey  string
	baseURL string
}

func newOpenRouterProvider(cfg Config) (LLMProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("openrouter provider requires an API key")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, errors.New("openrouter provider requires a model")
	}

	baseURL := defaultOpenRouterBaseURL
	if trimmed := strings.TrimSpace(cfg.BaseURL); trimmed != "" {
		baseURL = trimmed
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
		client:  client,
		model:   strings.TrimSpace(cfg.Model),
		apiKey:  strings.TrimSpace(cfg.APIKey),
		baseURL: baseURL,
	}, nil
}

func (o *openRouterProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	return ModelCatalog("openrouter")
}

func (o *openRouterProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if o.client == nil {
		return "", errors.New("openrouter client is not configured")
	}

	// Для моделей семейства GPT‑5 используем ручной вызов OpenRouter Chat Completions API
	// с явным указанием reasoning.effort и text.verbosity и без передачи temperature.
	if isGPT5FamilyModel(o.model) {
		return o.generateViaOpenRouterAPI(ctx, prompt)
	}

	// Для остальных моделей сохраняем текущее поведение через langchaingo.
	return llms.GenerateFromSinglePrompt(ctx, o.client, prompt,
		llms.WithModel(o.model),
		llms.WithTemperature(0.1),
	)
}

type openRouterChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterChatChoice struct {
	Message openRouterChatMessage `json:"message"`
}

type openRouterChatResponse struct {
	Choices []openRouterChatChoice `json:"choices"`
}

type openRouterReasoningConfig struct {
	Effort string `json:"effort"`
}

type openRouterTextConfig struct {
	Verbosity string `json:"verbosity"`
}

type openRouterChatRequest struct {
	Model     string                   `json:"model"`
	Messages  []openRouterChatMessage  `json:"messages"`
	Reasoning openRouterReasoningConfig `json:"reasoning"`
	Text      openRouterTextConfig      `json:"text"`
}

func (o *openRouterProvider) generateViaOpenRouterAPI(ctx context.Context, prompt string) (string, error) {
	apiKey := strings.TrimSpace(o.apiKey)
	if apiKey == "" {
		return "", errors.New("openrouter API key is required for GPT-5 models")
	}

	baseURL := strings.TrimSpace(o.baseURL)
	if baseURL == "" {
		baseURL = defaultOpenRouterBaseURL
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"

	payload := openRouterChatRequest{
		Model: o.model,
		Messages: []openRouterChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Reasoning: openRouterReasoningConfig{
			Effort: "medium",
		},
		Text: openRouterTextConfig{
			Verbosity: "high",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenRouter chat payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenRouter chat request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("openrouter chat request failed (model=%s): %v", o.model, err)
		return "", fmt.Errorf("openrouter chat request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		limitedBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		log.Printf("openrouter chat returned status %d for model %s: %s", resp.StatusCode, o.model, string(limitedBody))
		return "", fmt.Errorf("openrouter chat API returned non-2xx status %d", resp.StatusCode)
	}

	var decoded openRouterChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		log.Printf("failed to decode openrouter chat payload for model %s: %v", o.model, err)
		return "", fmt.Errorf("failed to decode openrouter chat payload: %w", err)
	}

	if len(decoded.Choices) == 0 {
		log.Printf("openrouter chat response did not contain any choices for model %s", o.model)
		return "", errors.New("openrouter chat response did not contain any choices")
	}

	text := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if text == "" {
		log.Printf("openrouter chat response contained empty message content for model %s", o.model)
		return "", errors.New("openrouter chat response did not contain text output")
	}

	return text, nil
}
