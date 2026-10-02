package widgets

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestStatusIndicator(t *testing.T) {
	dt := theme.DarkTheme()
	tests := []struct {
		name    string
		label   string
		variant Variant
		color   ansi.Color
	}{
		{"neutral", "idle", VariantNeutral, dt.Muted},
		{"info", "queued", VariantInfo, dt.Info},
		{"success", "Running", VariantSuccess, dt.Success},
		{"warning", "Degraded", VariantWarning, dt.Warning},
		{"error", "Failed", VariantError, dt.Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusIndicator(tt.label, tt.variant, dt)
			want := ansi.NewStyle().Foreground(tt.color).Render(statusGlyph(tt.variant)) + " " + tt.label
			if got != want {
				t.Errorf("StatusIndicator(%q, %v, Dark) = %q, want %q", tt.label, tt.variant, got, want)
			}
		})
	}
}

// statusGlyph is the dot StatusIndicator draws under theme.DarkTheme(): the neutral
// dot, or the variant's icon so the status shows without colour.
func statusGlyph(v Variant) string {
	if m := v.Mark(theme.DarkTheme().GlyphSet()); m != "" {
		return m
	}
	return theme.DarkTheme().GlyphSet().Dot
}

func TestStatusIndicatorUsesGivenTheme(t *testing.T) {
	got := StatusIndicator("Running", VariantSuccess, theme.LightTheme())
	want := ansi.NewStyle().Foreground(theme.LightTheme().Success).Render(VariantSuccess.Icon(theme.LightTheme().GlyphSet())) + " Running"
	if got != want {
		t.Errorf("StatusIndicator(%q, VariantSuccess, Light) = %q, want %q (should use Light's colors, not Dark's)", "Running", got, want)
	}
}
