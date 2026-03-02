// Package apikeysvm contains view model types for the API keys settings page.
// It is a leaf package (no internal imports) so that the templates package can
// import it without creating an import cycle with the apikeys service package.
package apikeysvm

// APIKeyViewModel is the display model for a single stored API key.
// The value is always masked — plaintext is never passed to templates.
type APIKeyViewModel struct {
	ID           uint
	KeyType      string
	Provider     string
	ProviderName string
	MaskedValue  string
}

// LLMProviderVM is the display model for a supported LLM provider.
type LLMProviderVM struct {
	ID     string
	Name   string
	Models []string
}

// APIKeysPageViewModel is the view model passed to the APIKeysSettingsPage template.
type APIKeysPageViewModel struct {
	LLMKeys           []APIKeyViewModel
	TavilyKey         *APIKeyViewModel // nil if not set
	Providers         []LLMProviderVM
	PreferredProvider string
	PreferredModel    string
}
