package carousel

import (
	"slices"
	"strconv"
	"strings"
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

// rows is the view without styling, one string per row.
func rows(m Model) []string { return strings.Split(ansi.StripANSI(m.View()), "\n") }

func three() Model {
	m := New("one\ntwo lines", "second", "3")
	m.ID = "tips"
	m.Focus()
	return m
}

// The slide area fits the widest and the tallest slide, so every slide
// draws at one size.
func TestViewDrawsOneSlideAndADotForEach(t *testing.T) {
	m := New("one\ntwo lines", "second", "3")
	if got := rows(m); !slices.Equal(got, []string{"one      ", "two lines", "  ● ○ ○  "}) {
		t.Errorf("View = %q", got)
	}
	if w, h := m.Size(); w != 9 || h != 3 {
		t.Errorf("Size() = %d, %d", w, h)
	}
	m.SetIndex(1)
	if got := rows(m); !slices.Equal(got, []string{"second   ", "         ", "  ○ ● ○  "}) {
		t.Errorf("second slide = %q", got)
	}
	m.SetIndex(99)
	if m.Index() != 2 {
		t.Errorf("SetIndex past the end = %d", m.Index())
	}
	m.SetIndex(-4)
	if m.Index() != 0 {
		t.Errorf("SetIndex before the start = %d", m.Index())
	}
	var empty Model
	empty.SetIndex(3)
	if w, h := empty.Size(); empty.View() != "" || w != 0 || h != 0 || empty.Index() != -1 {
		t.Errorf("empty: view %q, size %d,%d, index %d", empty.View(), w, h, empty.Index())
	}
}

func TestFocusPutsTheDotsInAngleBracketsWithoutChangingSize(t *testing.T) {
	m := three()
	if got := rows(m)[2]; got != " <● ○ ○> " {
		t.Errorf("focused = %q", got)
	}
	m.Blur()
	if got := rows(m)[2]; got != "  ● ○ ○  " || m.Focused() {
		t.Errorf("after Blur = %q", got)
	}
}

func TestArrowsStopAtTheEndsUnlessItLoops(t *testing.T) {
	m := three()
	steps := []struct {
		k     tui.KeyType
		want  int
		moved bool
	}{{tui.KeyLeft, 0, false}, {tui.KeyRight, 1, true}, {tui.KeyRight, 2, true}, {tui.KeyRight, 2, false},
		{tui.KeyHome, 0, true}, {tui.KeyHome, 0, false}, {tui.KeyEnd, 2, true}, {tui.KeyLeft, 1, true}}
	for i, s := range steps {
		var cmd tui.Cmd
		m, cmd = m.Update(key(s.k))
		c, ok := changed(cmd)
		if m.Index() != s.want || ok != s.moved || (ok && c != (ChangedMsg{ID: "tips", Index: s.want})) {
			t.Fatalf("step %d: index %d, message %+v (%v)", i, m.Index(), c, ok)
		}
	}
	m.Loop = true
	m.SetIndex(2)
	m, cmd := m.Update(key(tui.KeyRight))
	if c, ok := changed(cmd); !ok || c.Index != 0 || m.Index() != 0 {
		t.Fatalf("Right on the last slide of a loop: %+v", c)
	}
	m, _ = m.Update(key(tui.KeyLeft))
	if m.Index() != 2 {
		t.Fatalf("Left on the first slide of a loop: %d", m.Index())
	}
	one := New("only")
	one.Loop = true
	one.Focus()
	if _, cmd := one.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("a carousel of one slide reported a change")
	}
}

func TestWithoutFocusOrEmptyItIgnoresKeys(t *testing.T) {
	m := New("a", "b")
	if next, cmd := m.Update(key(tui.KeyRight)); cmd != nil || next.Index() != 0 {
		t.Fatal("a key changed the slide of a carousel that does not have focus")
	}
	m.Focus()
	if _, cmd := m.Update(key(tui.KeyTab)); cmd != nil {
		t.Fatal("Tab was handled")
	}
	var empty Model
	empty.Focus()
	if _, cmd := empty.Update(key(tui.KeyRight)); cmd != nil {
		t.Fatal("an empty carousel returned a Cmd")
	}
	if len(m.Bindings()) != 4 {
		t.Fatalf("Bindings() = %+v", m.Bindings())
	}
}

// A slide larger than a set size is cut; styling is kept, and any other
// escape sequence is dropped.
func TestASetSizeCutsAndPadsTheSlide(t *testing.T) {
	bold := ansi.NewStyle().Bold().Render("abcdefghijkl")
	m := New(bold+"\nsecond row\nthird row", "x\ty")
	m.Width, m.Height = 8, 2
	if got := rows(m); !slices.Equal(got, []string{"abcdefgh", "second r", "  ● ○   "}) {
		t.Errorf("cut = %q", got)
	}
	if !strings.Contains(m.View(), "\x1b[1m") {
		t.Error("the slide lost its styling")
	}
	for i, l := range strings.Split(m.View(), "\n") {
		if ansi.Width(l) != 8 {
			t.Errorf("row %d is %d cells", i, ansi.Width(l))
		}
	}
	m.SetIndex(1)
	if got := rows(m)[0]; !strings.HasPrefix(got, "x ") || strings.Contains(got, "\t") {
		t.Errorf("a tab was not expanded: %q", got)
	}
	m.Slides[1] = "a\x1b[2Jb"
	if got := m.View(); strings.Contains(got, "\x1b[2J") || !strings.HasPrefix(rows(m)[0], "ab") {
		t.Errorf("an escape sequence got through: %q", got)
	}
	m.Width, m.Height = 3, 0 // narrower than the dots: the width is raised
	if w, h := m.Size(); w != 5 || h != 4 {
		t.Errorf("Size() = %d, %d, want 5, 4", w, h)
	}
	m.Width, m.Height = 0, 1
	if w, h := m.Size(); w != 12 || h != 2 {
		t.Errorf("Size() = %d, %d, want 12, 2", w, h)
	}
}

// Twelve slides need 25 cells for their dots; in fewer the row is a count.
func TestManySlidesShowACount(t *testing.T) {
	slides := make([]string, 12)
	for i := range slides {
		slides[i] = "slide " + strconv.Itoa(i+1)
	}
	m := New(slides...)
	m.SetIndex(2)
	if got := rows(m)[1]; got != " 3 / 12 " {
		t.Errorf("count = %q", got)
	}
	m.Focus()
	if got := rows(m)[1]; got != "<3 / 12>" {
		t.Errorf("focused count = %q", got)
	}
	m.Mouse, m.Bounds = true, hittest.Rect{W: 8, H: 2}
	if _, cmd := m.Update(tui.MouseEvent{X: 1, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}); cmd != nil {
		t.Error("a click on the count changed the slide")
	}
	m.Width = 25
	m.Blur()
	if got := strings.TrimSpace(rows(m)[1]); got != "○ ○ ● ○ ○ ○ ○ ○ ○ ○ ○ ○" {
		t.Errorf("dots = %q", got)
	}
	m.Theme.Glyphs = theme.ASCIIGlyphSet()
	if got := strings.TrimSpace(rows(m)[1]); !strings.HasPrefix(got, "o o * o") {
		t.Errorf("ASCII = %q", got)
	}
}

// Nine cells wide, drawn at column 10, row 5: the slide is rows 5 and 6,
// and the dots of "  ● ○ ○  " stand at local columns 2, 4 and 6 of row 7.
func TestAClickOnADotOrAHalfChangesTheSlide(t *testing.T) {
	m := New("one\ntwo lines", "second", "3")
	m.Mouse, m.Bounds = true, hittest.Rect{X: 10, Y: 5, W: 9, H: 3}
	press := func(x, y int, b tui.MouseButton, a tui.MouseAction) (Model, tui.Cmd) {
		return m.Update(tui.MouseEvent{X: x, Y: y, Button: b, Action: a})
	}
	next, cmd := press(16, 7, tui.MouseButtonLeft, tui.MouseActionPress)
	if c, ok := changed(cmd); !ok || c.Index != 2 || next.Index() != 2 {
		t.Fatalf("click on the third dot: %+v", c)
	}
	for name, x := range map[string]int{"the dot of the slide showing": 12, "between two dots": 13, "before the dots": 10, "after the dots": 18} {
		if _, cmd := press(x, 7, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
			t.Errorf("a click on %s changed the slide", name)
		}
	}
	if next, _ := press(17, 5, tui.MouseButtonLeft, tui.MouseActionPress); next.Index() != 1 {
		t.Error("a click on the right half did not go on")
	}
	if _, cmd := press(11, 6, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Error("a click on the left half of the first slide changed it")
	}
	m.SetIndex(2)
	if next, _ := press(11, 6, tui.MouseButtonLeft, tui.MouseActionPress); next.Index() != 1 {
		t.Error("a click on the left half did not go back")
	}
	if _, cmd := press(11, 6, tui.MouseButtonRight, tui.MouseActionPress); cmd != nil {
		t.Error("the right button changed the slide")
	}
	if _, cmd := press(11, 6, tui.MouseButtonLeft, tui.MouseActionRelease); cmd != nil {
		t.Error("a release changed the slide")
	}
	if _, cmd := press(40, 6, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Error("a click outside changed the slide")
	}
	m.Bounds = hittest.Rect{X: 10, Y: 5, W: 30, H: 9}
	if _, cmd := press(25, 6, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Error("a click to the right of the carousel changed the slide")
	}
	if _, cmd := press(11, 9, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Error("a click under the carousel changed the slide")
	}
	m.Mouse = false
	if _, cmd := press(11, 6, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Error("the pointer changed the slide with Mouse off")
	}
}

func TestACustomKeyMapReplacesTheDefault(t *testing.T) {
	m := three()
	m.KeyMap = KeyMap{Next: keymap.NewBinding("on", "n")}
	if next, _ := m.Update(key(tui.KeyRight)); next.Index() != 0 {
		t.Fatal("Right still worked with a custom KeyMap")
	}
	if next, _ := m.Update(tui.Key{Type: tui.KeyRunes, Text: "n"}); next.Index() != 1 {
		t.Fatal("the custom key did not change the slide")
	}
	z := Model{Slides: []string{"A", "B"}, Theme: theme.DarkTheme()}
	z.Focus()
	if next, _ := z.Update(key(tui.KeyRight)); next.Index() != 1 {
		t.Fatal("a struct-literal Model with no KeyMap ignored Right")
	}
}

func TestLinearizeNamesTheSlideAndReadsItPlain(t *testing.T) {
	m := New("a", ansi.NewStyle().Bold().Render("bold")+"\nmore", "c")
	m.SetIndex(1)
	if got := m.Linearize(); got != "Slide 2 of 3\nbold\nmore" {
		t.Errorf("Linearize = %q", got)
	}
	if got := (Model{}).Linearize(); got != "No slides" {
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
	m = m.WithTokens(theme.Tokens{Accent: red})
	if m.Tokens().Accent != red {
		t.Error("WithTokens did not override the accent colour")
	}
	n := m.LayoutNode()
	if s := n.Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded}); s != (layout.Size{W: 5, H: 2}) {
		t.Fatalf("Measure = %+v, want 5x2", s)
	}
}
