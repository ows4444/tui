package theme

import (
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestComponentTokensOverrideWithoutForkingTheme(t *testing.T) {
	base := DarkTheme()
	red := ansi.RGB{R: 255}
	th := base.WithTokens("toast", Tokens{Accent: red, Border: red})

	if got := th.TokensFor("toast"); got.Accent != red || got.Border != red {
		t.Errorf("toast tokens = %+v, want accent and border overridden", got)
	}
	if got := th.TokensFor("toast"); got.Text != base.Text || got.Surface != base.Surface {
		t.Error("unset token fields must resolve from the theme roles")
	}
	if got := th.TokensFor("tabs"); got.Accent != base.Primary {
		t.Error("another component must keep the theme's roles")
	}
	if got := th.ForComponent("toast"); got.Primary != red || got.BorderColor != red {
		t.Error("ForComponent did not apply the overrides")
	}
	if th.Primary != base.Primary || base.Components != nil {
		t.Error("WithTokens modified the shared theme")
	}
	th2 := th.WithTokens("tabs", Tokens{Focus: red})
	if _, ok := th.Components.m["tabs"]; ok {
		t.Error("WithTokens shared its map with the receiver")
	}
	if th2.TokensFor("toast").Accent != red {
		t.Error("earlier overrides lost")
	}
}
