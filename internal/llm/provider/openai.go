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

type openAIProvider struct {
	model   string
	client  *openai.LLM
	apiKey  string
	baseURL string
}

func newOpenAIProvider(cfg Config) (LLMProvider, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, errors.New("openai provider requires an API key")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, errors.New("openai provider requires a model")
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		// Official default for OpenAI HTTP APIs.
		baseURL = "https://api.openai.com/v1"
	}

	opts := []openai.Option{
		openai.WithToken(apiKey),
		openai.WithModel(model),
	}
	// Preserve custom base URL behaviour for the langchaingo client.
	if strings.TrimSpace(cfg.BaseURL) != "" {
		opts = append(opts, openai.WithBaseURL(strings.TrimSpace(cfg.BaseURL)))
	}

	client, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to init openai client: %w", err)
	}

	return &openAIProvider{
		client:  client,
		model:   model,
		apiKey:  apiKey,
		baseURL: baseURL,
	}, nil
}

func (o *openAIProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	return ModelCatalog("openai")
}

func (o *openAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if o.client == nil {
		return "", errors.New("openai client is not configured")
	}

	// For GPT-5 family models, use the Responses API with reasoning and verbosity controls,
	// and **never** send temperature/top_p/logprobs.
	if isGPT5FamilyModel(o.model) {
		return o.generateViaResponsesAPI(ctx, prompt)
	}

	// For non-GPT-5 models we keep the existing behaviour with a small temperature.
	return llms.GenerateFromSinglePrompt(ctx, o.client, prompt,
		llms.WithModel(o.model),
		llms.WithTemperature(0.1),
	)
}

type responsesAPIReasoningConfig struct {
	Effort string `json:"effort"`
}

type responsesAPITextConfig struct {
	Verbosity string `json:"verbosity"`
}

type responsesAPIRequest struct {
	Model     string                     `json:"model"`
	Input     string                     `json:"input"`
	Reasoning responsesAPIReasoningConfig `json:"reasoning"`
	Text      responsesAPITextConfig      `json:"text"`
}

type responsesAPIOutputText struct {
	Text string `json:"text"`
}

type responsesAPIOutputItem struct {
	Type       string                 `json:"type"`
	OutputText responsesAPIOutputText `json:"output_text"`
}

type responsesAPIResponse struct {
	Output []responsesAPIOutputItem `json:"output"`
}

func (o *openAIProvider) generateViaResponsesAPI(ctx context.Context, prompt string) (string, error) {
	apiKey := strings.TrimSpace(o.apiKey)
	if apiKey == "" {
		return "", errors.New("openai API key is required for GPT-5 models")
	}

	baseURL := strings.TrimSpace(o.baseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/responses"

	payload := responsesAPIRequest{
		Model: o.model,
		Input: prompt,
		Reasoning: responsesAPIReasoningConfig{
			Effort: "medium",
		},
		Text: responsesAPITextConfig{
			Verbosity: "high",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenAI Responses payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI Responses request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	// Newer Responses API may expect an explicit beta header; sending it is safe and explicit.
	req.Header.Set("OpenAI-Beta", "responses=v1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("openai responses API request failed: %v", err)
		return "", fmt.Errorf("openai responses API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		limitedBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		log.Printf("openai responses API returned status %d: %s", resp.StatusCode, string(limitedBody))
		return "", fmt.Errorf("openai responses API returned non-2xx status %d", resp.StatusCode)
	}

	var decoded responsesAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		log.Printf("failed to decode openai responses API payload: %v", err)
		return "", fmt.Errorf("failed to decode openai responses API payload: %w", err)
	}

	if len(decoded.Output) == 0 {
		log.Printf("openai responses API response did not contain any output for model %s", o.model)
		return "", errors.New("openai responses API response did not contain any output")
	}

	text := strings.TrimSpace(decoded.Output[0].OutputText.Text)
	if text == "" {
		log.Printf("openai responses API output_text.text is empty for model %s", o.model)
		return "", errors.New("openai responses API response did not contain text output")
	}

	return text, nil
}

