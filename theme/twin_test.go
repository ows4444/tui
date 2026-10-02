package theme

import (
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestAutoPicksLightTwinThatPassesCheck(t *testing.T) {
	presets := map[string]Theme{
		"Dark": DarkTheme(), "Dracula": DraculaTheme(), "Nord": NordTheme(), "Gruvbox": GruvboxTheme(),
		"TokyoNight": TokyoNightTheme(), "Monokai": MonokaiTheme(), "SolarizedDark": SolarizedDarkTheme(),
		"Catppuccin": CatppuccinTheme(), "OneDark": OneDarkTheme(), "NightOwl": NightOwlTheme(),
		"RosePine": RosePineTheme(), "EverforestDark": EverforestDarkTheme(),
		"GitHubDark": GitHubDarkTheme(), "HighContrast": HighContrastTheme(),
	}
	lightBG, darkBG := ansi.RGB{R: 0xfa, G: 0xfa, B: 0xfa}, ansi.RGB{R: 0x10, G: 0x10, B: 0x10}
	for name, p := range presets {
		t.Run(name, func(t *testing.T) {
			got := Pair(p).For(lightBG)
			if got.Text == p.Text {
				t.Fatal("light background did not select a different twin")
			}
			if issues := got.Check(4.5); len(issues) != 0 {
				t.Fatalf("light twin fails Check(4.5): %v", issues)
			}
			if got.Border != p.Border {
				t.Error("twin changed Border")
			}
			if Pair(p).For(darkBG) != p {
				t.Error("dark background should keep the preset")
			}
		})
	}
}

func TestLightTwinOfLightThemeIsIdentity(t *testing.T) {
	if SolarizedLightTheme().LightTwin() != SolarizedLightTheme() {
		t.Fatal("light theme should be returned unchanged")
	}
}
