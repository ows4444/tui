package checkbox

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

func box() Model {
	m := New("Keep metadata")
	m.ID = "keep"
	m.Focus()
	return m
}

func TestSpaceChecksAndUnchecks(t *testing.T) {
	m := box()
	m, cmd := m.Update(key(tui.KeySpace))
	if c, ok := changed(cmd); !ok || c != (ChangedMsg{ID: "keep", Checked: true}) || !m.Checked() {
		t.Fatalf("first Space: %+v, Checked() %v", c, m.Checked())
	}
	m, cmd = m.Update(key(tui.KeySpace))
	if c, ok := changed(cmd); !ok || c.Checked || m.Checked() {
		t.Fatalf("second Space: %+v, Checked() %v", c, m.Checked())
	}
}

func TestEnterAndOtherKeysDoNothing(t *testing.T) {
	m := box()
	for _, msg := range []tui.Msg{key(tui.KeyEnter), key(tui.KeyTab), tui.Key{Type: tui.KeyRunes, Text: "x"}, struct{}{}} {
		if next, cmd := m.Update(msg); cmd != nil || next.Checked() {
			t.Errorf("%#v changed the checkbox", msg)
		}
	}
}

func TestWithoutFocusOrDisabledItIgnoresInput(t *testing.T) {
	m := New("A")
	if _, cmd := m.Update(key(tui.KeySpace)); cmd != nil {
		t.Fatal("Space checked a box that does not have focus")
	}
	m.Focus()
	m.Blur()
	if m.Focused() {
		t.Fatal("Focused() after Blur")
	}
	m.Focus()
	m.Disabled = true
	m.Mouse, m.Bounds = true, hittest.Rect{W: 5, H: 1}
	if _, cmd := m.Update(key(tui.KeySpace)); cmd != nil {
		t.Fatal("Space checked a disabled box")
	}
	if _, cmd := m.Update(tui.MouseEvent{X: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}); cmd != nil {
		t.Fatal("a click checked a disabled box")
	}
	if len(m.Bindings()) != 0 {
		t.Fatal("a disabled box lists a key")
	}
	m.Disabled = false
	if b := m.Bindings(); len(b) != 1 || b[0].Desc != "check" {
		t.Fatalf("Bindings() = %+v", b)
	}
}

func TestSetCheckedDeliversNothingAndClearsIndeterminate(t *testing.T) {
	m := New("A")
	m.Indeterminate = true
	if m.Checked() {
		t.Fatal("an indeterminate box reports checked")
	}
	m.SetChecked(true)
	if !m.Checked() || m.Indeterminate {
		t.Fatal("SetChecked(true) left it unchecked or indeterminate")
	}
	m.SetChecked(false)
	if m.Checked() {
		t.Fatal("SetChecked(false) left it checked")
	}
}

// A press on a box that is partly checked checks it, whatever it was
// underneath.
func TestAPressOnAnIndeterminateBoxChecksIt(t *testing.T) {
	for _, was := range []bool{false, true} {
		m := box()
		m.SetChecked(was)
		m.Indeterminate = true
		m, cmd := m.Update(key(tui.KeySpace))
		if c, ok := changed(cmd); !ok || !c.Checked || !m.Checked() || m.Indeterminate {
			t.Fatalf("was %v: %+v, Checked() %v, Indeterminate %v", was, c, m.Checked(), m.Indeterminate)
		}
	}
}

func TestViewPerStateWithoutColour(t *testing.T) {
	m := New("Keep")
	if got := plain(m); got != "[ ] Keep" {
		t.Errorf("unchecked = %q", got)
	}
	m.SetChecked(true)
	if got := plain(m); got != "[x] Keep" {
		t.Errorf("checked = %q", got)
	}
	m.Focus()
	if got := plain(m); got != "<x> Keep" {
		t.Errorf("focused = %q", got)
	}
	m.Indeterminate = true
	if got := plain(m); got != "<-> Keep" {
		t.Errorf("indeterminate = %q", got)
	}
	if m.Width() != ansi.Width(m.View()) || m.Width() != 8 {
		t.Errorf("Width() = %d, view is %d", m.Width(), ansi.Width(m.View()))
	}
	m.Disabled = true
	if got := plain(m); got != "[-] Keep" {
		t.Errorf("disabled = %q", got)
	}
	if v := m.View(); v == New("Keep").View() || len(v) == len(plain(m)) {
		t.Errorf("a disabled box carries no faint attribute: %q", v)
	}
	bare := New("")
	if got := plain(bare); got != "[ ]" || bare.Width() != 3 {
		t.Errorf("no label = %q, width %d", got, bare.Width())
	}
}

func TestAClickChecksAndUnchecks(t *testing.T) {
	m := New("Keep")
	m.ID = "keep"
	m.Mouse, m.Bounds = true, hittest.Rect{X: 9, Y: 4, W: 8, H: 1}
	press := func(x, y int, b tui.MouseButton, a tui.MouseAction) (Model, tui.Cmd) {
		return m.Update(tui.MouseEvent{X: x, Y: y, Button: b, Action: a})
	}
	m, cmd := press(14, 4, tui.MouseButtonLeft, tui.MouseActionPress) // on the label
	if c, ok := changed(cmd); !ok || !c.Checked {
		t.Fatalf("press on the label: %+v", c)
	}
	m, cmd = press(10, 4, tui.MouseButtonLeft, tui.MouseActionPress) // on the box
	if c, ok := changed(cmd); !ok || c.Checked {
		t.Fatalf("press on the box: %+v", c)
	}
	for name, c := range map[string]tui.Cmd{
		"outside":      second(press(30, 4, tui.MouseButtonLeft, tui.MouseActionPress)),
		"right button": second(press(10, 4, tui.MouseButtonRight, tui.MouseActionPress)),
		"release":      second(press(10, 4, tui.MouseButtonLeft, tui.MouseActionRelease)),
	} {
		if c != nil {
			t.Errorf("%s changed the checkbox", name)
		}
	}
	m.Mouse = false
	if _, cmd := press(10, 4, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Fatal("the pointer changed a checkbox with Mouse off")
	}
}

func second(_ Model, c tui.Cmd) tui.Cmd { return c }

func TestACustomKeyMapReplacesTheDefault(t *testing.T) {
	m := box()
	m.KeyMap = KeyMap{Toggle: keymap.NewBinding("mark", "x")}
	if _, cmd := m.Update(key(tui.KeySpace)); cmd != nil {
		t.Fatal("Space still checked with a custom KeyMap")
	}
	if _, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "x"}); cmd == nil {
		t.Fatal("the custom key did not check")
	}
	z := Model{Label: "A", Theme: theme.DarkTheme()}
	z.Focus()
	if _, cmd := z.Update(key(tui.KeySpace)); cmd == nil {
		t.Fatal("a struct-literal Model with no KeyMap ignored Space")
	}
}

func TestLinearize(t *testing.T) {
	m := New("Keep metadata")
	if got := m.Linearize(); got != "Keep metadata, checkbox, not checked" {
		t.Errorf("unchecked = %q", got)
	}
	m.SetChecked(true)
	if got := m.Linearize(); got != "Keep metadata, checkbox, checked" {
		t.Errorf("checked = %q", got)
	}
	m.Indeterminate, m.Disabled = true, true
	if got := m.Linearize(); got != "Keep metadata, checkbox, partly checked, unavailable" {
		t.Errorf("indeterminate and disabled = %q", got)
	}
	if got := New("").Linearize(); got != "checkbox, not checked" {
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
	if s := n.Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded}); s != (layout.Size{W: 5, H: 1}) {
		t.Fatalf("Measure = %+v, want 5x1", s)
	}
}
