package templates

// Branding holds white-label branding values for use in templates.
type Branding struct {
	AppName          string
	Tagline          string
	TitleSuffix      string
	LogoSVG          string // resolved inline SVG, empty = use default
	ThemeStorageKey  string
	AccentLight      string // empty = use theme default
	AccentHoverLight string
	AccentDark       string
	AccentHoverDark  string
	CustomCSSPath    string // URL path, e.g. "/static/css/custom.css"
	FaviconPath      string // URL path, e.g. "/favicon.ico"
}
