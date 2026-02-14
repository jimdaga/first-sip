package plugins

import (
	"fmt"
	"os"
	"strings"

	"github.com/kaptinlin/jsonschema"
)

// ValidateUserSettings validates user settings against a JSON Schema file
func ValidateUserSettings(schemaPath string, userSettings map[string]interface{}) error {
	// Read schema file
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Create compiler and compile schema
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(schemaData)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	// Validate user settings
	result := schema.Validate(userSettings)
	if !result.IsValid() {
		// Collect all validation errors
		var errorMessages []string
		for field, evalErr := range result.Errors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", field, evalErr.Error()))
		}
		return fmt.Errorf("settings validation failed: %s", strings.Join(errorMessages, "; "))
	}

	return nil
}
