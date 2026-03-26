package main

import (
	"os"
	"testing"
)

func TestIsLLMStreamEnabledDefaultAndOverrides(t *testing.T) {
	old := os.Getenv("LLM_STREAM_ENABLED")
	defer func() {
		_ = os.Setenv("LLM_STREAM_ENABLED", old)
	}()

	_ = os.Unsetenv("LLM_STREAM_ENABLED")
	if !isLLMStreamEnabled() {
		t.Fatal("expected streaming enabled by default")
	}

	_ = os.Setenv("LLM_STREAM_ENABLED", "false")
	if isLLMStreamEnabled() {
		t.Fatal("expected streaming disabled when env=false")
	}

	_ = os.Setenv("LLM_STREAM_ENABLED", "true")
	if !isLLMStreamEnabled() {
		t.Fatal("expected streaming enabled when env=true")
	}
}

func TestNextRequestIDUnique(t *testing.T) {
	svc := NewLLMService(&App{})
	first := svc.nextRequestID()
	second := svc.nextRequestID()
	if first == second {
		t.Fatalf("expected unique request IDs, got identical value %s", first)
	}
}
