package rating

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func digit(s string) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: s} }

func changed(cmd tui.Cmd) (ChangedMsg, bool) {
	if cmd == nil {
		return ChangedMsg{}, false
	}
	c, ok := cmd().(ChangedMsg)
	return c, ok
}

func plain(m Model) string { return ansi.StripANSI(m.View()) }

func stars() Model {
	m := New(5)
	m.ID = "stars"
	m.Focus()
	return m
}

func TestKeysChangeTheScore(t *testing.T) {
	m := stars()
	steps := []struct {
		msg  tui.Msg
		want int
	}{
		{key(tui.KeyRight), 1}, {key(tui.KeyUp), 2}, {digit("4"), 4}, {key(tui.KeyLeft), 3},
		{key(tui.KeyDown), 2}, {key(tui.KeyEnd), 5}, {key(tui.KeyHome), 0}, {digit("9"), 5},
		{key(tui.KeyBackspace), 0},
	}
	for i, s := range steps {
		var cmd tui.Cmd
		m, cmd = m.Update(s.msg)
		c, ok := changed(cmd)
		if !ok || c.Value != s.want || c.ID != "stars" || m.Value() != s.want {
			t.Fatalf("step %d: score %d (message %+v, delivered %v), want %d", i, m.Value(), c, ok, s.want)
		}
	}
}

func TestTheScoreStopsAtTheEnds(t *testing.T) {
	m := stars()
	if _, cmd := m.Update(key(tui.KeyLeft)); cmd != nil {
		t.Fatal("Left below 0 reported a change")
	}
	m.SetValue(99)
	if m.Value() != 5 {
		t.Fatalf("SetValue(99) = %d, want 5", m.Value())
	}
	if _, cmd := m.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("Right above Max reported a change")
	}
	m.SetValue(-3)
	if m.Value() != 0 {
		t.Fatalf("SetValue(-3) = %d, want 0", m.Value())
	}
	m.SetValue(5)
	m.Max = 3 // the score follows a Max that was lowered
	if m.Value() != 3 {
		t.Fatalf("Value() = %d after Max fell to 3", m.Value())
	}
}

func TestOtherKeysDoNothing(t *testing.T) {
	m := stars()
	for _, msg := range []tui.Msg{key(tui.KeyTab), digit("x"), digit("12"), struct{}{}} {
		if _, cmd := m.Update(msg); cmd != nil {
			t.Errorf("%#v changed the score", msg)
		}
	}
}

func TestReadOnlyDisabledOrUnfocusedTakesNoInput(t *testing.T) {
	for name, set := range map[string]func(*Model){
		"read only": func(m *Model) { m.ReadOnly = true },
		"disabled":  func(m *Model) { m.Disabled = true },
		"unfocused": func(m *Model) { m.Blur() },
	} {
		m := stars()
		m.Mouse, m.Bounds = true, hittest.Rect{W: 7, H: 1}
		set(&m)
		if _, cmd := m.Update(key(tui.KeyRight)); cmd != nil {
			t.Errorf("%s: a key changed the score", name)
		}
		_, cmd := m.Update(tui.MouseEvent{X: 3, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
		if (cmd != nil) != (name == "unfocused") {
			t.Errorf("%s: pointer changed the score = %v", name, cmd != nil)
		}
		if name != "unfocused" && len(m.Bindings()) != 0 {
			t.Errorf("%s: Bindings() lists keys that do nothing", name)
		}
	}
	if len(stars().Bindings()) != 4 {
		t.Fatal("an editable rating does not list its four keys")
	}
	m := stars()
	m.Blur()
	if m.Focused() {
		t.Fatal("Focused() after Blur")
	}
}

func TestViewShowsScoreAndFocusWithoutColour(t *testing.T) {
	m := New(5)
	if got := plain(m); got != " ○○○○○ " {
		t.Errorf("empty = %q", got)
	}
	m.SetValue(3)
	if got := plain(m); got != " ●●●○○ " {
		t.Errorf("three = %q", got)
	}
	m.Focus()
	if got := plain(m); got != "[●●●○○]" {
		t.Errorf("focused = %q", got)
	}
	m.ShowValue = true
	if got := plain(m); got != "[●●●○○] 3/5" {
		t.Errorf("with the score = %q", got)
	}
	m.Disabled = true
	if got := plain(m); got != "(●●●○○) 3/5" {
		t.Errorf("disabled = %q", got)
	}
	z := New(0) // read as 5
	if got := plain(z); got != " ○○○○○ " {
		t.Errorf("a Max of 0 = %q, want five marks", got)
	}
	a := New(3)
	a.Theme.Glyphs = theme.ASCIIGlyphSet()
	a.SetValue(1)
	if got := plain(a); got != " *oo " {
		t.Errorf("ASCII = %q", got)
	}
}

func TestMouseSetsAndClearsTheScore(t *testing.T) {
	m := New(5)
	m.Mouse, m.Bounds = true, hittest.Rect{X: 10, Y: 2, W: 7, H: 1} // marks at columns 11..15
	press := func(x int, b tui.MouseButton) (Model, tui.Cmd) {
		return m.Update(tui.MouseEvent{X: x, Y: 2, Button: b, Action: tui.MouseActionPress})
	}
	m, cmd := press(14, tui.MouseButtonLeft)
	if c, ok := changed(cmd); !ok || c.Value != 4 {
		t.Fatalf("press on the fourth mark: %+v", c)
	}
	m, cmd = press(14, tui.MouseButtonLeft)
	if c, ok := changed(cmd); !ok || c.Value != 0 {
		t.Fatalf("a second press on the same mark: %+v, want the score cleared", c)
	}
	for _, x := range []int{10, 16, 40} {
		if _, cmd := press(x, tui.MouseButtonLeft); cmd != nil {
			t.Errorf("a press at column %d, off the marks, changed the score", x)
		}
	}
	if _, cmd := press(12, tui.MouseButtonRight); cmd != nil {
		t.Fatal("the right button changed the score")
	}
	if _, cmd := m.Update(tui.MouseEvent{X: 12, Y: 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease}); cmd != nil {
		t.Fatal("a release changed the score")
	}
	m.Mouse = false
	if _, cmd := press(12, tui.MouseButtonLeft); cmd != nil {
		t.Fatal("the pointer changed a rating with Mouse off")
	}
}

func TestACustomKeyMapReplacesTheDefault(t *testing.T) {
	m := stars()
	m.KeyMap = KeyMap{Inc: keymap.NewBinding("more", "+")}
	if _, cmd := m.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("Right still raised the score with a custom KeyMap")
	}
	if _, cmd := m.Update(digit("+")); cmd == nil {
		t.Fatal("the custom key did not raise the score")
	}
	z := Model{Max: 3, Theme: theme.DarkTheme()}
	z.Focus()
	if next, _ := z.Update(key(tui.KeyRight)); next.Value() != 1 {
		t.Fatal("a struct-literal Model with no KeyMap ignored Right")
	}
}

func TestLinearize(t *testing.T) {
	m := New(5)
	m.SetValue(3)
	if got := m.Linearize(); got != "rating, 3 of 5" {
		t.Errorf("Linearize = %q", got)
	}
	m.ReadOnly = true
	if got := m.Linearize(); got != "rating, 3 of 5, read only" {
		t.Errorf("read only = %q", got)
	}
	m.Disabled = true
	if got := m.Linearize(); got != "rating, 3 of 5, unavailable" {
		t.Errorf("disabled = %q", got)
	}
}

var _ tui.Linearizer = Model{}

func TestThemeTokensAndLayoutNode(t *testing.T) {
	m := New(5).SetTheme(theme.LightTheme())
	if m.Theme.Primary != theme.LightTheme().Primary {
		t.Error("SetTheme did not apply the theme")
	}
	red := ansi.RGB{R: 255}
	m = m.WithTokens(theme.Tokens{Warning: red})
	if m.Tokens().Warning != red {
		t.Error("WithTokens did not override the warning colour")
	}
	n := m.LayoutNode()
	if s := n.Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded}); s != (layout.Size{W: 7, H: 1}) {
		t.Fatalf("Measure = %+v, want 7x1", s)
	}
}
