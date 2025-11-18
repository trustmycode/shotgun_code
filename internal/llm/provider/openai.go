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

func (o *openAIProvider) Generate(ctx context.Context, prompt string) (string, string, error) {
	if o.client == nil {
		return "", "", errors.New("openai client is not configured")
	}

	// For GPT-5 family models, use the Responses API with reasoning and verbosity controls,
	// and **never** send temperature/top_p/logprobs.
	if isGPT5FamilyModel(o.model) {
		return o.generateViaResponsesAPI(ctx, prompt)
	}

	// For non-GPT-5 models we keep the existing behaviour with a small temperature.
	output, err := llms.GenerateFromSinglePrompt(ctx, o.client, prompt,
		llms.WithModel(o.model),
		llms.WithTemperature(0.1),
	)

	// Build a generic debug representation for the SDK-based call (no API key / raw text).
	debug := o.buildGenericAPICallDebug()

	if err != nil {
		return "", debug, err
	}
	return output, debug, nil
}

type responsesAPIReasoningConfig struct {
	Effort string `json:"effort"`
}

type responsesAPITextConfig struct {
	Verbosity string `json:"verbosity"`
}

type responsesAPIRequest struct {
	Model           string                      `json:"model"`
	Input           string                      `json:"input"`
	Reasoning       responsesAPIReasoningConfig `json:"reasoning"`
	Text            responsesAPITextConfig      `json:"text"`
	MaxOutputTokens int                         `json:"max_output_tokens,omitempty"`
}

type responsesAPIResponse struct {
	Output     json.RawMessage `json:"output"`
	OutputText string          `json:"output_text"`
}

func (o *openAIProvider) generateViaResponsesAPI(ctx context.Context, prompt string) (string, string, error) {
	apiKey := strings.TrimSpace(o.apiKey)
	if apiKey == "" {
		return "", "", errors.New("openai API key is required for GPT-5 models")
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
		// Явно ограничиваем длину ответа, чтобы модель уверенно возвращала сообщение.
		// Значение можно будет вынести в настройки при необходимости.
		MaxOutputTokens: 4096,
	}

	// Build sanitized debug view BEFORE marshalling real payload.
	debugPayload := responsesAPIRequest{
		Model: o.model,
		Input: "[request_text]",
		Reasoning: responsesAPIReasoningConfig{
			Effort: "medium",
		},
		Text: responsesAPITextConfig{
			Verbosity: "high",
		},
		MaxOutputTokens: payload.MaxOutputTokens,
	}
	debug := map[string]any{
		"provider": "openai",
		"endpoint": endpoint,
		"method":   http.MethodPost,
		"headers": map[string]string{
			"Authorization": "Bearer [apikey]",
			"Content-Type":  "application/json",
			"OpenAI-Beta":   "responses=v1",
		},
		"body": debugPayload,
	}
	debugBytes, err := json.MarshalIndent(debug, "", "  ")
	if err != nil {
		// In case of debug marshalling error, we still proceed with the real request but without debug details.
		debugBytes = nil
	}
	debugString := string(debugBytes)

	body, err := json.Marshal(payload)
	if err != nil {
		return "", debugString, fmt.Errorf("failed to marshal OpenAI Responses payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", debugString, fmt.Errorf("failed to create OpenAI Responses request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	// Newer Responses API may expect an explicit beta header; sending it is safe and explicit.
	req.Header.Set("OpenAI-Beta", "responses=v1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("openai responses API request failed: %v", err)
		return "", debugString, fmt.Errorf("openai responses API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		limitedBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		log.Printf("openai responses API returned status %d: %s", resp.StatusCode, string(limitedBody))
		return "", debugString, fmt.Errorf("openai responses API returned non-2xx status %d", resp.StatusCode)
	}

	var decoded responsesAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		log.Printf("failed to decode openai responses API payload: %v", err)
		return "", debugString, fmt.Errorf("failed to decode openai responses API payload: %w", err)
	}

	// 1) Сначала пытаемся использовать агрегированное поле output_text на верхнем уровне.
	if txt := strings.TrimSpace(decoded.OutputText); txt != "" {
		return txt, debugString, nil
	}

	// 2) Если его нет — извлекаем текст из массива output.
	text, extractErr := extractTextFromResponsesOutput(decoded.Output)
	if extractErr != nil {
		log.Printf("failed to extract text from openai responses API output for model %s: %v", o.model, extractErr)
		return "", debugString, extractErr
	}

	return text, debugString, nil
}

// extractTextFromResponsesOutput tries to handle current JSON shapes of the Responses API:
// 1) output: [ { "type": "output_text", "text": "..." } ]
// 2) output: [ { "type": "message", "role": "assistant", "content": [
//        { "type": "output_text", "text": "..." }, ...
//    ] } ]
func extractTextFromResponsesOutput(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", errors.New("openai responses API response did not contain any output")
	}

	// Shape 1: array of plain output_text items
	type outputTextItem struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		Annotations any    `json:"annotations"`
	}
	var asOutputText []outputTextItem
	if err := json.Unmarshal(raw, &asOutputText); err == nil && len(asOutputText) > 0 {
		for _, item := range asOutputText {
			if strings.EqualFold(item.Type, "output_text") && strings.TrimSpace(item.Text) != "" {
				return strings.TrimSpace(item.Text), nil
			}
		}
	}

	// Shape 2: array of messages with nested content[].text (string)
	type contentItem struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		Annotations any    `json:"annotations"`
	}
	type messageShape struct {
		Type    string        `json:"type"`
		Role    string        `json:"role"`
		Content []contentItem `json:"content"`
	}
	var asMessages []messageShape
	if err := json.Unmarshal(raw, &asMessages); err == nil && len(asMessages) > 0 {
		for _, msg := range asMessages {
			if !strings.EqualFold(msg.Type, "message") {
				continue
			}
			for _, c := range msg.Content {
				if !strings.EqualFold(c.Type, "output_text") {
					continue
				}
				if strings.TrimSpace(c.Text) != "" {
					return strings.TrimSpace(c.Text), nil
				}
			}
		}
	}

	return "", errors.New("openai responses API response did not contain text output")
}

// buildGenericAPICallDebug builds a high-level debug representation for SDK-based calls
// (non-GPT‑5 models). It intentionally masks the actual API key and request text.
func (o *openAIProvider) buildGenericAPICallDebug() string {
	debug := map[string]any{
		"provider": "openai",
		"model":    o.model,
		"baseURL":  o.baseURL,
		"sdk":      "langchaingo/llms.openai",
		"call":     "llms.GenerateFromSinglePrompt",
		"input":    "[request_text]",
		"headers": map[string]string{
			"Authorization": "Bearer [apikey]",
		},
	}

	data, err := json.MarshalIndent(debug, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

