package templates

import (
	"fmt"
	"strings"
)

// timeSlots returns a slice of HH:MM strings at 30-minute intervals (00:00 to 23:30).
// Used by renderTimeSelectField to populate the time picker dropdown.
func timeSlots() []string {
	slots := make([]string, 0, 48)
	for h := 0; h < 24; h++ {
		for _, m := range []int{0, 30} {
			slots = append(slots, fmt.Sprintf("%02d:%02d", h, m))
		}
	}
	return slots
}

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
