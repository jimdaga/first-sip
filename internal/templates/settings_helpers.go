package templates

import "strings"

// isChecked reports whether option is selected in currentValue.
// currentValue may be a JSON array string (e.g. `["tech","science"]`) or a
// comma-separated string (e.g. "tech,science").  Both formats are checked so
// that saved DB values (JSON-marshalled []string) and raw submitted form values
// (comma-joined or multi-param) all work correctly.
func isChecked(option, currentValue string) bool {
	if currentValue == "" {
		return false
	}
	// JSON array form: ["tech","science"]
	trimmed := strings.TrimSpace(currentValue)
	if strings.HasPrefix(trimmed, "[") {
		// Strip brackets and quotes, then split by comma.
		inner := strings.Trim(trimmed, "[]")
		parts := strings.Split(inner, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			p = strings.Trim(p, `"`)
			if p == option {
				return true
			}
		}
		return false
	}
	// Comma-separated form (from multi-checkbox submit joined as one value).
	parts := strings.Split(currentValue, ",")
	for _, p := range parts {
		if strings.TrimSpace(p) == option {
			return true
		}
	}
	return false
}
