package toggle

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

func changed(cmd tui.Cmd) (ChangedMsg, bool) {
	if cmd == nil {
		return ChangedMsg{}, false
	}
	c, ok := cmd().(ChangedMsg)
	return c, ok
}

func plain(m Model) string { return ansi.StripANSI(m.View()) }

func sw() Model {
	m := New("Overwrite")
	m.ID = "overwrite"
	m.Focus()
	return m
}

func TestSpaceAndEnterFlip(t *testing.T) {
	m := sw()
	m, cmd := m.Update(key(tui.KeySpace))
	if c, ok := changed(cmd); !ok || c != (ChangedMsg{ID: "overwrite", On: true}) || !m.On() {
		t.Fatalf("Space: %+v, On() %v", c, m.On())
	}
	m, cmd = m.Update(key(tui.KeyEnter))
	if c, ok := changed(cmd); !ok || c.On || m.On() {
		t.Fatalf("Enter: %+v, On() %v", c, m.On())
	}
}

// Left and Right say which way to set it, so pressing the one it is already
// at changes nothing and delivers nothing.
func TestLeftTurnsOffAndRightTurnsOn(t *testing.T) {
	m := sw()
	if _, cmd := m.Update(key(tui.KeyLeft)); cmd != nil {
		t.Fatal("Left on a switch that is off delivered a change")
	}
	m, cmd := m.Update(key(tui.KeyRight))
	if c, ok := changed(cmd); !ok || !c.On {
		t.Fatalf("Right: %+v", c)
	}
	if _, cmd := m.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("Right on a switch that is on delivered a change")
	}
	m, cmd = m.Update(key(tui.KeyLeft))
	if c, ok := changed(cmd); !ok || c.On || m.On() {
		t.Fatalf("Left: %+v", c)
	}
}

func TestOtherKeysDoNothing(t *testing.T) {
	m := sw()
	for _, msg := range []tui.Msg{key(tui.KeyTab), tui.Key{Type: tui.KeyRunes, Text: "x"}, struct{}{}} {
		if next, cmd := m.Update(msg); cmd != nil || next.On() {
			t.Errorf("%#v changed the switch", msg)
		}
	}
}

func TestWithoutFocusOrDisabledItIgnoresInput(t *testing.T) {
	m := New("A")
	if _, cmd := m.Update(key(tui.KeySpace)); cmd != nil {
		t.Fatal("Space flipped a switch that does not have focus")
	}
	m.Focus()
	m.Blur()
	if m.Focused() {
		t.Fatal("Focused() after Blur")
	}
	m.Focus()
	m.Disabled = true
	m.Mouse, m.Bounds = true, hittest.Rect{W: 7, H: 1}
	if _, cmd := m.Update(key(tui.KeySpace)); cmd != nil {
		t.Fatal("Space flipped a disabled switch")
	}
	if _, cmd := m.Update(tui.MouseEvent{X: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}); cmd != nil {
		t.Fatal("a click flipped a disabled switch")
	}
	if len(m.Bindings()) != 0 {
		t.Fatal("a disabled switch lists keys")
	}
	m.Disabled = false
	if len(m.Bindings()) != 3 {
		t.Fatalf("Bindings() = %+v", m.Bindings())
	}
}

func TestSetOnDeliversNothing(t *testing.T) {
	m := New("A")
	m.SetOn(true)
	if !m.On() {
		t.Fatal("SetOn(true) left it off")
	}
	m.SetOn(false)
	if m.On() {
		t.Fatal("SetOn(false) left it on")
	}
}

// The knob is at the left and empty when off, at the right and filled when
// on: two differences that do not need colour.
func TestViewPerStateWithoutColour(t *testing.T) {
	m := New("Overwrite")
	if got := plain(m); got != "(○  ) Overwrite" {
		t.Errorf("off = %q", got)
	}
	m.SetOn(true)
	if got := plain(m); got != "(  ●) Overwrite" {
		t.Errorf("on = %q", got)
	}
	m.Focus()
	if got := plain(m); got != "<  ●> Overwrite" {
		t.Errorf("focused = %q", got)
	}
	if m.Width() != ansi.Width(m.View()) || m.Width() != 15 {
		t.Errorf("Width() = %d, view is %d", m.Width(), ansi.Width(m.View()))
	}
	m.Disabled = true
	if got := plain(m); got != "(  ●) Overwrite" {
		t.Errorf("disabled = %q", got)
	}
	on := New("Overwrite")
	on.SetOn(true)
	if m.View() == on.View() {
		t.Error("a disabled switch is drawn exactly as an enabled one")
	}
	bare := New("")
	if got := plain(bare); got != "(○  )" || bare.Width() != 5 {
		t.Errorf("no label = %q, width %d", got, bare.Width())
	}
	a := New("A")
	a.Theme.Glyphs = theme.ASCIIGlyphSet()
	if got := plain(a); got != "(o  ) A" {
		t.Errorf("ASCII off = %q", got)
	}
	a.SetOn(true)
	if got := plain(a); got != "(  *) A" {
		t.Errorf("ASCII on = %q", got)
	}
}

func TestAClickFlips(t *testing.T) {
	m := New("Overwrite")
	m.ID = "overwrite"
	m.Mouse, m.Bounds = true, hittest.Rect{X: 9, Y: 4, W: 15, H: 1}
	press := func(x int, b tui.MouseButton, a tui.MouseAction) (Model, tui.Cmd) {
		return m.Update(tui.MouseEvent{X: x, Y: 4, Button: b, Action: a})
	}
	m, cmd := press(10, tui.MouseButtonLeft, tui.MouseActionPress)
	if c, ok := changed(cmd); !ok || !c.On {
		t.Fatalf("press on the switch: %+v", c)
	}
	m, cmd = press(20, tui.MouseButtonLeft, tui.MouseActionPress)
	if c, ok := changed(cmd); !ok || c.On {
		t.Fatalf("press on the label: %+v", c)
	}
	if _, cmd := press(40, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Error("a press outside Bounds flipped the switch")
	}
	if _, cmd := press(10, tui.MouseButtonRight, tui.MouseActionPress); cmd != nil {
		t.Error("the right button flipped the switch")
	}
	if _, cmd := press(10, tui.MouseButtonLeft, tui.MouseActionRelease); cmd != nil {
		t.Error("a release flipped the switch")
	}
	m.Mouse = false
	if _, cmd := press(10, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Fatal("the pointer flipped a switch with Mouse off")
	}
}

func TestACustomKeyMapReplacesTheDefault(t *testing.T) {
	m := sw()
	m.KeyMap = KeyMap{Flip: keymap.NewBinding("flip", "f")}
	if _, cmd := m.Update(key(tui.KeySpace)); cmd != nil {
		t.Fatal("Space still flipped with a custom KeyMap")
	}
	if _, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "f"}); cmd == nil {
		t.Fatal("the custom key did not flip")
	}
	z := Model{Label: "A", Theme: theme.DarkTheme()}
	z.Focus()
	if _, cmd := z.Update(key(tui.KeyRight)); cmd == nil {
		t.Fatal("a struct-literal Model with no KeyMap ignored Right")
	}
}

func TestLinearize(t *testing.T) {
	m := New("Overwrite")
	if got := m.Linearize(); got != "Overwrite, switch, off" {
		t.Errorf("off = %q", got)
	}
	m.SetOn(true)
	m.Disabled = true
	if got := m.Linearize(); got != "Overwrite, switch, on, unavailable" {
		t.Errorf("on and disabled = %q", got)
	}
	if got := New("").Linearize(); got != "switch, off" {
		t.Errorf("no label = %q", got)
	}
}

var _ tui.Linearizer = Model{}

func TestThemeTokensAndLayoutNode(t *testing.T) {
	m := New("A").SetTheme(theme.LightTheme())
	if m.Theme.Primary != theme.LightTheme().Primary {
		t.Error("SetTheme did not apply the theme")
	}
	red := ansi.RGB{R: 255}
	m = m.WithTokens(theme.Tokens{Text: red})
	if m.Tokens().Text != red {
		t.Error("WithTokens did not override the text colour")
	}
	n := m.LayoutNode()
	if s := n.Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded}); s != (layout.Size{W: 7, H: 1}) {
		t.Fatalf("Measure = %+v, want 7x1", s)
	}
}
