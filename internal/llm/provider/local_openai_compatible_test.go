package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocalProviderGenerate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hello"}}]}`))
	}))
	defer ts.Close()

	p, err := newOpenAICompatibleLocalProvider("ollama", Config{BaseURL: ts.URL + "/v1", Model: "llama3.2"}, defaultOllamaBaseURL, "llama3.2")
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	text, _, err := p.Generate(context.Background(), "say hi")
	if err != nil {
		t.Fatalf("generate returned error: %v", err)
	}
	if text != "hello" {
		t.Fatalf("expected 'hello', got %q", text)
	}
}

func TestLocalProviderGenerateStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer ts.Close()

	p, err := newOpenAICompatibleLocalProvider("lmstudio", Config{BaseURL: ts.URL + "/v1", Model: "local-model"}, defaultLMStudioBaseURL, "local-model")
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	var chunks []string
	text, _, err := p.GenerateStream(context.Background(), "say hi", func(chunk string) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Fatalf("stream returned error: %v", err)
	}
	if text != "Hello" {
		t.Fatalf("expected Hello, got %q", text)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	if strings.Join(chunks, "") != "Hello" {
		t.Fatalf("expected chunk concat Hello, got %q", strings.Join(chunks, ""))
	}
}
