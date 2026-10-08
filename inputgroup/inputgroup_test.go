package inputgroup

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func plain(m Model) string { return ansi.StripANSI(m.View()) }

func typed(s string) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: s, Code: []rune(s)[0]} }

func TestTheFieldTypesAsATextInputDoes(t *testing.T) {
	m := New("https://", ".com")
	m.Focus()
	for _, r := range "example" {
		m, _ = m.Update(typed(string(r)))
	}
	if m.Value() != "example" {
		t.Fatalf("Value = %q", m.Value())
	}
	if m.FullValue() != "https://example.com" {
		t.Fatalf("FullValue = %q", m.FullValue())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	if m.Value() != "exampl" {
		t.Fatalf("after Backspace: %q", m.Value())
	}
}

func TestViewDrawsTheAdditionsEitherSide(t *testing.T) {
	m := New("$ ", " USD")
	if got := plain(m); got != "$  USD" {
		t.Errorf("empty = %q", got)
	}
	m.SetValue("40")
	if got := plain(m); got != "$ 40 USD" {
		t.Errorf("with a value = %q", got)
	}
	if got := plain(New("", "")); got != "" {
		t.Errorf("no additions and no value = %q", got)
	}
	only := New("", "kg")
	only.SetValue("12")
	if got := plain(only); got != "12kg" {
		t.Errorf("suffix only = %q", got)
	}
}

// With a Width the field is padded to it, so the suffix does not move as
// the text grows or the cursor comes and goes.
func TestASetWidthKeepsTheSuffixInOneColumn(t *testing.T) {
	m := New("https://", ".com")
	m.Width = 12
	col := func(m Model) int { return strings.Index(plain(m), ".com") }
	want := col(m)
	if want != len("https://")+13 {
		t.Fatalf("suffix column = %d, want %d", want, len("https://")+13)
	}
	m.Focus()
	if col(m) != want {
		t.Errorf("the suffix moved on focus: %q", plain(m))
	}
	m.SetValue("example")
	if col(m) != want {
		t.Errorf("the suffix moved with a value: %q", plain(m))
	}
	m.SetValue("a-value-longer-than-the-field")
	if col(m) != want {
		t.Errorf("the suffix moved with a long value: %q", plain(m))
	}
}

func TestCursorCellIsPastThePrefix(t *testing.T) {
	m := New("https://", ".com")
	if _, _, ok := m.CursorCell(); ok {
		t.Fatal("CursorCell reports a cursor without focus")
	}
	m.Focus()
	m.SetValue("abc")
	x, y, ok := m.CursorCell()
	if !ok || y != 0 || x != len("https://")+3 {
		t.Fatalf("CursorCell = %d, %d, %v; want %d, 0, true", x, y, ok, len("https://")+3)
	}
}

func TestAdditionsAreSanitised(t *testing.T) {
	m := New("\x1b[2Jpre", "\x1b]52;c;x\x07post")
	out := m.View()
	if strings.Contains(out, "\x1b[2J") || strings.Contains(out, "\x1b]52") {
		t.Fatalf("an escape sequence in an addition reached the view: %q", out)
	}
	if got := plain(m); got != "prepost" {
		t.Errorf("View = %q", got)
	}
}

func TestLayoutNodeGivesTheFieldWhatIsLeft(t *testing.T) {
	m := New("https://", ".com")
	m.SetValue("example")
	n := m.LayoutNode()
	s := n.Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded})
	if s != (layout.Size{W: len("https://") + len("example") + 1 + len(".com"), H: 1}) {
		t.Fatalf("Measure = %+v", s)
	}
	got := ansi.StripANSI(n.Render(layout.Size{W: 30, H: 1}))
	if ansi.Width(got) != 30 || !strings.HasSuffix(got, ".com") || !strings.HasPrefix(got, "https://example") {
		t.Fatalf("Render(30) = %q, want the suffix at the right edge", got)
	}
	if n.Render(layout.Size{}) != "" {
		t.Error("Render of no room is not empty")
	}
	// Too narrow for the additions: the field still gets one cell.
	if got := n.Render(layout.Size{W: 5, H: 1}); ansi.Width(got) != 5 {
		t.Errorf("Render(5) is %d cells wide", ansi.Width(got))
	}
}

func TestLinearize(t *testing.T) {
	m := New("https://", ".com")
	m.SetValue("example")
	got := m.Linearize()
	if !strings.Contains(got, "example") || !strings.HasSuffix(got, `, before it "https://", after it ".com"`) {
		t.Errorf("Linearize = %q", got)
	}
	if bare := New("", "").Linearize(); strings.Contains(bare, "before it") || strings.Contains(bare, "after it") {
		t.Errorf("no additions = %q", bare)
	}
}

var _ tui.Linearizer = Model{}

func TestThemeAndTokensReachTheAdditions(t *testing.T) {
	m := New("$ ", "").SetTheme(theme.LightTheme())
	if m.Theme.Muted != theme.LightTheme().Muted {
		t.Error("SetTheme did not reach the additions")
	}
	red := ansi.RGB{R: 255}
	m = m.WithTokens(theme.Tokens{Muted: red})
	if m.Tokens().Muted != red {
		t.Error("WithTokens did not set the override")
	}
	if !strings.Contains(m.View(), "255;0;0") {
		t.Errorf("the prefix is not drawn in the overridden muted colour: %q", m.View())
	}
}
