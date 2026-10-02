package theme

import (
	"testing"

	"github.com/ows4444/tui/ansi"
)

func allPresets() map[string]Theme {
	return map[string]Theme{
		"dark": DarkTheme(), "light": LightTheme(), "dracula": DraculaTheme(), "nord": NordTheme(),
		"gruvbox": GruvboxTheme(), "tokyoNight": TokyoNightTheme(), "monokai": MonokaiTheme(),
		"solarizedDark": SolarizedDarkTheme(), "solarizedLight": SolarizedLightTheme(),
		"catppuccin": CatppuccinTheme(), "oneDark": OneDarkTheme(), "nightOwl": NightOwlTheme(),
		"rosePine": RosePineTheme(), "everforest": EverforestDarkTheme(), "github": GitHubDarkTheme(),
		"highContrast": HighContrastTheme(),
	}
}

func TestSurfaceRolesInEveryPreset(t *testing.T) {
	for name, th := range allPresets() {
		if th.Background == nil || th.Surface == nil || th.Overlay == nil {
			t.Errorf("%s: Background/Surface/Overlay not all set", name)
		}
		if tw := th.LightTwin(); tw.Background == nil || tw.Surface == nil || tw.Overlay == nil {
			t.Errorf("%s: LightTwin lost a surface role", name)
		}
		if p := th.ForProfile(ansi.TrueColor); p.Surface == nil {
			t.Errorf("%s: ForProfile dropped Surface", name)
		}
	}
}

func TestCheckUsesBackground(t *testing.T) {
	th := DarkTheme()
	th.TextInverse = th.Text // the old reference: text measured against itself
	th.Background = ansi.Black
	if got := th.Check(4.0); len(got) != 0 {
		t.Errorf("Check ignored Background: %v", got)
	}
	th.Background = nil
	if got := th.Check(4.0); len(got) == 0 {
		t.Error("Check without Background should fall back to TextInverse")
	}
	if got := DarkTheme().CheckOn(ansi.White, 4.5); len(got) == 0 {
		t.Error("CheckOn(White) should flag light text")
	}
}
