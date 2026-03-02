package apikeys

import (
	"github.com/jimdaga/first-sip/internal/apikeysvm"
	"github.com/jimdaga/first-sip/internal/models"
	"gorm.io/gorm"
)

// BuildViewModel constructs an APIKeysPageViewModel for the given user.
// It fetches all API keys for the user, separates them by type, masks the values,
// and maps provider IDs to human-readable names.
func BuildViewModel(db *gorm.DB, user *models.User) apikeysvm.APIKeysPageViewModel {
	keys, _ := GetKeysForUser(db, user.ID)

	// Convert SupportedLLMProviders to view model type.
	providers := make([]apikeysvm.LLMProviderVM, len(SupportedLLMProviders))
	for i, p := range SupportedLLMProviders {
		providers[i] = apikeysvm.LLMProviderVM{
			ID:     p.ID,
			Name:   p.Name,
			Models: p.Models,
		}
	}

	vm := apikeysvm.APIKeysPageViewModel{
		LLMKeys:           []apikeysvm.APIKeyViewModel{},
		Providers:         providers,
		PreferredProvider: user.LLMPreferredProvider,
		PreferredModel:    user.LLMPreferredModel,
	}

	for _, key := range keys {
		providerName := key.Provider
		if p := GetProviderByID(key.Provider); p != nil {
			providerName = p.Name
		}

		kvm := apikeysvm.APIKeyViewModel{
			ID:           key.ID,
			KeyType:      key.KeyType,
			Provider:     key.Provider,
			ProviderName: providerName,
			MaskedValue:  MaskAPIKey(key.EncryptedValue),
		}

		switch key.KeyType {
		case "llm":
			vm.LLMKeys = append(vm.LLMKeys, kvm)
		case "tavily":
			kvm.ProviderName = "Tavily"
			kvmCopy := kvm
			vm.TavilyKey = &kvmCopy
		}
	}

	return vm
}
