package templates

import "testing"

func TestBranding_ZeroValueIsUsable(t *testing.T) {
	var b Branding
	if b.AppName != "" {
		t.Errorf("AppName = %q, want empty", b.AppName)
	}
	if b.Tagline != "" {
		t.Errorf("Tagline = %q, want empty", b.Tagline)
	}
	if b.TitleSuffix != "" {
		t.Errorf("TitleSuffix = %q, want empty", b.TitleSuffix)
	}
	if b.LogoSVG != "" {
		t.Errorf("LogoSVG = %q, want empty", b.LogoSVG)
	}
	if b.ThemeStorageKey != "" {
		t.Errorf("ThemeStorageKey = %q, want empty", b.ThemeStorageKey)
	}
}
