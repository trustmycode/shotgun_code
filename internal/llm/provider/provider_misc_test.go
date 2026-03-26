package provider

import (
	"context"
	"strings"
	"testing"
)

func TestIsGPT5FamilyModel(t *testing.T) {
	cases := map[string]bool{
		"gpt-5":        true,
		"gpt-5-mini":   true,
		"openai/gpt-5": true,
		"gpt-4o":       false,
		"":             false,
	}
	for model, expected := range cases {
		if got := isGPT5FamilyModel(model); got != expected {
			t.Fatalf("isGPT5FamilyModel(%q)=%v, expected %v", model, got, expected)
		}
	}
}

func TestGeminiProviderValidationAndNilClientPaths(t *testing.T) {
	if _, err := newGeminiProvider(Config{Provider: "gemini", APIKey: "", Model: "gemini-2.5-pro"}); err == nil {
		t.Fatal("expected error when API key is missing")
	}

	g := &geminiProvider{model: "gemini-2.5-pro", client: nil}
	if _, err := g.ListModels(context.Background()); err != nil {
		t.Fatalf("ListModels should return static catalog: %v", err)
	}
	if _, _, err := g.Generate(context.Background(), "hello"); err == nil {
		t.Fatal("expected error when gemini client is nil")
	}
	if _, _, err := g.GenerateStream(context.Background(), "hello", nil); err == nil {
		t.Fatal("expected stream error when gemini client is nil")
	}
}

func TestOpenRouterDebugBuilder(t *testing.T) {
	op := &openRouterProvider{model: "openai/gpt-5", baseURL: defaultOpenRouterBaseURL}
	debug := op.buildGenericAPICallDebug()
	if !strings.Contains(debug, "[apikey]") {
		t.Fatalf("expected masked key in debug payload: %s", debug)
	}
}

func TestOpenAIAndOpenRouterGenerateStreamWithNilClients(t *testing.T) {
	oa := &openAIProvider{model: "gpt-4o-mini", client: nil}
	if _, _, err := oa.GenerateStream(context.Background(), "x", nil); err == nil {
		t.Fatal("expected error for openai nil client")
	}
	or := &openRouterProvider{model: "openai/gpt-5", client: nil}
	if _, _, err := or.GenerateStream(context.Background(), "x", nil); err == nil {
		t.Fatal("expected error for openrouter nil client")
	}
}
