package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty uses provider default", value: ""},
		{name: "https endpoint", value: "https://example.com/v1"},
		{name: "reject http", value: "http://example.com/v1", wantErr: true},
		{name: "reject credentials", value: "https://user:pass@example.com/v1", wantErr: true},
		{name: "reject query", value: "https://example.com/v1?key=value", wantErr: true},
		{name: "reject missing host", value: "https:///v1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBaseURL(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateBaseURL(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestGetLlmSettingsDoesNotExposeKeys(t *testing.T) {
	app := NewApp()
	app.settings.LLMSettings = LLMSettings{
		ActiveProvider: "openai",
		Model:          "gpt-5",
		OpenAIKey:      "top-secret-openai-key",
		OpenRouterKey:  "top-secret-openrouter-key",
		GeminiKey:      "top-secret-gemini-key",
	}

	encoded, err := json.Marshal(app.GetLlmSettings())
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	for _, secret := range []string{"top-secret-openai-key", "top-secret-openrouter-key", "top-secret-gemini-key"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("public settings exposed a secret: %s", encoded)
		}
	}
	if !app.GetLlmSettings().HasOpenAIKey || !app.GetLlmSettings().HasOpenRouterKey || !app.GetLlmSettings().HasGeminiKey {
		t.Fatal("public settings must report configured keys")
	}
}
