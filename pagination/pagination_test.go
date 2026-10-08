package pagination

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

func pager(total, page int) Model {
	m := New(total)
	m.ID = "pages"
	m.SetPage(page)
	m.Focus()
	return m
}

func TestKeysChangeThePage(t *testing.T) {
	m := pager(20, 1)
	steps := []struct {
		k    tui.KeyType
		want int
	}{{tui.KeyRight, 2}, {tui.KeyRight, 3}, {tui.KeyLeft, 2}, {tui.KeyEnd, 20}, {tui.KeyHome, 1}}
	for i, s := range steps {
		var cmd tui.Cmd
		m, cmd = m.Update(key(s.k))
		if c, ok := changed(cmd); !ok || c != (ChangedMsg{ID: "pages", Page: s.want}) || m.Page() != s.want {
			t.Fatalf("step %d: page %d (message %+v), want %d", i, m.Page(), c, s.want)
		}
	}
}

func TestThePageStopsAtTheEnds(t *testing.T) {
	m := pager(3, 1)
	if _, cmd := m.Update(key(tui.KeyLeft)); cmd != nil {
		t.Fatal("Left before the first page reported a change")
	}
	if _, cmd := m.Update(key(tui.KeyHome)); cmd != nil {
		t.Fatal("Home on the first page reported a change")
	}
	m.SetPage(99)
	if m.Page() != 3 {
		t.Fatalf("SetPage(99) = %d, want 3", m.Page())
	}
	if _, cmd := m.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("Right after the last page reported a change")
	}
	m.SetPage(-4)
	if m.Page() != 1 {
		t.Fatalf("SetPage(-4) = %d, want 1", m.Page())
	}
	m.SetPage(3)
	m.Total = 2 // the page follows a Total that was lowered
	if m.Page() != 2 {
		t.Fatalf("Page() = %d after Total fell to 2", m.Page())
	}
}

func TestNoPages(t *testing.T) {
	m := pager(0, 1)
	if m.Page() != 0 || m.View() != "" || m.Width() != 0 || m.Linearize() != "no pages" {
		t.Fatalf("no pages: page %d, view %q, width %d, %q", m.Page(), m.View(), m.Width(), m.Linearize())
	}
	if _, cmd := m.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("a key changed the page of a pager with none")
	}
	m.SetPage(5)
	if m.Page() != 0 {
		t.Fatal("SetPage gave a page to a pager with none")
	}
}

// The first and last pages are always shown, the current one with its
// siblings, and a gap of one page shows the page, not an ellipsis.
func TestWhichPagesAreShown(t *testing.T) {
	for _, tc := range []struct {
		total, page, siblings int
		want                  string
	}{
		{1, 1, 1, "[1]"},
		{5, 3, 1, " 1  2 [3] 4  5 "},
		{20, 1, 1, "[1] 2  …  20 "},
		{20, 2, 1, " 1 [2] 3  …  20 "},
		{20, 4, 1, " 1  2  3 [4] 5  …  20 "},
		{20, 10, 1, " 1  …  9 [10] 11  …  20 "},
		{20, 17, 1, " 1  …  16 [17] 18  19  20 "},
		{20, 20, 1, " 1  …  19 [20]"},
		{20, 10, 0, " 1  … [10] …  20 "},
		{20, 10, 2, " 1  …  8  9 [10] 11  12  …  20 "},
		{20, 10, -3, " 1  … [10] …  20 "},
	} {
		m := New(tc.total)
		m.Siblings = tc.siblings
		m.SetPage(tc.page)
		if got := plain(m); got != tc.want {
			t.Errorf("total %d page %d siblings %d = %q, want %q", tc.total, tc.page, tc.siblings, got, tc.want)
		}
		if m.Width() != ansi.Width(m.View()) {
			t.Errorf("total %d page %d: Width() %d, view is %d", tc.total, tc.page, m.Width(), ansi.Width(m.View()))
		}
	}
}

func TestFocusTurnsTheCurrentPagesBracketsToAngles(t *testing.T) {
	m := New(5)
	m.SetPage(3)
	rest := plain(m)
	m.Focus()
	if got := plain(m); got != " 1  2 <3> 4  5 " || ansi.Width(got) != ansi.Width(rest) {
		t.Errorf("focused = %q", got)
	}
	m.Disabled = true
	if got := plain(m); got != " 1  2 [3] 4  5 " {
		t.Errorf("disabled = %q", got)
	}
	a := New(20)
	a.Theme.Glyphs = theme.ASCIIGlyphSet()
	if got := plain(a); got != "[1] 2  ~  20 " {
		t.Errorf("ASCII = %q", got)
	}
}

func TestWithoutFocusOrDisabledItIgnoresInput(t *testing.T) {
	m := New(5)
	if _, cmd := m.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("a key changed a pager that does not have focus")
	}
	m.Focus()
	m.Blur()
	if m.Focused() {
		t.Fatal("Focused() after Blur")
	}
	m.Focus()
	m.Disabled = true
	m.Mouse, m.Bounds = true, hittest.Rect{W: 15, H: 1}
	if _, cmd := m.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("a key changed a disabled pager")
	}
	if _, cmd := m.Update(tui.MouseEvent{X: 4, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}); cmd != nil {
		t.Fatal("a click changed a disabled pager")
	}
	if len(m.Bindings()) != 0 {
		t.Fatal("a disabled pager lists keys")
	}
	m.Disabled = false
	if len(m.Bindings()) != 4 {
		t.Fatalf("Bindings() = %+v", m.Bindings())
	}
	if _, cmd := m.Update(key(tui.KeyTab)); cmd != nil {
		t.Fatal("Tab was handled")
	}
}

// " 1  …  9 [10] 11  …  20 " drawn at column 4: page 9 is local 6..8, the
// first ellipsis 3..5, page 20 local 20..23.
func TestAClickGoesToThePageUnderIt(t *testing.T) {
	m := New(20)
	m.ID = "pages"
	m.SetPage(10)
	m.Mouse, m.Bounds = true, hittest.Rect{X: 4, Y: 1, W: m.Width(), H: 1}
	press := func(x int, b tui.MouseButton, a tui.MouseAction) (Model, tui.Cmd) {
		return m.Update(tui.MouseEvent{X: x, Y: 1, Button: b, Action: a})
	}
	if next, cmd := press(11, tui.MouseButtonLeft, tui.MouseActionPress); next.Page() != 9 {
		t.Errorf("click on 9: page %d", next.Page())
	} else if c, ok := changed(cmd); !ok || c.Page != 9 {
		t.Errorf("click on 9: %+v", c)
	}
	if next, _ := press(26, tui.MouseButtonLeft, tui.MouseActionPress); next.Page() != 20 {
		t.Errorf("click on 20: page %d", next.Page())
	}
	for name, x := range map[string]int{"the ellipsis": 8, "the current page": 14, "outside": 60} {
		if _, cmd := press(x, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
			t.Errorf("a click on %s changed the page", name)
		}
	}
	if _, cmd := press(11, tui.MouseButtonRight, tui.MouseActionPress); cmd != nil {
		t.Error("the right button changed the page")
	}
	if _, cmd := press(11, tui.MouseButtonLeft, tui.MouseActionRelease); cmd != nil {
		t.Error("a release changed the page")
	}
	m.Bounds.W = 80
	if _, cmd := press(70, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Error("a click past the last page changed the page")
	}
	m.Mouse = false
	if _, cmd := press(11, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Error("the pointer changed a pager with Mouse off")
	}
}

func TestACustomKeyMapReplacesTheDefault(t *testing.T) {
	m := pager(5, 1)
	m.KeyMap = KeyMap{Next: keymap.NewBinding("next", "n")}
	if _, cmd := m.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("Right still changed the page with a custom KeyMap")
	}
	if _, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "n"}); cmd == nil {
		t.Fatal("the custom key did not change the page")
	}
	z := Model{Total: 3, Theme: theme.DarkTheme()}
	z.Focus()
	if next, _ := z.Update(key(tui.KeyRight)); next.Page() != 2 {
		t.Fatal("a struct-literal Model with no KeyMap ignored Right")
	}
}

func TestLinearize(t *testing.T) {
	m := New(20)
	m.SetPage(5)
	if got := m.Linearize(); got != "page 5 of 20" {
		t.Errorf("Linearize = %q", got)
	}
	m.Disabled = true
	if got := m.Linearize(); got != "page 5 of 20, unavailable" {
		t.Errorf("disabled = %q", got)
	}
}

var _ tui.Linearizer = Model{}

func TestThemeTokensAndLayoutNode(t *testing.T) {
	m := New(3).SetTheme(theme.LightTheme())
	if m.Theme.Primary != theme.LightTheme().Primary {
		t.Error("SetTheme did not apply the theme")
	}
	red := ansi.RGB{R: 255}
	m = m.WithTokens(theme.Tokens{Accent: red})
	if m.Tokens().Accent != red {
		t.Error("WithTokens did not override the accent colour")
	}
	n := m.LayoutNode()
	if s := n.Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded}); s != (layout.Size{W: 9, H: 1}) {
		t.Fatalf("Measure = %+v, want 9x1", s)
	}
}
