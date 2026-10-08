package button

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

// pressed runs cmd and reports the PressedMsg it delivered, if any.
func pressed(cmd tui.Cmd) (PressedMsg, bool) {
	if cmd == nil {
		return PressedMsg{}, false
	}
	p, ok := cmd().(PressedMsg)
	return p, ok
}

func plain(m Model) string { return ansi.StripANSI(m.View()) }

func TestEnterAndSpacePressAFocusedButton(t *testing.T) {
	m := New("Save")
	m.ID = "save"
	m.Focus()
	for _, k := range []tui.KeyType{tui.KeyEnter, tui.KeySpace} {
		_, cmd := m.Update(key(k))
		if p, ok := pressed(cmd); !ok || p.ID != "save" {
			t.Errorf("key %v: PressedMsg = %+v, delivered = %v; want ID save", k, p, ok)
		}
	}
}

func TestAButtonWithoutFocusIgnoresKeys(t *testing.T) {
	m := New("Save")
	if _, cmd := m.Update(key(tui.KeyEnter)); cmd != nil {
		t.Fatal("Enter pressed a button that does not have focus")
	}
	m.Focus()
	m.Blur()
	if _, cmd := m.Update(key(tui.KeyEnter)); cmd != nil {
		t.Fatal("Enter pressed a button after Blur")
	}
	if m.Focused() {
		t.Fatal("Focused() after Blur")
	}
}

func TestOtherKeysAndMessagesDoNothing(t *testing.T) {
	m := New("Save")
	m.Focus()
	for _, msg := range []tui.Msg{key(tui.KeyTab), tui.Key{Type: tui.KeyRunes, Text: "x"}, struct{}{}} {
		if next, cmd := m.Update(msg); cmd != nil || next.View() != m.View() || !next.Focused() {
			t.Errorf("%#v changed the button or returned a Cmd", msg)
		}
	}
}

func TestADisabledOrLoadingButtonCannotBePressed(t *testing.T) {
	for name, set := range map[string]func(*Model){
		"disabled": func(m *Model) { m.Disabled = true },
		"loading":  func(m *Model) { m.Loading = true },
	} {
		m := New("Save")
		m.Mouse, m.Bounds = true, hittest.Rect{X: 0, Y: 0, W: 8, H: 1}
		m.Focus()
		set(&m)
		if _, cmd := m.Update(key(tui.KeyEnter)); cmd != nil {
			t.Errorf("%s: Enter pressed it", name)
		}
		m, _ = m.Update(tui.MouseEvent{X: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
		if m.Down() {
			t.Errorf("%s: a pointer press went down on it", name)
		}
		if _, cmd := m.Update(tui.MouseEvent{X: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease}); cmd != nil {
			t.Errorf("%s: a click pressed it", name)
		}
		if len(m.Bindings()) != 0 {
			t.Errorf("%s: Bindings() lists a key that does nothing", name)
		}
	}
}

func TestACustomKeyMapReplacesTheDefault(t *testing.T) {
	m := New("Go")
	m.KeyMap = KeyMap{Press: keymap.NewBinding("go", "g")}
	m.Focus()
	if _, cmd := m.Update(key(tui.KeyEnter)); cmd != nil {
		t.Fatal("Enter still pressed with a custom KeyMap")
	}
	if _, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "g"}); cmd == nil {
		t.Fatal("the custom key did not press")
	}
	if b := m.Bindings(); len(b) != 1 || b[0].Desc != "go" {
		t.Fatalf("Bindings() = %+v", b)
	}
}

func TestAZeroKeyMapBehavesAsTheDefault(t *testing.T) {
	m := Model{Label: "Save", Theme: theme.DarkTheme()}
	m.Focus()
	if _, cmd := m.Update(key(tui.KeyEnter)); cmd == nil {
		t.Fatal("a struct-literal Model with no KeyMap ignored Enter")
	}
}

func TestViewPerStateWithoutColour(t *testing.T) {
	m := New("Save")
	if got := plain(m); got != "[ Save ]" {
		t.Errorf("default = %q", got)
	}
	m.Size = SizeSmall
	if got := plain(m); got != "[Save]" {
		t.Errorf("small = %q", got)
	}
	m.Size = SizeLarge
	if got := plain(m); got != "[   Save   ]" {
		t.Errorf("large = %q", got)
	}
	m.Size = SizeDefault
	m.Disabled = true
	if got := plain(m); got != "( Save )" {
		t.Errorf("disabled = %q", got)
	}
	m.Disabled, m.Loading = false, true
	if got := plain(m); got != "[ Save… ]" {
		t.Errorf("loading = %q", got)
	}
	m.Theme.Glyphs = theme.ASCIIGlyphSet()
	if got := plain(m); strings.ContainsRune(got, '…') {
		t.Errorf("loading with ASCII glyphs = %q", got)
	}
}

func TestGhostAndLinkShowBracketsOnlyWhenFocusedOrHeld(t *testing.T) {
	for _, v := range []Variant{VariantGhost, VariantLink} {
		m := New("More")
		m.Variant = v
		if got := plain(m); got != "  More  " {
			t.Errorf("variant %d at rest = %q", v, got)
		}
		m.Focus()
		if got := plain(m); got != "< More >" {
			t.Errorf("variant %d focused = %q", v, got)
		}
		m.Blur()
		m.Mouse, m.Bounds = true, hittest.Rect{W: 8, H: 1}
		m, _ = m.Update(tui.MouseEvent{X: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
		if got := plain(m); got != "[ More ]" {
			t.Errorf("variant %d held down without focus = %q", v, got)
		}
	}
}

// Every state of a button is the same width, so focusing or pressing one
// does not move what is beside it.
func TestWidthHoldsAcrossStates(t *testing.T) {
	for _, v := range []Variant{VariantDefault, VariantSecondary, VariantDestructive, VariantGhost, VariantLink} {
		m := New("Save")
		m.Variant = v
		m.Mouse, m.Bounds = true, hittest.Rect{W: 8, H: 1}
		want := m.Width()
		check := func(state string, m Model) {
			if got := ansi.Width(m.View()); got != want || m.Width() != want {
				t.Errorf("variant %d %s: width %d (Width() %d), want %d", v, state, got, m.Width(), want)
			}
		}
		check("rest", m)
		f := m
		f.Focus()
		check("focused", f)
		d, _ := m.Update(tui.MouseEvent{X: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
		check("down", d)
		h, _ := m.Update(tui.MouseEvent{X: 1, Action: tui.MouseActionMotion})
		check("hovered", h)
		x := m
		x.Disabled = true
		check("disabled", x)
	}
}

// Each state differs from the resting one in something other than colour:
// its characters, or bold, underline or reverse video.
func TestStatesAreToldApartWithoutColour(t *testing.T) {
	m := New("Save")
	m.Mouse, m.Bounds = true, hittest.Rect{W: 8, H: 1}
	rest := m.View()
	f := m
	f.Focus()
	if got := plain(f); got != "< Save >" || f.View() == rest {
		t.Errorf("a focused button is not in angle brackets: %q", got)
	}
	d2, _ := f.Update(tui.MouseEvent{X: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if got := plain(d2); got != "< Save >" {
		t.Errorf("a focused button held down lost its angle brackets: %q", got)
	}
	d, _ := m.Update(tui.MouseEvent{X: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if v := d.View(); !strings.Contains(v, "7") || v == rest {
		t.Errorf("held-down view carries no reverse video: %q", v)
	}
	x := m
	x.Disabled = true
	if plain(x) == plain(m) {
		t.Error("a disabled button reads the same as an enabled one without colour")
	}
}

func TestLinearize(t *testing.T) {
	m := New("Save")
	if got := m.Linearize(); got != "Save, button" {
		t.Errorf("rest = %q", got)
	}
	m.Focus()
	if got := m.Linearize(); got != "Save, button, focused" {
		t.Errorf("focused = %q", got)
	}
	m.Loading = true
	if got := m.Linearize(); got != "Save, button, busy" {
		t.Errorf("loading = %q", got)
	}
	m.Disabled = true
	if got := m.Linearize(); got != "Save, button, unavailable" {
		t.Errorf("disabled = %q", got)
	}
}

var _ tui.Linearizer = Model{}

func TestSetThemeAndTokens(t *testing.T) {
	m := New("Save").SetTheme(theme.LightTheme())
	if m.Theme.Primary != theme.LightTheme().Primary {
		t.Error("SetTheme did not apply the theme")
	}
	red := ansi.RGB{R: 255}
	m = m.WithTokens(theme.Tokens{Accent: red})
	if m.Tokens().Accent != red {
		t.Errorf("Tokens().Accent = %v, want the override", m.Tokens().Accent)
	}
	if !strings.Contains(m.View(), "255;0;0") {
		t.Errorf("the override colour is not in the view: %q", m.View())
	}
}
