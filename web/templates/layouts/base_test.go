package layouts

import (
	"strings"
	"testing"

	"graphiti/web/templates"
)

func TestSafeColor_ValidHex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"#E11D48", "#E11D48"},
		{"#fff", "#fff"},
		{"#00FF00FF", "#00FF00FF"},
		{"", ""},
		{"red", ""},                              // named colors rejected
		{"#E11D48; } body { display:none", ""},   // injection attempt
		{"rgb(255,0,0)", ""},                      // non-hex rejected
	}
	for _, tt := range tests {
		got := safeColor(tt.input)
		if got != tt.want {
			t.Errorf("safeColor(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuildAccentStyleTag(t *testing.T) {
	b := templates.Branding{
		AccentLight:      "#E11D48",
		AccentHoverLight: "#BE123C",
		AccentDark:       "#FB7185",
		AccentHoverDark:  "#F43F5E",
	}
	result := buildAccentStyleTag(b)

	if !strings.HasPrefix(result, "<style>") {
		t.Error("expected result to start with <style>")
	}
	if !strings.HasSuffix(result, "</style>") {
		t.Error("expected result to end with </style>")
	}
	if !strings.Contains(result, "--accent:#E11D48;") {
		t.Error("expected light accent color in output")
	}
	if !strings.Contains(result, "--accent-hover:#BE123C;") {
		t.Error("expected light accent hover in output")
	}
	if !strings.Contains(result, `[data-theme="dark"]`) {
		t.Error("expected dark theme selector in output")
	}
}

func TestBuildAccentStyleTag_Injection(t *testing.T) {
	b := templates.Branding{
		AccentLight: `#E11D48; } body { display:none`,
	}
	result := buildAccentStyleTag(b)
	// Invalid color should be rejected, so no CSS properties emitted
	if strings.Contains(result, "display:none") {
		t.Error("CSS injection should be blocked by safeColor validation")
	}
}

func TestSafeStorageKey_Valid(t *testing.T) {
	b := templates.Branding{ThemeStorageKey: "acme-theme"}
	if got := safeStorageKey(b); got != "acme-theme" {
		t.Errorf("safeStorageKey = %q, want %q", got, "acme-theme")
	}
}

func TestSafeStorageKey_Invalid(t *testing.T) {
	b := templates.Branding{ThemeStorageKey: "acme')+alert(1)+'"}
	if got := safeStorageKey(b); got != "graphiti-theme" {
		t.Errorf("safeStorageKey = %q, want fallback %q", got, "graphiti-theme")
	}
}

func TestSafeStorageKey_Empty(t *testing.T) {
	b := templates.Branding{}
	if got := safeStorageKey(b); got != "graphiti-theme" {
		t.Errorf("safeStorageKey = %q, want fallback %q", got, "graphiti-theme")
	}
}
