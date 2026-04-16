package detect

import "os"

// ProviderInfo holds the name and availability of an AI provider.
type ProviderInfo struct {
	Name      string
	Available bool
}

// DetectProviders checks environment variables for known AI providers and
// returns a slice of ProviderInfo indicating which are configured.
func DetectProviders() []ProviderInfo {
	providers := []ProviderInfo{
		{Name: "anthropic", Available: os.Getenv("ANTHROPIC_API_KEY") != ""},
		{Name: "openai", Available: os.Getenv("OPENAI_API_KEY") != ""},
		{Name: "google", Available: os.Getenv("GOOGLE_API_KEY") != "" || os.Getenv("GEMINI_API_KEY") != ""},
	}
	return providers
}

// GetAvailableProviderNames returns the names of all available providers from
// the given slice.
func GetAvailableProviderNames(providers []ProviderInfo) []string {
	var names []string
	for _, p := range providers {
		if p.Available {
			names = append(names, p.Name)
		}
	}
	return names
}
