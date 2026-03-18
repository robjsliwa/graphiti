package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_BrandingDefaults(t *testing.T) {
	// Load with no config file — should get branding defaults
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Branding.AppName != "Graphiti" {
		t.Errorf("AppName = %q, want %q", cfg.Branding.AppName, "Graphiti")
	}
	if cfg.Branding.Tagline != "Visual Workflow Builder" {
		t.Errorf("Tagline = %q, want %q", cfg.Branding.Tagline, "Visual Workflow Builder")
	}
	if cfg.Branding.TitleSuffix != "Graphiti" {
		t.Errorf("TitleSuffix = %q, want %q", cfg.Branding.TitleSuffix, "Graphiti")
	}
	if cfg.Branding.ThemeStorageKey != "graphiti-theme" {
		t.Errorf("ThemeStorageKey = %q, want %q", cfg.Branding.ThemeStorageKey, "graphiti-theme")
	}
	// Logo, colors, favicon, customCSS should be empty
	if cfg.Branding.Logo.SVG != "" {
		t.Errorf("Logo.SVG = %q, want empty", cfg.Branding.Logo.SVG)
	}
	if cfg.Branding.Favicon != "" {
		t.Errorf("Favicon = %q, want empty", cfg.Branding.Favicon)
	}
}

func TestLoad_BrandingCustomValues(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "app.yaml")

	yaml := `
branding:
  appName: "Acme Workflows"
  tagline: "Internal Pipeline Builder"
  titleSuffix: "Acme"
  logo:
    svg: "<svg>custom</svg>"
  favicon: "./branding/favicon.ico"
  themeStorageKey: "acme-theme"
  colors:
    accentLight: "#E11D48"
    accentHoverLight: "#BE123C"
    accentDark: "#FB7185"
    accentHoverDark: "#F43F5E"
  customCSS: "./branding/custom.css"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	b := cfg.Branding
	if b.AppName != "Acme Workflows" {
		t.Errorf("AppName = %q, want %q", b.AppName, "Acme Workflows")
	}
	if b.Tagline != "Internal Pipeline Builder" {
		t.Errorf("Tagline = %q, want %q", b.Tagline, "Internal Pipeline Builder")
	}
	if b.TitleSuffix != "Acme" {
		t.Errorf("TitleSuffix = %q, want %q", b.TitleSuffix, "Acme")
	}
	if b.Logo.SVG != "<svg>custom</svg>" {
		t.Errorf("Logo.SVG = %q, want %q", b.Logo.SVG, "<svg>custom</svg>")
	}
	if b.Favicon != "./branding/favicon.ico" {
		t.Errorf("Favicon = %q, want %q", b.Favicon, "./branding/favicon.ico")
	}
	if b.ThemeStorageKey != "acme-theme" {
		t.Errorf("ThemeStorageKey = %q, want %q", b.ThemeStorageKey, "acme-theme")
	}
	if b.Colors.AccentLight != "#E11D48" {
		t.Errorf("AccentLight = %q, want %q", b.Colors.AccentLight, "#E11D48")
	}
	if b.Colors.AccentHoverLight != "#BE123C" {
		t.Errorf("AccentHoverLight = %q, want %q", b.Colors.AccentHoverLight, "#BE123C")
	}
	if b.Colors.AccentDark != "#FB7185" {
		t.Errorf("AccentDark = %q, want %q", b.Colors.AccentDark, "#FB7185")
	}
	if b.Colors.AccentHoverDark != "#F43F5E" {
		t.Errorf("AccentHoverDark = %q, want %q", b.Colors.AccentHoverDark, "#F43F5E")
	}
	if b.CustomCSS != "./branding/custom.css" {
		t.Errorf("CustomCSS = %q, want %q", b.CustomCSS, "./branding/custom.css")
	}
}

func TestLoad_BrandingEnvVarExpansion(t *testing.T) {
	t.Setenv("ACME_APP_NAME", "EnvApp")
	t.Setenv("ACME_TAGLINE", "EnvTagline")

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "app.yaml")

	yaml := `
branding:
  appName: "${ACME_APP_NAME}"
  tagline: "${ACME_TAGLINE}"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Branding.AppName != "EnvApp" {
		t.Errorf("AppName = %q, want %q", cfg.Branding.AppName, "EnvApp")
	}
	if cfg.Branding.Tagline != "EnvTagline" {
		t.Errorf("Tagline = %q, want %q", cfg.Branding.Tagline, "EnvTagline")
	}
}
