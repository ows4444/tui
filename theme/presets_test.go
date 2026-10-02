package theme

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// TestPresetThemesHaveNoZeroColors proves criterion #472: every preset
// theme sets all 12 color fields plus Border to a non-zero value, matching
// Dark/Light's complete-struct convention.
func TestPresetThemesHaveNoZeroColors(t *testing.T) {
	tests := []struct {
		name  string
		theme Theme
	}{
		{"Dracula", DraculaTheme()},
		{"Nord", NordTheme()},
		{"Gruvbox", GruvboxTheme()},
		{"TokyoNight", TokyoNightTheme()},
		{"Monokai", MonokaiTheme()},
		{"SolarizedDark", SolarizedDarkTheme()},
		{"SolarizedLight", SolarizedLightTheme()},
		{"Catppuccin", CatppuccinTheme()},
		{"OneDark", OneDarkTheme()},
		{"NightOwl", NightOwlTheme()},
		{"RosePine", RosePineTheme()},
		{"EverforestDark", EverforestDarkTheme()},
		{"GitHubDark", GitHubDarkTheme()},
		{"HighContrast", HighContrastTheme()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := map[string]any{
				"Primary":     tt.theme.Primary,
				"Secondary":   tt.theme.Secondary,
				"Success":     tt.theme.Success,
				"Warning":     tt.theme.Warning,
				"Error":       tt.theme.Error,
				"Info":        tt.theme.Info,
				"Muted":       tt.theme.Muted,
				"Text":        tt.theme.Text,
				"TextInverse": tt.theme.TextInverse,
				"BorderColor": tt.theme.BorderColor,
				"Focus":       tt.theme.Focus,
				"Selection":   tt.theme.Selection,
			}
			for name, v := range fields {
				if v == nil {
					t.Errorf("%s.%s is nil", tt.name, name)
				}
			}
			if tt.theme.Border == (layout.Border{}) {
				t.Errorf("%s.Border is the zero value", tt.name)
			}
		})
	}
}

// TestPresetThemesUseRGBPaletteColors proves criterion #473: every color
// field on every preset is an ansi.RGB value (not a named ANSI-16 color),
// since the presets are documented as sourced from each theme's real,
// published truecolor hex palette rather than invented or approximated
// with the 16-color ANSI set.
func TestPresetThemesUseRGBPaletteColors(t *testing.T) {
	tests := []struct {
		name  string
		theme Theme
	}{
		{"Dracula", DraculaTheme()},
		{"Nord", NordTheme()},
		{"Gruvbox", GruvboxTheme()},
		{"TokyoNight", TokyoNightTheme()},
		{"Monokai", MonokaiTheme()},
		{"SolarizedDark", SolarizedDarkTheme()},
		{"SolarizedLight", SolarizedLightTheme()},
		{"Catppuccin", CatppuccinTheme()},
		{"OneDark", OneDarkTheme()},
		{"NightOwl", NightOwlTheme()},
		{"RosePine", RosePineTheme()},
		{"EverforestDark", EverforestDarkTheme()},
		{"GitHubDark", GitHubDarkTheme()},
		{"HighContrast", HighContrastTheme()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := map[string]any{
				"Primary":     tt.theme.Primary,
				"Secondary":   tt.theme.Secondary,
				"Success":     tt.theme.Success,
				"Warning":     tt.theme.Warning,
				"Error":       tt.theme.Error,
				"Info":        tt.theme.Info,
				"Muted":       tt.theme.Muted,
				"Text":        tt.theme.Text,
				"TextInverse": tt.theme.TextInverse,
				"BorderColor": tt.theme.BorderColor,
				"Focus":       tt.theme.Focus,
				"Selection":   tt.theme.Selection,
			}
			for name, v := range fields {
				if _, ok := v.(ansi.RGB); !ok {
					t.Errorf("%s.%s = %#v, want an ansi.RGB (real published palette value)", tt.name, name, v)
				}
			}
		})
	}
}
