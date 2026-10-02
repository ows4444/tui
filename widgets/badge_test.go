package widgets

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestBadge(t *testing.T) {
	tt := theme.DarkTheme()
	tests := []struct {
		name    string
		text    string
		variant Variant
		color   ansi.Color
	}{
		{"neutral", "n/a", VariantNeutral, tt.Muted},
		{"info", "info", VariantInfo, tt.Info},
		{"success", "OK", VariantSuccess, tt.Success},
		{"warning", "warn", VariantWarning, tt.Warning},
		{"error", "FAIL", VariantError, tt.Error},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Badge(test.text, test.variant, tt)
			want := ansi.NewStyle().Bold().Background(test.color).Foreground(tt.TextInverse).Render(" " + markPrefix(test.variant, tt) + test.text + " ")
			if got != want {
				t.Errorf("Badge(%q, %v, Dark) = %q, want %q", test.text, test.variant, got, want)
			}
		})
	}
}

func TestBadgeUsesGivenTheme(t *testing.T) {
	got := Badge("OK", VariantSuccess, theme.LightTheme())
	want := ansi.NewStyle().Bold().Background(theme.LightTheme().Success).Foreground(theme.LightTheme().TextInverse).Render(" " + markPrefix(VariantSuccess, theme.LightTheme()) + "OK ")
	if got != want {
		t.Errorf("Badge(%q, VariantSuccess, Light) = %q, want %q (should use Light's colors, not Dark's)", "OK", got, want)
	}
}
