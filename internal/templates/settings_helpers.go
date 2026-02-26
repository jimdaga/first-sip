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

// sidebarLinkClass returns the CSS class string for a sidebar nav link.
// When pageName matches activePage, the active modifier class is included.
// For "settings", any activePage starting with "settings" is considered active.
// For "dashboard", any activePage starting with "dashboard" is considered active.
func sidebarLinkClass(activePage, pageName string) string {
	active := false
	switch pageName {
	case "settings":
		active = isSettingsActive(activePage)
	case "dashboard":
		active = isDashboardActive(activePage)
	default:
		active = activePage == pageName
	}
	if active {
		return "sidebar-link sidebar-link-active"
	}
	return "sidebar-link"
}

// isSettingsActive returns true when activePage is "settings" or any settings sub-page
// (e.g. "settings-plugins").
func isSettingsActive(activePage string) bool {
	return activePage == "settings" || strings.HasPrefix(activePage, "settings-")
}

// settingsSubLinkClass returns the CSS class for a settings sub-link.
// sub should be the suffix after "settings-", e.g. "plugins".
func settingsSubLinkClass(activePage, sub string) string {
	if activePage == "settings-"+sub {
		return "sidebar-sub-link sidebar-sub-link-active"
	}
	return "sidebar-sub-link"
}

// isDashboardActive returns true when activePage is "dashboard" or any dashboard sub-page
// (e.g. "dashboard-daily-news-digest").
func isDashboardActive(activePage string) bool {
	return activePage == "dashboard" || strings.HasPrefix(activePage, "dashboard-")
}

// dashboardSubLinkClass returns the CSS class for a dashboard plugin sub-link.
// pluginName should be the plugin slug, e.g. "daily-news-digest".
func dashboardSubLinkClass(activePage, pluginName string) string {
	if activePage == "dashboard-"+pluginName {
		return "sidebar-sub-link sidebar-sub-link-active"
	}
	return "sidebar-sub-link"
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
