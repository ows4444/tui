package slider

import (
	"fmt"
	"math"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func changed(cmd tui.Cmd) (ChangedMsg, bool) {
	if cmd == nil {
		return ChangedMsg{}, false
	}
	c, ok := cmd().(ChangedMsg)
	return c, ok
}

func plain(m Model) string { return ansi.StripANSI(m.View()) }

func volume() Model {
	m := New(0, 100)
	m.ID = "volume"
	m.Width = 11
	m.Focus()
	return m
}

func TestKeysMoveTheValue(t *testing.T) {
	m := volume()
	steps := []struct {
		k    tui.KeyType
		want float64
	}{
		{tui.KeyRight, 1}, {tui.KeyUp, 2}, {tui.KeyPgUp, 12}, {tui.KeyLeft, 11}, {tui.KeyDown, 10},
		{tui.KeyPgDown, 0}, {tui.KeyEnd, 100}, {tui.KeyHome, 0},
	}
	for i, s := range steps {
		var cmd tui.Cmd
		m, cmd = m.Update(key(s.k))
		c, ok := changed(cmd)
		if !ok || c.Value != s.want || c.ID != "volume" || m.Value() != s.want {
			t.Fatalf("step %d: value %v (message %+v, delivered %v), want %v", i, m.Value(), c, ok, s.want)
		}
	}
}

func TestTheValueStopsAtTheEnds(t *testing.T) {
	m := volume()
	if next, cmd := m.Update(key(tui.KeyLeft)); cmd != nil || next.Value() != 0 {
		t.Fatal("Left moved the value below Min, or reported no change as a change")
	}
	m.SetValue(100)
	if next, cmd := m.Update(key(tui.KeyPgUp)); cmd != nil || next.Value() != 100 {
		t.Fatal("Page Up moved the value above Max")
	}
}

func TestStepRoundsAndMaxStaysReachable(t *testing.T) {
	m := New(0, 10)
	m.Step = 4
	m.Focus()
	m.SetValue(5)
	if m.Value() != 4 {
		t.Fatalf("SetValue(5) with a step of 4 = %v, want 4", m.Value())
	}
	m, _ = m.Update(key(tui.KeyRight))
	m, _ = m.Update(key(tui.KeyRight))
	if m.Value() != 10 {
		t.Fatalf("value = %v, want Max 10: the range is not a whole number of steps", m.Value())
	}
	m.Step, m.BigStep = 0, -1 // read as 1 and as ten steps
	m.SetValue(0)
	m, _ = m.Update(key(tui.KeyRight))
	if m.Value() != 1 {
		t.Fatalf("a Step of 0 moved to %v, want 1", m.Value())
	}
	m, _ = m.Update(key(tui.KeyPgUp))
	if m.Value() != 10 {
		t.Fatalf("a BigStep of -1 moved to %v, want 10", m.Value())
	}
	half := New(0, 1)
	half.Step = 0.25
	half.SetValue(0.6)
	if half.Value() != 0.5 {
		t.Fatalf("SetValue(0.6) with a step of 0.25 = %v, want 0.5", half.Value())
	}
	half.SetValue(math.NaN())
	if half.Value() != 0 {
		t.Fatalf("SetValue(NaN) = %v, want Min", half.Value())
	}
}

func TestAMaxBelowMinIsReadAsMin(t *testing.T) {
	m := New(5, 1)
	m.Focus()
	m, cmd := m.Update(key(tui.KeyRight))
	if cmd != nil || m.Value() != 5 {
		t.Fatalf("value = %v, want 5 with nowhere to move", m.Value())
	}
	if got := plain(m); got == "" {
		t.Fatal("View of an empty range is empty")
	}
}

func TestWithoutFocusOrDisabledItIgnoresKeys(t *testing.T) {
	m := New(0, 10)
	if _, cmd := m.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("a key moved a slider that does not have focus")
	}
	m.Focus()
	m.Disabled = true
	if _, cmd := m.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("a key moved a disabled slider")
	}
	if len(m.Bindings()) != 0 {
		t.Fatal("a disabled slider lists keys")
	}
	m.Disabled = false
	if len(m.Bindings()) != 6 {
		t.Fatalf("Bindings() = %+v", m.Bindings())
	}
	if _, cmd := m.Update(key(tui.KeyTab)); cmd != nil {
		t.Fatal("Tab was handled")
	}
	m.Blur()
	if m.Focused() {
		t.Fatal("Focused() after Blur")
	}
}

func TestViewShowsThumbFocusAndValueWithoutColour(t *testing.T) {
	m := New(0, 100)
	m.Width = 11
	if got := plain(m); got != " ●░░░░░░░░░░ " {
		t.Errorf("at Min = %q", got)
	}
	m.SetValue(50)
	if got := plain(m); got != " █████●░░░░░ " {
		t.Errorf("at 50 = %q", got)
	}
	m.SetValue(100)
	m.Focus()
	if got := plain(m); got != "[██████████●]" {
		t.Errorf("focused at Max = %q", got)
	}
	m.ShowValue = true
	if got := plain(m); got != "[██████████●] 100" {
		t.Errorf("with the value = %q", got)
	}
	m.Format = func(v float64) string { return fmt.Sprintf("%.0f%%", v) }
	if got := plain(m); got != "[██████████●] 100%" {
		t.Errorf("with Format = %q", got)
	}
	m.Disabled = true
	if got := plain(m); got != "(██████████●) 100%" {
		t.Errorf("disabled = %q", got)
	}
	m.Width = 0 // read as 2
	if got := ansi.Width(ansi.StripANSI(New(0, 1).SetTheme(theme.DarkTheme()).View())); got != 22 {
		t.Errorf("default width = %d cells, want 22", got)
	}
	m.Disabled, m.ShowValue = false, false
	if got := plain(m); got != "[█●]" {
		t.Errorf("a Width of 0 = %q, want a track of 2", got)
	}
	a := New(0, 10)
	a.Width = 5
	a.Theme.Glyphs = theme.ASCIIGlyphSet()
	a.SetValue(5)
	if got := plain(a); got != " ##*-- " {
		t.Errorf("ASCII = %q", got)
	}
}

func TestMouseSetsAndDragsTheThumb(t *testing.T) {
	m := New(0, 100)
	m.Width = 11 // track cells at screen columns 6..16
	m.Mouse, m.Bounds = true, hittest.Rect{X: 5, Y: 3, W: 13, H: 1}
	ev := func(x, y int, b tui.MouseButton, a tui.MouseAction) tui.MouseEvent {
		return tui.MouseEvent{X: x, Y: y, Button: b, Action: a}
	}
	m, cmd := m.Update(ev(11, 3, tui.MouseButtonLeft, tui.MouseActionPress))
	if c, ok := changed(cmd); !ok || c.Value != 50 || !m.Dragging() {
		t.Fatalf("press mid-track: %+v, dragging %v", c, m.Dragging())
	}
	m, cmd = m.Update(ev(14, 9, tui.MouseButtonLeft, tui.MouseActionMotion))
	if c, ok := changed(cmd); !ok || c.Value != 80 {
		t.Fatalf("drag to column 14, off the row: %+v", c)
	}
	m, _ = m.Update(ev(90, 3, tui.MouseButtonLeft, tui.MouseActionMotion))
	if m.Value() != 100 {
		t.Fatalf("drag past the right end = %v, want Max", m.Value())
	}
	m, _ = m.Update(ev(0, 3, tui.MouseButtonLeft, tui.MouseActionMotion))
	if m.Value() != 0 {
		t.Fatalf("drag past the left end = %v, want Min", m.Value())
	}
	m, _ = m.Update(ev(0, 3, tui.MouseButtonLeft, tui.MouseActionRelease))
	if m.Dragging() {
		t.Fatal("still dragging after the release")
	}
	if next, cmd := m.Update(ev(11, 3, tui.MouseButtonNone, tui.MouseActionMotion)); cmd != nil || next.Value() != 0 {
		t.Fatal("motion with no drag in progress moved the thumb")
	}
	if _, cmd := m.Update(ev(40, 3, tui.MouseButtonLeft, tui.MouseActionPress)); cmd != nil {
		t.Fatal("a press outside Bounds moved the thumb")
	}
	if next, _ := m.Update(ev(11, 3, tui.MouseButtonRight, tui.MouseActionPress)); next.Dragging() {
		t.Fatal("the right button started a drag")
	}
	m, _ = m.Update(ev(11, 3, tui.MouseButtonLeft, tui.MouseActionPress))
	m.Blur()
	if m.Dragging() {
		t.Fatal("Blur did not let go of the drag")
	}
	m.Disabled = true
	if _, cmd := m.Update(ev(14, 3, tui.MouseButtonLeft, tui.MouseActionPress)); cmd != nil {
		t.Fatal("the pointer moved a disabled slider")
	}
	m.Disabled, m.Mouse = false, false
	if _, cmd := m.Update(ev(14, 3, tui.MouseButtonLeft, tui.MouseActionPress)); cmd != nil {
		t.Fatal("the pointer moved a slider with Mouse off")
	}
}

func TestACustomKeyMapReplacesTheDefault(t *testing.T) {
	m := volume()
	m.KeyMap = KeyMap{Inc: keymap.NewBinding("up", "k")}
	if _, cmd := m.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("Right still moved with a custom KeyMap")
	}
	if _, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "k"}); cmd == nil {
		t.Fatal("the custom key did not move the value")
	}
	z := Model{Max: 10, Theme: theme.DarkTheme()}
	z.Focus()
	if next, _ := z.Update(key(tui.KeyRight)); next.Value() != 1 {
		t.Fatal("a struct-literal Model with no KeyMap ignored Right")
	}
}

func TestLinearize(t *testing.T) {
	m := New(0, 100)
	m.SetValue(40)
	if got := m.Linearize(); got != "slider, 40, from 0 to 100" {
		t.Errorf("Linearize = %q", got)
	}
	m.Disabled = true
	if got := m.Linearize(); got != "slider, 40, from 0 to 100, unavailable" {
		t.Errorf("disabled = %q", got)
	}
}

var _ tui.Linearizer = Model{}

func TestThemeTokensAndLayoutNode(t *testing.T) {
	m := New(0, 10).SetTheme(theme.LightTheme())
	if m.Theme.Primary != theme.LightTheme().Primary {
		t.Error("SetTheme did not apply the theme")
	}
	red := ansi.RGB{R: 255}
	m = m.WithTokens(theme.Tokens{Accent: red})
	if m.Tokens().Accent != red {
		t.Error("WithTokens did not override the accent colour")
	}
	m.Width = 8
	n := m.LayoutNode()
	if s := n.Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded}); s != (layout.Size{W: 10, H: 1}) {
		t.Fatalf("Measure = %+v, want 10x1", s)
	}
}
