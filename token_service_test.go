package main

import "testing"

func TestTokenServiceEstimateOpenAIUsesTikToken(t *testing.T) {
	svc := NewTokenService()
	est := svc.Estimate(LLMProviderOpenAI, "gpt-4o-mini", "hello world")
	if est.Tokens <= 0 {
		t.Fatalf("expected positive token count, got %d", est.Tokens)
	}
	if est.Method != "tiktoken" {
		t.Fatalf("expected tiktoken method, got %s", est.Method)
	}
}

func TestTokenServiceEstimateFallbackHeuristic(t *testing.T) {
	svc := NewTokenService()
	est := svc.Estimate(LLMProviderGemini, "gemini-2.5-pro", "abcdefgh")
	if est.Method != "heuristic" {
		t.Fatalf("expected heuristic method, got %s", est.Method)
	}
	if est.Tokens != 2 {
		t.Fatalf("expected 2 tokens for 8 chars heuristic, got %d", est.Tokens)
	}
}

func TestNormalizeTokenModelOpenRouter(t *testing.T) {
	model := normalizeTokenModel(LLMProviderOpenRouter, "openai/gpt-5")
	if model != "gpt-5" {
		t.Fatalf("expected gpt-5, got %s", model)
	}
}

func TestNormalizeProviderNameSupportsLocalProviders(t *testing.T) {
	if got := normalizeProviderName("ollama"); got != LLMProviderOllama {
		t.Fatalf("expected %s, got %s", LLMProviderOllama, got)
	}
	if got := normalizeProviderName("lmstudio"); got != LLMProviderLMStudio {
		t.Fatalf("expected %s, got %s", LLMProviderLMStudio, got)
	}
}
