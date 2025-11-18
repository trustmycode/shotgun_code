package provider

import "fmt"

var openAIModelCatalog = []ModelInfo{
	// GPT-5 family (latest reasoning-capable models)
	{Name: "gpt-5.1", Description: "Latest GPT-5.1 flagship for complex reasoning and coding tasks"},
	{Name: "gpt-5", Description: "Previous GPT-5 flagship reasoning model"},
	{Name: "gpt-5-mini", Description: "Cost-optimized GPT-5 mini model"},
	{Name: "gpt-5-nano", Description: "High-throughput GPT-5 nano model"},

	// GPT-4 family
	{Name: "gpt-4o-mini", Description: "Latest GPT-4o mini for general reasoning"},
	{Name: "gpt-4.1-mini", Description: "GPT-4.1 mini tier"},
	{Name: "o4-mini", Description: "Reasoning optimized 04-mini"},
	{Name: "gpt-4o", Description: "Full GPT-4o"},
	{Name: "gpt-4.1", Description: "Full GPT-4.1"},
}

var openRouterModelCatalog = []ModelInfo{
	{Name: "openrouter/auto", Description: "Router automatically selects the best model"},
	{Name: "openrouter/gpt-5", Description: "GPT-5 family routed via OpenRouter"},
	{Name: "anthropic/claude-3.5-sonnet", Description: "Claude 3.5 Sonnet via OpenRouter"},
	{Name: "google/gemini-pro-1.5", Description: "Gemini Pro 1.5 proxied through OpenRouter"},
	{Name: "openai/gpt-4o-mini", Description: "GPT-4o mini from OpenRouter catalog"},
	{Name: "meta-llama/llama-3.1-70b-instruct", Description: "Llama 3.1 70B Instruct via OpenRouter"},
}

var geminiModelCatalog = []ModelInfo{
	{Name: "gemini-2.5-pro", Description: "Most capable Gemini 2.5 Pro"},
	{Name: "gemini-2.5-flash", Description: "Flash"},
}

func cloneModelCatalog(models []ModelInfo) []ModelInfo {
	if len(models) == 0 {
		return nil
	}
	out := make([]ModelInfo, len(models))
	copy(out, models)
	return out
}

// ModelCatalog returns a provider specific list of models without requiring the provider to be fully configured.
func ModelCatalog(providerName string) ([]ModelInfo, error) {
	switch providerName {
	case "openai":
		return cloneModelCatalog(openAIModelCatalog), nil
	case "openrouter":
		return cloneModelCatalog(openRouterModelCatalog), nil
	case "gemini":
		return cloneModelCatalog(geminiModelCatalog), nil
	default:
		return nil, fmt.Errorf("provider %s is not supported", providerName)
	}
}
