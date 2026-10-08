package breadcrumb

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

func chosen(cmd tui.Cmd) (ChosenMsg, bool) {
	if cmd == nil {
		return ChosenMsg{}, false
	}
	c, ok := cmd().(ChosenMsg)
	return c, ok
}

func plain(m Model) string { return ansi.StripANSI(m.View()) }

func trail() Model {
	m := New("Home", "Docs", "Guide")
	m.ID = "path"
	m.Focus()
	return m
}

func TestViewDrawsTheTrailWithACellEitherSide(t *testing.T) {
	m := New("Home", "Docs", "Guide")
	if got := plain(m); got != " Home / Docs / Guide " {
		t.Errorf("View = %q", got)
	}
	if m.Width() != ansi.Width(m.View()) || m.Width() != 21 {
		t.Errorf("Width() = %d, view is %d", m.Width(), ansi.Width(m.View()))
	}
	m.Separator = ">"
	if got := plain(m); got != " Home > Docs > Guide " {
		t.Errorf("another separator = %q", got)
	}
	if got := plain(New("Only")); got != " Only " {
		t.Errorf("one place = %q", got)
	}
	var empty Model
	if empty.View() != "" || empty.Width() != 0 || empty.Cursor() != -1 {
		t.Errorf("empty: view %q, width %d, cursor %d", empty.View(), empty.Width(), empty.Cursor())
	}
}

// The cursor starts on the current place, the last, and takes angle brackets
// in the two cells every place already has, so the width holds.
func TestFocusMarksTheCursorWithoutChangingWidth(t *testing.T) {
	m := trail()
	if got := plain(m); got != " Home / Docs /<Guide>" {
		t.Errorf("focused = %q", got)
	}
	m, _ = m.Update(key(tui.KeyLeft))
	if got := plain(m); got != " Home /<Docs>/ Guide " {
		t.Errorf("after Left = %q", got)
	}
	m.Blur()
	if got := plain(m); got != " Home / Docs / Guide " || m.Focused() {
		t.Errorf("after Blur = %q", got)
	}
}

func TestArrowsMoveAndStopAtTheEnds(t *testing.T) {
	m := trail()
	steps := []struct {
		k    tui.KeyType
		want int
	}{{tui.KeyRight, 2}, {tui.KeyLeft, 1}, {tui.KeyLeft, 0}, {tui.KeyLeft, 0}, {tui.KeyEnd, 2}, {tui.KeyHome, 0}, {tui.KeyRight, 1}}
	for i, s := range steps {
		var cmd tui.Cmd
		m, cmd = m.Update(key(s.k))
		if m.Cursor() != s.want || cmd != nil {
			t.Fatalf("step %d: cursor %d, want %d", i, m.Cursor(), s.want)
		}
	}
}

func TestEnterAndSpaceChooseThePlaceUnderTheCursor(t *testing.T) {
	m := trail()
	_, cmd := m.Update(key(tui.KeyEnter))
	if c, ok := chosen(cmd); !ok || c != (ChosenMsg{ID: "path", Index: 2, Label: "Guide"}) {
		t.Fatalf("Enter on the current place: %+v", c)
	}
	m, _ = m.Update(key(tui.KeyHome))
	m, cmd = m.Update(key(tui.KeySpace))
	if c, ok := chosen(cmd); !ok || c.Index != 0 || c.Label != "Home" {
		t.Fatalf("Space on the first place: %+v", c)
	}
	if len(m.Items) != 3 {
		t.Fatal("choosing a place changed the trail")
	}
}

func TestWithoutFocusOrEmptyItIgnoresKeys(t *testing.T) {
	m := New("A", "B")
	if next, cmd := m.Update(key(tui.KeyLeft)); cmd != nil || next.Cursor() != 1 {
		t.Fatal("a key moved the cursor of a trail that does not have focus")
	}
	if _, cmd := m.Update(key(tui.KeyEnter)); cmd != nil {
		t.Fatal("Enter chose a place of a trail that does not have focus")
	}
	var empty Model
	empty.Focus()
	if _, cmd := empty.Update(key(tui.KeyEnter)); cmd != nil {
		t.Fatal("an empty trail returned a Cmd")
	}
	m.Focus()
	if _, cmd := m.Update(key(tui.KeyTab)); cmd != nil {
		t.Fatal("Tab was handled")
	}
	if len(m.Bindings()) != 5 {
		t.Fatalf("Bindings() = %+v", m.Bindings())
	}
}

// A long trail keeps its first place and its last few; the ones between
// fold into an ellipsis that the cursor steps over.
func TestMaxFoldsTheMiddle(t *testing.T) {
	m := New("Home", "Docs", "API", "Widgets", "Button")
	m.Max = 3
	if got := plain(m); got != " Home / … / Widgets / Button " {
		t.Errorf("folded = %q", got)
	}
	if m.Width() != ansi.Width(m.View()) {
		t.Errorf("Width() = %d, view is %d", m.Width(), ansi.Width(m.View()))
	}
	m.Focus()
	m, _ = m.Update(key(tui.KeyLeft)) // Widgets
	m, _ = m.Update(key(tui.KeyLeft)) // over the ellipsis, to Home
	if m.Cursor() != 0 {
		t.Fatalf("cursor = %d, want 0: the folded places are stepped over", m.Cursor())
	}
	m.SetCursor(1) // folded: ignored
	if m.Cursor() != 0 {
		t.Fatal("SetCursor put the cursor on a folded place")
	}
	m.SetCursor(3)
	if m.Cursor() != 3 {
		t.Fatal("SetCursor did not move to a place that is drawn")
	}
	m.SetCursor(-1)
	m.SetCursor(9)
	if m.Cursor() != 3 {
		t.Fatal("SetCursor took an index out of range")
	}
	m.Max = 9 // more than there are: nothing folds
	if got := plain(m); got != " Home / Docs / API /<Widgets>/ Button " {
		t.Errorf("Max above the length = %q", got)
	}
	m.Max = 1 // too small to fold around: read as no limit
	if got := ansi.Width(plain(m)); got != m.Width() || got < 30 {
		t.Errorf("a Max of 1 folded the trail: %q", plain(m))
	}
	m.Theme.Glyphs = theme.ASCIIGlyphSet()
	m.Max = 2
	m.Blur()
	if got := plain(m); got != " Home / ~ / Button " {
		t.Errorf("ASCII = %q", got)
	}
}

// The cursor follows a trail that got shorter.
func TestTheCursorStaysInsideAShorterTrail(t *testing.T) {
	m := trail()
	m, _ = m.Update(key(tui.KeyEnd))
	m.Items = m.Items[:1]
	if m.Cursor() != 0 {
		t.Fatalf("cursor = %d on a trail of one", m.Cursor())
	}
}

// " Home / Docs / Guide " drawn at column 10: Home owns local 0..5, the
// first separator is 6, Docs owns 7..12, the second is 13, Guide 14..20.
func TestAClickChoosesThePlaceUnderIt(t *testing.T) {
	m := New("Home", "Docs", "Guide")
	m.ID = "path"
	m.Mouse, m.Bounds = true, hittest.Rect{X: 10, Y: 0, W: 21, H: 1}
	press := func(x int, b tui.MouseButton, a tui.MouseAction) (Model, tui.Cmd) {
		return m.Update(tui.MouseEvent{X: x, Y: 0, Button: b, Action: a})
	}
	next, cmd := press(19, tui.MouseButtonLeft, tui.MouseActionPress)
	if c, ok := chosen(cmd); !ok || c.Index != 1 || c.Label != "Docs" || next.Cursor() != 1 {
		t.Fatalf("click on Docs: %+v, cursor %d", c, next.Cursor())
	}
	if _, cmd := press(11, tui.MouseButtonLeft, tui.MouseActionPress); cmd == nil {
		t.Error("a click on Home chose nothing")
	}
	for name, x := range map[string]int{"a separator": 16, "outside": 60} {
		if _, cmd := press(x, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
			t.Errorf("a click on %s chose a place", name)
		}
	}
	if _, cmd := press(19, tui.MouseButtonRight, tui.MouseActionPress); cmd != nil {
		t.Error("the right button chose a place")
	}
	if _, cmd := press(19, tui.MouseButtonLeft, tui.MouseActionRelease); cmd != nil {
		t.Error("a release chose a place")
	}
	m.Bounds.W = 60
	if _, cmd := press(50, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Error("a click past the trail chose a place")
	}
	f := New("Home", "Docs", "API", "Guide")
	f.Max, f.Mouse, f.Bounds = 2, true, hittest.Rect{W: 40, H: 1}
	if _, cmd := f.Update(tui.MouseEvent{X: 8, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}); cmd != nil {
		t.Error("a click on the ellipsis chose a place")
	}
	m.Mouse = false
	if _, cmd := press(19, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Error("the pointer chose a place with Mouse off")
	}
}

func TestACustomKeyMapReplacesTheDefault(t *testing.T) {
	m := trail()
	m.KeyMap = KeyMap{Prev: keymap.NewBinding("back", "b")}
	if next, _ := m.Update(key(tui.KeyLeft)); next.Cursor() != 2 {
		t.Fatal("Left still moved with a custom KeyMap")
	}
	if next, _ := m.Update(tui.Key{Type: tui.KeyRunes, Text: "b"}); next.Cursor() != 1 {
		t.Fatal("the custom key did not move the cursor")
	}
	z := Model{Items: []string{"A", "B"}, Theme: theme.DarkTheme()}
	z.Focus()
	if next, _ := z.Update(key(tui.KeyLeft)); next.Cursor() != 0 {
		t.Fatal("a struct-literal Model with no KeyMap ignored Left")
	}
}

func TestLinearizeListsEveryPlace(t *testing.T) {
	m := New("Home", "Docs", "API", "Guide")
	m.Max = 2
	want := "Home, place 1 of 4\nDocs, place 2 of 4\nAPI, place 3 of 4\nGuide, place 4 of 4, current"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	if got := (Model{}).Linearize(); got != "No places" {
		t.Errorf("empty = %q", got)
	}
}

var _ tui.Linearizer = Model{}

func TestThemeTokensAndLayoutNode(t *testing.T) {
	m := New("A", "B").SetTheme(theme.LightTheme())
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
