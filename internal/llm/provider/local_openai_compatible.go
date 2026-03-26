package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultOllamaBaseURL   = "http://localhost:11434/v1"
	defaultLMStudioBaseURL = "http://localhost:1234/v1"
)

type openAICompatibleLocalProvider struct {
	providerName string
	model        string
	apiKey       string
	baseURL      string
}

type openAICompatibleChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAICompatibleChatRequest struct {
	Model    string                        `json:"model"`
	Messages []openAICompatibleChatMessage `json:"messages"`
	Stream   bool                          `json:"stream,omitempty"`
}

type openAICompatibleChatChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
}

type openAICompatibleChatResponse struct {
	Choices []openAICompatibleChatChoice `json:"choices"`
}

func newOllamaProvider(cfg Config) (LLMProvider, error) {
	return newOpenAICompatibleLocalProvider("ollama", cfg, defaultOllamaBaseURL, "llama3.2")
}

func newLMStudioProvider(cfg Config) (LLMProvider, error) {
	return newOpenAICompatibleLocalProvider("lmstudio", cfg, defaultLMStudioBaseURL, "local-model")
}

func newOpenAICompatibleLocalProvider(name string, cfg Config, defaultBaseURL, defaultModel string) (LLMProvider, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultModel
	}
	if model == "" {
		return nil, errors.New("model is required")
	}

	return &openAICompatibleLocalProvider{
		providerName: name,
		model:        model,
		apiKey:       strings.TrimSpace(cfg.APIKey),
		baseURL:      baseURL,
	}, nil
}

func (p *openAICompatibleLocalProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	return ModelCatalog(p.providerName)
}

func (p *openAICompatibleLocalProvider) Generate(ctx context.Context, prompt string) (string, string, error) {
	return p.generateInternal(ctx, prompt, false, nil)
}

func (p *openAICompatibleLocalProvider) GenerateStream(ctx context.Context, prompt string, onChunk func(chunk string)) (string, string, error) {
	output, debug, err := p.generateInternal(ctx, prompt, true, onChunk)
	if err == nil {
		return output, debug, nil
	}

	fallbackOutput, fallbackDebug, fallbackErr := p.generateInternal(ctx, prompt, false, nil)
	if fallbackErr != nil {
		return "", fallbackDebug, err
	}
	if onChunk != nil && fallbackOutput != "" {
		onChunk(fallbackOutput)
	}
	return fallbackOutput, fallbackDebug, nil
}

func (p *openAICompatibleLocalProvider) generateInternal(ctx context.Context, prompt string, stream bool, onChunk func(chunk string)) (string, string, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(p.baseURL), "/") + "/chat/completions"

	payload := openAICompatibleChatRequest{
		Model: p.model,
		Messages: []openAICompatibleChatMessage{{
			Role:    "user",
			Content: prompt,
		}},
		Stream: stream,
	}

	debugPayload := openAICompatibleChatRequest{
		Model: p.model,
		Messages: []openAICompatibleChatMessage{{
			Role:    "user",
			Content: "[request_text]",
		}},
		Stream: stream,
	}
	debug := map[string]any{
		"provider": p.providerName,
		"endpoint": endpoint,
		"method":   http.MethodPost,
		"headers": map[string]string{
			"Authorization": maskedAuthHeader(p.apiKey),
			"Content-Type":  "application/json",
		},
		"body": debugPayload,
	}
	debugBytes, _ := json.MarshalIndent(debug, "", "  ")
	debugString := string(debugBytes)

	body, err := json.Marshal(payload)
	if err != nil {
		return "", debugString, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", debugString, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(p.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(p.apiKey))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", debugString, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		limitedBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", debugString, fmt.Errorf("chat API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(limitedBody)))
	}

	if !stream {
		var decoded openAICompatibleChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
			return "", debugString, fmt.Errorf("failed to decode response: %w", err)
		}
		if len(decoded.Choices) == 0 {
			return "", debugString, errors.New("response did not contain choices")
		}
		text := strings.TrimSpace(decoded.Choices[0].Message.Content)
		if text == "" {
			return "", debugString, errors.New("response did not contain text output")
		}
		return text, debugString, nil
	}

	var builder strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk openAICompatibleChatResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		piece := chunk.Choices[0].Delta.Content
		if piece == "" {
			piece = chunk.Choices[0].Message.Content
		}
		if piece == "" {
			continue
		}

		builder.WriteString(piece)
		if onChunk != nil {
			onChunk(piece)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", debugString, fmt.Errorf("stream read failed: %w", err)
	}

	output := strings.TrimSpace(builder.String())
	if output == "" {
		return "", debugString, errors.New("stream response did not contain text output")
	}
	return output, debugString, nil
}

func maskedAuthHeader(apiKey string) string {
	if strings.TrimSpace(apiKey) == "" {
		return "[none]"
	}
	return "Bearer [apikey]"
}
