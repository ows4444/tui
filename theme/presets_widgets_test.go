package theme_test

import (
	"testing"

	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// TestPresetThemesRenderWidgetsWithoutPanicking proves criterion #474:
// existing widgets render without panicking under each new preset, the
// same smoke-test guarantee Dark/Light already get. This lives in the
// theme_test external test package (rather than package theme, alongside
// presets_test.go) because widgets imports theme, so a theme-package test
// importing widgets would be an import cycle.
func TestPresetThemesRenderWidgetsWithoutPanicking(t *testing.T) {
	presets := []struct {
		name  string
		theme theme.Theme
	}{
		{"Dracula", theme.DraculaTheme()},
		{"Nord", theme.NordTheme()},
		{"Gruvbox", theme.GruvboxTheme()},
		{"TokyoNight", theme.TokyoNightTheme()},
		{"Monokai", theme.MonokaiTheme()},
		{"SolarizedDark", theme.SolarizedDarkTheme()},
		{"SolarizedLight", theme.SolarizedLightTheme()},
		{"Catppuccin", theme.CatppuccinTheme()},
		{"OneDark", theme.OneDarkTheme()},
		{"NightOwl", theme.NightOwlTheme()},
		{"RosePine", theme.RosePineTheme()},
		{"EverforestDark", theme.EverforestDarkTheme()},
		{"GitHubDark", theme.GitHubDarkTheme()},
		{"HighContrast", theme.HighContrastTheme()},
	}
	variants := []widgets.Variant{
		widgets.VariantNeutral,
		widgets.VariantInfo,
		widgets.VariantSuccess,
		widgets.VariantWarning,
		widgets.VariantError,
	}
	for _, p := range presets {
		t.Run(p.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panicked rendering widgets under %s: %v", p.name, r)
				}
			}()
			for _, v := range variants {
				if out := widgets.Badge("status", v, p.theme); out == "" {
					t.Errorf("Badge under %s returned empty output", p.name)
				}
				if out := widgets.Alert("message", v, p.theme, 40); out == "" {
					t.Errorf("Alert under %s returned empty output", p.name)
				}
			}
			if out := widgets.Panel("title", "content", p.theme, 40); out == "" {
				t.Errorf("Panel under %s returned empty output", p.name)
			}
		})
	}
}
