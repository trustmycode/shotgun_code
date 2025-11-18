package provider

import "strings"

// isGPT5FamilyModel reports whether the given model name belongs to the GPT-5 family.
// It matches plain names like "gpt-5.1" as well as vendor-prefixed names like "openrouter/gpt-5".
func isGPT5FamilyModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return false
	}

	if strings.HasPrefix(m, "gpt-5") {
		return true
	}

	// Handle vendor-prefixed variants: "<vendor>/<model>"
	slash := strings.IndexByte(m, '/')
	if slash <= 0 || slash+1 >= len(m) {
		return false
	}

	return strings.HasPrefix(m[slash+1:], "gpt-5")
}


