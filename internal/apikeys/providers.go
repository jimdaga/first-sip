package apikeys

// LLMProvider defines a supported LLM provider and its available models
type LLMProvider struct {
	ID     string
	Name   string
	Models []string
}

// SupportedLLMProviders is the curated list of LLM providers available for selection
var SupportedLLMProviders = []LLMProvider{
	{
		ID:   "openai",
		Name: "OpenAI",
		Models: []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
			"gpt-3.5-turbo",
		},
	},
	{
		ID:   "anthropic",
		Name: "Anthropic",
		Models: []string{
			"claude-opus-4-5",
			"claude-sonnet-4-5",
			"claude-haiku-3-5",
		},
	},
	{
		ID:   "groq",
		Name: "Groq",
		Models: []string{
			"llama-3.3-70b-versatile",
			"llama-3.1-8b-instant",
			"mixtral-8x7b-32768",
		},
	},
}

// GetProviderByID returns a pointer to the LLMProvider matching the given ID, or nil if not found
func GetProviderByID(id string) *LLMProvider {
	for i := range SupportedLLMProviders {
		if SupportedLLMProviders[i].ID == id {
			return &SupportedLLMProviders[i]
		}
	}
	return nil
}

// GetAllModelsForProvider returns the list of models for the given provider ID, or nil if not found
func GetAllModelsForProvider(providerID string) []string {
	provider := GetProviderByID(providerID)
	if provider == nil {
		return nil
	}
	return provider.Models
}
