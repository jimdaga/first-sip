package templates

// SidebarPlugin holds the data needed to render a plugin sub-link in the sidebar.
type SidebarPlugin struct {
	PluginName  string // slug, e.g. "daily-news-digest"
	DisplayName string // e.g. "Daily News Digest"
	Icon        string // emoji
}
