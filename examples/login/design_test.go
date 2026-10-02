package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func toggle(m model) model {
	m, _ = send(m, tui.Key{Type: tui.KeyCtrl, Code: 't'})
	return m
}

func TestCtrlTMovesKeyboardToDesignAndBack(t *testing.T) {
	m := toggle(initialModel())
	if !m.designing || m.form.Focused() {
		t.Fatalf("designing=%v formFocused=%v, want the design panel to have the keyboard", m.designing, m.form.Focused())
	}
	m = typed(m, "zzz")
	if m.form.Values()["user"] != "" {
		t.Error("typing reached the form while the design panel had the keyboard")
	}
	m, _ = key(m, tui.KeyEsc) // Esc goes back rather than quitting
	if m.designing || !m.form.Focused() {
		t.Errorf("after Esc designing=%v formFocused=%v, want the form back", m.designing, m.form.Focused())
	}
	m, _ = send(m, tui.Key{Type: tui.KeyF2})
	if !m.designing {
		t.Error("F2 did not open the design panel")
	}
}

func TestChangingThemeRethemesWidgets(t *testing.T) {
	m := toggle(initialModel())
	m, _ = key(m, tui.KeyRight) // Theme: Dark -> Light
	if m.t.Primary != theme.LightTheme().Primary {
		t.Errorf("theme primary = %v, want Light's", m.t.Primary)
	}
	if m.form.Theme.Primary != m.t.Primary || m.spin.Theme.Primary != m.t.Primary {
		t.Error("the form or spinner kept the old theme")
	}
	if !strings.Contains(plain(m), "◂ Light ▸") {
		t.Errorf("the panel should show the chosen theme, got:\n%s", plain(m))
	}
}

func TestBorderAndResetSettings(t *testing.T) {
	m := toggle(initialModel())
	m, _ = key(m, tui.KeyDown)
	m, _ = key(m, tui.KeyDown) // Border
	m, _ = key(m, tui.KeyLeft) // wraps to None
	if m.t.Border != (layout.Border{}) {
		t.Errorf("border = %v, want none", m.t.Border)
	}
	m, _ = key(m, tui.KeyLeft) // ASCII
	if m.t.Border != layout.ASCIIBorder() || !strings.Contains(plain(m), "+---") {
		t.Errorf("want ASCII borders, got:\n%s", plain(m))
	}
	m = typed(m, "r")
	if m.design.pick != (design{}).pick || m.t.Border != theme.DarkTheme().Border {
		t.Errorf("reset left pick=%v border=%v", m.design.pick, m.t.Border)
	}
}

func TestAccentIsAComponentToken(t *testing.T) {
	var d design
	d.pick[setAccent] = 4 // "Theme", Primary, Secondary, Success, Warning
	th := d.theme()
	if got := th.TokensFor(theme.ComponentForm).Focus; got != th.Warning {
		t.Errorf("form focus token = %v, want Warning %v", got, th.Warning)
	}
	if th.Focus != theme.DarkTheme().Focus {
		t.Error("the accent leaked into the theme-wide Focus role")
	}
}

func TestNoColorDepthDropsColours(t *testing.T) {
	var d design
	d.pick[setDepth] = len(profiles) - 1
	if th := d.theme(); th.Primary != nil || th.Text != nil {
		t.Errorf("No color kept colours: primary=%v text=%v", th.Primary, th.Text)
	}
}

// TestEveryOptionRenders draws the screen with each option of each setting
// at a wide and a narrow size, and checks no line overflows the narrow one.
func TestEveryOptionRenders(t *testing.T) {
	for s := setting(0); s < numSettings; s++ {
		for i := range options(s) {
			for _, designing := range []bool{false, true} {
				m := initialModel()
				m.design.pick[s] = i
				m.design.cursor = s
				m.designing = designing
				m.restyle()
				if strings.TrimSpace(plain(m)) == "" {
					t.Errorf("%s=%s: empty view", settingNames[s], options(s)[i])
				}
				m, _ = send(m, tui.ResizeMsg{Width: 50, Height: 40})
				for _, line := range strings.Split(m.View(), "\n") {
					if w := ansi.Width(line); w > 50 {
						t.Errorf("%s=%s designing=%v: line %d wide: %q", settingNames[s], options(s)[i], designing, w, ansi.StripANSI(line))
						break
					}
				}
			}
		}
	}
}

func TestNarrowShowsOnlyThePanelWithTheKeyboard(t *testing.T) {
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 60, Height: 30})
	if strings.Contains(plain(m), "Design system") {
		t.Error("narrow login view should not show the design panel")
	}
	m = toggle(m)
	if v := plain(m); !strings.Contains(v, "Design system") || strings.Contains(v, "Password") {
		t.Errorf("narrow design view should show only the design panel, got:\n%s", v)
	}
}
