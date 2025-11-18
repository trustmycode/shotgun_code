package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"

	"shotgun_code/internal/llm/provider"
)

//go:embed design/prompts/contextPreparation.md
var embeddedPromptFS embed.FS

const (
	autoContextTemplatePath = "design/prompts/contextPreparation.md"
	maxAutoContextTreeChars = 15_000
)

var errAutoContextTreeTooLarge = errors.New("auto context file tree exceeds the allowed size")

type autoContextParser struct{}

type AutoContextResult struct {
	Files     []string `json:"files"`
	Reasoning string   `json:"reasoning,omitempty"`
}

func (autoContextParser) Parse(text string) (AutoContextResult, error) {
	return parseAutoContextJSON(text)
}

func (autoContextParser) ParseWithPrompt(text string, _ schema.PromptValue) (AutoContextResult, error) {
	return parseAutoContextJSON(text)
}

func (autoContextParser) GetFormatInstructions() string {
	return "Respond ONLY with a JSON object that matches this schema:\n" +
		"```\n{\n  \"files\": [\"relative/path/from/project/root\"],\n  \"reasoning\": \"optional short description\"\n}\n```\n" +
		"No code fences, commentary, or explanations outside the JSON object."
}

func (autoContextParser) Type() string {
	return "auto_context_json_parser"
}

type AutoContextService struct {
	parser         autoContextParser
	templateMu     sync.Mutex
	template       prompts.PromptTemplate
	templateLoaded bool
}

func NewAutoContextService() *AutoContextService {
	return &AutoContextService{}
}

func (s *AutoContextService) ensureTemplate() error {
	s.templateMu.Lock()
	defer s.templateMu.Unlock()
	if s.templateLoaded {
		return nil
	}

	var templateBody string
	if bytes, err := os.ReadFile(autoContextTemplatePath); err == nil {
		templateBody = string(bytes)
	} else {
		content, readErr := embeddedPromptFS.ReadFile(autoContextTemplatePath)
		if readErr != nil {
			return fmt.Errorf("failed to load auto-context prompt template: %w", readErr)
		}
		templateBody = string(content)
	}

	s.template = prompts.NewPromptTemplate(
		templateBody,
		[]string{"FILE_TREE", "USER_TASK", "CURRENT_UNDERSTANDING"},
	)
	s.templateLoaded = true
	return nil
}

func (s *AutoContextService) BuildPrompt(fileTree, userTask, understanding string) (string, error) {
	if err := s.ensureTemplate(); err != nil {
		return "", err
	}

	formatted, err := s.template.Format(map[string]any{
		"FILE_TREE":             fileTree,
		"USER_TASK":             userTask,
		"CURRENT_UNDERSTANDING": understanding,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render auto-context prompt: %w", err)
	}

	return strings.TrimSpace(formatted) + "\n\n" + s.parser.GetFormatInstructions(), nil
}

func (s *AutoContextService) ParseResponse(text string) (AutoContextResult, error) {
	return s.parser.Parse(text)
}

func parseAutoContextJSON(text string) (AutoContextResult, error) {
	cleaned := strings.TrimSpace(text)
	if cleaned == "" {
		return AutoContextResult{}, errors.New("empty response from LLM")
	}

	// Strip markdown fences if present.
	if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimPrefix(cleaned, "```JSON")
		cleaned = strings.TrimPrefix(cleaned, "```")
		if idx := strings.LastIndex(cleaned, "```"); idx >= 0 {
			cleaned = cleaned[:idx]
		}
	}
	cleaned = strings.TrimSpace(cleaned)

	var result AutoContextResult
	decoder := json.NewDecoder(strings.NewReader(cleaned))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return AutoContextResult{}, fmt.Errorf("failed to decode auto-context response: %w", err)
	}

	normalized := make([]string, 0, len(result.Files))
	for _, f := range result.Files {
		f = normalizeRelativePath(f)
		if f != "" {
			normalized = append(normalized, f)
		}
	}
	if len(normalized) == 0 {
		return AutoContextResult{}, errors.New("response did not include any valid files")
	}
	result.Files = normalized
	return result, nil
}

func buildAutoContextTree(rootDir string, excludedMap map[string]bool) (string, error) {
	var builder strings.Builder
	builder.WriteString(filepath.Base(rootDir) + string(os.PathSeparator) + "\n")

	var walk func(string, string) error
	walk = func(currentPath, prefix string) error {
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", currentPath, err)
		}
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].IsDir() && !entries[j].IsDir() {
				return true
			}
			if !entries[i].IsDir() && entries[j].IsDir() {
				return false
			}
			return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
		})

		visibleEntries := make([]os.DirEntry, 0, len(entries))
		for _, entry := range entries {
			relPath, _ := filepath.Rel(rootDir, filepath.Join(currentPath, entry.Name()))
			if excludedMap[normalizeRelativePath(relPath)] {
				continue
			}
			visibleEntries = append(visibleEntries, entry)
		}

		for idx, entry := range visibleEntries {
			branch := "├── "
			nextPrefix := prefix + "│   "
			if idx == len(visibleEntries)-1 {
				branch = "└── "
				nextPrefix = prefix + "    "
			}
			builder.WriteString(prefix + branch + entry.Name() + "\n")
			if builder.Len() > maxAutoContextTreeChars {
				return errAutoContextTreeTooLarge
			}

			if entry.IsDir() {
				if err := walk(filepath.Join(currentPath, entry.Name()), nextPrefix); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := walk(rootDir, ""); err != nil {
		return "", err
	}
	if builder.Len() > maxAutoContextTreeChars {
		return "", errAutoContextTreeTooLarge
	}
	return builder.String(), nil
}

func normalizeRelativePath(rel string) string {
	rel = strings.TrimSpace(rel)
	if rel == "" || rel == "." {
		return ""
	}

	rel = strings.TrimPrefix(rel, "./")
	rel = filepath.ToSlash(rel)
	rel = strings.TrimPrefix(rel, "/")
	return rel
}

func resolveLLMSelection(rootDir string, candidates []string) ([]string, error) {
	if len(candidates) == 0 {
		return nil, errors.New("no candidate paths provided")
	}
	selected := make(map[string]struct{})
	for _, candidate := range candidates {
		candidate = normalizeRelativePath(candidate)
		if candidate == "" {
			continue
		}
		absPath := filepath.Join(rootDir, filepath.FromSlash(candidate))
		info, err := os.Stat(absPath)
		if err != nil {
			continue
		}
		if info.IsDir() {
			filepath.WalkDir(absPath, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					return nil
				}
				rel, relErr := filepath.Rel(rootDir, path)
				if relErr != nil {
					return nil
				}
				rel = normalizeRelativePath(rel)
				if rel != "" {
					selected[rel] = struct{}{}
				}
				return nil
			})
		} else {
			selected[filepath.ToSlash(candidate)] = struct{}{}
		}
	}

	if len(selected) == 0 {
		return nil, errors.New("no existing files matched the LLM selection")
	}

	sorted := make([]string, 0, len(selected))
	for rel := range selected {
		sorted = append(sorted, rel)
	}
	sort.Strings(sorted)
	return sorted, nil
}

func buildProviderConfig(settings LLMSettings) provider.Config {
	return provider.Config{
		Provider: settings.ActiveProvider,
		Model:    fallbackModel(settings),
		APIKey:   settings.keyForProvider(settings.ActiveProvider),
		BaseURL:  strings.TrimSpace(settings.BaseURL),
	}
}

func fallbackModel(settings LLMSettings) string {
	model := strings.TrimSpace(settings.Model)
	if model != "" {
		return model
	}
	return defaultModelForProvider(settings.ActiveProvider)
}
