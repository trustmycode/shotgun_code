package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractTextFromResponsesOutputShape1(t *testing.T) {
	raw := json.RawMessage(`[{"type":"output_text","text":"Hello from shape1"}]`)
	text, err := extractTextFromResponsesOutput(raw)
	if err != nil {
		t.Fatalf("extractTextFromResponsesOutput failed: %v", err)
	}
	if text != "Hello from shape1" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestExtractTextFromResponsesOutputShape2(t *testing.T) {
	raw := json.RawMessage(`[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Nested text"}]}]`)
	text, err := extractTextFromResponsesOutput(raw)
	if err != nil {
		t.Fatalf("extractTextFromResponsesOutput failed: %v", err)
	}
	if text != "Nested text" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestExtractTextFromResponsesOutputError(t *testing.T) {
	_, err := extractTextFromResponsesOutput(json.RawMessage(`[]`))
	if err == nil {
		t.Fatal("expected error for empty output")
	}
}

func TestFactoryAndModelCatalog(t *testing.T) {
	if _, err := Factory(Config{Provider: "unknown"}); err == nil {
		t.Fatal("expected unsupported provider error")
	}

	p, err := Factory(Config{Provider: "ollama", Model: "llama3.2", BaseURL: "http://localhost:11434/v1"})
	if err != nil {
		t.Fatalf("Factory should support ollama: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}

	models, err := ModelCatalog("openrouter")
	if err != nil {
		t.Fatalf("ModelCatalog failed: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("expected non-empty openrouter model catalog")
	}
}

func TestBuildGenericAPICallDebugMasked(t *testing.T) {
	op := &openAIProvider{model: "gpt-4o-mini", baseURL: "https://api.openai.com/v1"}
	debug := op.buildGenericAPICallDebug()
	if !strings.Contains(debug, "[apikey]") {
		t.Fatalf("expected masked api key in debug output: %s", debug)
	}
}
