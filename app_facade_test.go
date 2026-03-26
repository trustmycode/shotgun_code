package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"shotgun_code/internal/llm/provider"
)

type appTestProvider struct {
	response string
	err      error
}

func (p *appTestProvider) ListModels(context.Context) ([]provider.ModelInfo, error) {
	return nil, nil
}

func (p *appTestProvider) Generate(context.Context, string) (string, string, error) {
	return p.response, "api-call", p.err
}

func (p *appTestProvider) GenerateStream(_ context.Context, _ string, onChunk func(string)) (string, string, error) {
	if p.response != "" && onChunk != nil {
		onChunk(p.response)
	}
	return p.response, "api-call", p.err
}

func TestStartupInitializesFacadeServices(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app := NewApp()
	app.startup(context.Background())

	if app.contextGenerator == nil || app.llmService == nil || app.tokenService == nil {
		t.Fatal("expected startup to initialize core services")
	}
	_ = app.GetAutoContextButtonTexture()
}

func TestAutoContextParserAndServiceMethods(t *testing.T) {
	parser := autoContextParser{}
	if parser.Type() == "" {
		t.Fatal("expected parser type")
	}
	if parser.GetFormatInstructions() == "" {
		t.Fatal("expected non-empty format instructions")
	}

	parsed, err := parser.Parse(`{"files":["src/main.go"]}`)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(parsed.Files) != 1 {
		t.Fatalf("unexpected parsed files: %+v", parsed.Files)
	}

	parsed2, err := parser.ParseWithPrompt(`{"files":["src/main.go"]}`, nil)
	if err != nil {
		t.Fatalf("ParseWithPrompt failed: %v", err)
	}
	if len(parsed2.Files) != 1 {
		t.Fatalf("unexpected ParseWithPrompt files: %+v", parsed2.Files)
	}

	svc := NewAutoContextService()
	parsed3, err := svc.ParseResponse(`{"files":["src/main.go"]}`)
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}
	if len(parsed3.Files) != 1 {
		t.Fatalf("unexpected ParseResponse files: %+v", parsed3.Files)
	}
}

func TestRequestAutoContextSelectionUsesProviderAndResolvesFiles(t *testing.T) {
	app := newTestApp(t)
	app.autoContextService = NewAutoContextService()
	app.historyManager = NewHistoryManager(app)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	app.settings.LLMSettings = LLMSettings{
		ActiveProvider: LLMProviderOllama,
		Model:          "llama3.2",
		BaseURL:        "http://localhost:11434/v1",
	}
	cfg := buildProviderConfig(app.settings.LLMSettings)
	app.llmCache = cachedProvider{
		cfg: cfg,
		instance: &appTestProvider{
			response: `{"files":["src/main.go"],"reasoning":"needed"}`,
		},
	}

	selected, err := app.RequestAutoContextSelection(root, nil, "fix issue")
	if err != nil {
		t.Fatalf("RequestAutoContextSelection failed: %v", err)
	}
	if len(selected) != 1 || selected[0] != "src/main.go" {
		t.Fatalf("unexpected selected files: %+v", selected)
	}
	if len(app.historyManager.GetItems()) == 0 {
		t.Fatal("expected auto-context call to write history item")
	}
}

func TestRequestShotgunContextGenerationWithNilGenerator(t *testing.T) {
	app := newTestApp(t)
	app.contextGenerator = nil
	app.RequestShotgunContextGeneration(t.TempDir(), nil)
}

func TestRuntimeAdapterNoopWithoutWailsContext(t *testing.T) {
	ctx := context.Background()
	safeLogWarningf(ctx, "warn: %s", "x")
	safeLogError(ctx, "err")
	safeLogErrorf(ctx, "err: %d", 1)
	safeEventsEmit(ctx, "event", map[string]string{"k": "v"})
}
