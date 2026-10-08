package transferlist

import (
	"slices"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func ctrlA() tui.Key {
	return tui.Key{Type: tui.KeyRunes, Text: "a", Code: 'a', Mod: input.ModCtrl}
}

func changed(cmd tui.Cmd) (ChangedMsg, bool) {
	if cmd == nil {
		return ChangedMsg{}, false
	}
	c, ok := cmd().(ChangedMsg)
	return c, ok
}

// rows is the view without styling, one string per row, with the blank
// cells at the end of each row dropped.
func rows(m Model) []string {
	out := strings.Split(ansi.StripANSI(m.View()), "\n")
	for i := range out {
		out[i] = strings.TrimRight(out[i], " ")
	}
	return out
}

func fruit() Model {
	m := New("Apples", "Bananas", "Cherries")
	m.ID = "fruit"
	m.Focus()
	return m
}

func TestViewDrawsTwoListsUnderTheirTitles(t *testing.T) {
	m := New("Apples", "Bananas", "Cherries")
	m.Right = []string{"Dates"}
	want := []string{
		" Available (3)  │  Chosen (1)",
		" Apples         │  Dates",
		" Bananas        │",
		" Cherries       │",
	}
	if got := rows(m); !slices.Equal(got, want) {
		t.Errorf("View =\n%s", strings.Join(got, "\n"))
	}
	for i, l := range strings.Split(m.View(), "\n") {
		if ansi.Width(l) != m.Width() {
			t.Errorf("row %d is %d cells, Width() is %d", i, ansi.Width(l), m.Width())
		}
	}
	if m.Rows() != 4 || m.Width() != 15+3+12 {
		t.Errorf("Rows() = %d, Width() = %d", m.Rows(), m.Width())
	}
	m.LeftTitle, m.RightTitle = "Off", "On"
	if got := rows(m)[0]; got != " Off (3)   │  On (1)" {
		t.Errorf("titles = %q", got)
	}
	var empty Model
	if got := rows(empty); !slices.Equal(got, []string{" Available (0)  │  Chosen (0)", "                │"}) {
		t.Errorf("empty = %q", got)
	}
	if empty.Cursor(Left) != -1 || empty.Active() != Left {
		t.Errorf("empty: cursor %d, active %d", empty.Cursor(Left), empty.Active())
	}
}

func TestFocusMarksTheCursorWithoutChangingWidth(t *testing.T) {
	m := fruit()
	w := m.Width()
	if got := rows(m)[1]; got != "<Apples>        │" {
		t.Errorf("focused = %q", got)
	}
	m, _ = m.Update(key(tui.KeyDown))
	if got := rows(m); got[1] != " Apples         │" || got[2] != "<Bananas>       │" {
		t.Errorf("after Down = %q", got)
	}
	if m.Width() != w {
		t.Errorf("focus changed the width from %d to %d", w, m.Width())
	}
	m.Blur()
	if strings.ContainsAny(ansi.StripANSI(m.View()), "<>") || m.Focused() {
		t.Error("the cursor is still marked after Blur")
	}
}

func TestArrowsMoveTheCursorAndStopAtTheEnds(t *testing.T) {
	m := fruit()
	steps := []struct {
		k    tui.KeyType
		want int
	}{{tui.KeyUp, 0}, {tui.KeyDown, 1}, {tui.KeyDown, 2}, {tui.KeyDown, 2}, {tui.KeyHome, 0}, {tui.KeyEnd, 2}, {tui.KeyUp, 1}}
	for i, s := range steps {
		var cmd tui.Cmd
		m, cmd = m.Update(key(s.k))
		if m.Cursor(Left) != s.want || cmd != nil {
			t.Fatalf("step %d: cursor %d, want %d", i, m.Cursor(Left), s.want)
		}
	}
}

func TestEnterMovesTheItemUnderTheCursorAcross(t *testing.T) {
	m := fruit()
	given := m.Left
	m, _ = m.Update(key(tui.KeyDown))
	m, cmd := m.Update(key(tui.KeyEnter))
	c, ok := changed(cmd)
	if !ok || c.ID != "fruit" || !slices.Equal(c.Moved, []string{"Bananas"}) || c.To != Right {
		t.Fatalf("ChangedMsg = %+v", c)
	}
	if !slices.Equal(m.Left, []string{"Apples", "Cherries"}) || !slices.Equal(m.Right, []string{"Bananas"}) {
		t.Fatalf("lists = %q, %q", m.Left, m.Right)
	}
	if !slices.Equal(given, []string{"Apples", "Bananas", "Cherries"}) {
		t.Fatalf("the caller's slice was written into: %q", given)
	}
	// The cursor stays where it was, now on the item that came up.
	if m.Active() != Left || m.Cursor(Left) != 1 {
		t.Fatalf("after the move: active %d, cursor %d", m.Active(), m.Cursor(Left))
	}
	m, _ = m.Update(key(tui.KeySpace)) // Cherries, the last: the cursor steps back
	if m.Cursor(Left) != 0 || !slices.Equal(m.Right, []string{"Bananas", "Cherries"}) {
		t.Fatalf("after the last item moved: cursor %d, right %q", m.Cursor(Left), m.Right)
	}
	m, _ = m.Update(key(tui.KeyEnter)) // Apples: the left list is now empty
	if m.Active() != Right || len(m.Left) != 0 {
		t.Fatalf("an emptied list kept the cursor: active %d, left %q", m.Active(), m.Left)
	}
	if got := rows(m)[1]; got != "                │ <Bananas>" {
		t.Errorf("cursor in the right list = %q", got)
	}
	m, cmd = m.Update(key(tui.KeyEnter))
	if c, _ := changed(cmd); c.To != Left || !slices.Equal(m.Left, []string{"Bananas"}) {
		t.Fatalf("moving back: %+v, left %q", c, m.Left)
	}
}

func TestLeftAndRightChangeListUnlessItIsEmpty(t *testing.T) {
	m := fruit()
	m, _ = m.Update(key(tui.KeyRight))
	if m.Active() != Left {
		t.Fatal("Right went to an empty list")
	}
	m.Right = []string{"Dates", "Elderberries"}
	m, _ = m.Update(key(tui.KeyRight))
	m, _ = m.Update(key(tui.KeyDown))
	if m.Active() != Right || m.Cursor(Right) != 1 || m.Cursor(Left) != 0 {
		t.Fatalf("active %d, cursors %d and %d", m.Active(), m.Cursor(Left), m.Cursor(Right))
	}
	m, _ = m.Update(key(tui.KeyLeft))
	if m.Active() != Left {
		t.Fatal("Left did not go back")
	}
	m.Left = nil
	m.side = Left
	if m.Active() != Right {
		t.Fatal("the cursor stayed in a list the caller emptied")
	}
	m, _ = m.Update(key(tui.KeyLeft))
	if m.Active() != Right {
		t.Fatal("Left went to an empty list")
	}
}

func TestCtrlAMovesTheWholeList(t *testing.T) {
	m := fruit()
	m.Right = []string{"Dates"}
	m, cmd := m.Update(ctrlA())
	c, ok := changed(cmd)
	if !ok || len(c.Moved) != 3 || c.To != Right {
		t.Fatalf("ChangedMsg = %+v", c)
	}
	if len(m.Left) != 0 || !slices.Equal(m.Right, []string{"Dates", "Apples", "Bananas", "Cherries"}) || m.Active() != Right {
		t.Fatalf("lists = %q, %q, active %d", m.Left, m.Right, m.Active())
	}
}

func TestItIgnoresKeysWithoutFocusDisabledOrEmpty(t *testing.T) {
	m := New("A", "B")
	if next, cmd := m.Update(key(tui.KeyEnter)); cmd != nil || len(next.Left) != 2 {
		t.Fatal("Enter moved an item of lists that do not have focus")
	}
	m.Focus()
	m.Disabled = true
	if _, cmd := m.Update(key(tui.KeyEnter)); cmd != nil {
		t.Fatal("Enter moved an item of disabled lists")
	}
	if strings.ContainsAny(ansi.StripANSI(m.View()), "<>") {
		t.Error("disabled lists mark a cursor")
	}
	m.Mouse, m.Bounds = true, hittest.Rect{W: 40, H: 3}
	if _, cmd := m.Update(tui.MouseEvent{X: 2, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}); cmd != nil {
		t.Fatal("a click moved an item of disabled lists")
	}
	m.Disabled = false
	if _, cmd := m.Update(key(tui.KeyTab)); cmd != nil {
		t.Fatal("Tab was handled")
	}
	var empty Model
	empty.Focus()
	for _, k := range []tui.Key{key(tui.KeyEnter), ctrlA(), key(tui.KeyDown), key(tui.KeyEnd)} {
		if next, cmd := empty.Update(k); cmd != nil || next.Cursor(Left) != -1 {
			t.Fatalf("empty lists handled %v", k)
		}
	}
	if len(m.Bindings()) != 8 {
		t.Fatalf("Bindings() = %+v", m.Bindings())
	}
}

// With a Height the lists scroll to keep the cursor in view.
func TestHeightScrollsToKeepTheCursorInView(t *testing.T) {
	m := New("a", "b", "c", "d", "e")
	m.Height = 2
	m.Focus()
	if m.Rows() != 3 {
		t.Fatalf("Rows() = %d", m.Rows())
	}
	for range 3 {
		m, _ = m.Update(key(tui.KeyDown))
	}
	if got := rows(m); got[1] != " c              │" || got[2] != "<d>             │" {
		t.Errorf("scrolled down = %q", got)
	}
	m, _ = m.Update(key(tui.KeyUp)) // still in view: no scroll
	if got := rows(m); got[1] != "<c>             │" {
		t.Errorf("after Up = %q", got)
	}
	m, _ = m.Update(key(tui.KeyHome))
	if got := rows(m); got[1] != "<a>             │" || got[2] != " b              │" {
		t.Errorf("after Home = %q", got)
	}
	m, _ = m.Update(key(tui.KeyEnd))
	m, _ = m.Update(key(tui.KeyEnter)) // e leaves; no blank row is left under d
	if got := rows(m); got[1] != " c              │  e" || got[2] != "<d>             │" {
		t.Errorf("after the last item moved = %q", got)
	}
	m.SetCursor(Left, 0)
	if got := rows(m); got[1] != "<a>             │  e" {
		t.Errorf("after SetCursor = %q", got)
	}
	m.SetCursor(Right, 0)
	if m.Active() != Right {
		t.Error("SetCursor did not change list")
	}
	m.SetCursor(Right, 4)
	m.SetCursor(Left, -1)
	m.SetCursor(Side(7), 0)
	if m.Active() != Right || m.Cursor(Right) != 0 {
		t.Error("SetCursor took an index or a list out of range")
	}
}

func TestColumnWidthCutsLongText(t *testing.T) {
	m := New("Pomegranates", "Fig")
	m.ColumnWidth = 8
	want := []string{" Avail…  │  Chose…", " Pomeg…  │", " Fig     │"}
	if got := rows(m); !slices.Equal(got, want) {
		t.Errorf("View =\n%s", strings.Join(got, "\n"))
	}
	if m.Width() != 19 {
		t.Errorf("Width() = %d", m.Width())
	}
	for _, w := range []int{1, 2} { // room for the two end cells and no text
		m.ColumnWidth = w
		if got := rows(m)[1]; got != "   │" || m.Width() != 7 {
			t.Errorf("ColumnWidth %d = %q, Width() %d", w, got, m.Width())
		}
	}
	m.ColumnWidth = 8
	m.Theme.Glyphs = theme.ASCIIGlyphSet()
	if got := rows(m)[1]; got != " Pomeg~  |" {
		t.Errorf("ASCII = %q", got)
	}
}

// " Available (3)  │  Chosen (1) ": the left list owns local columns 0..14,
// the rule 15..17, the right list 18..29. Drawn at column 10, row 5.
func TestAClickMovesTheItemUnderIt(t *testing.T) {
	m := New("Apples", "Bananas", "Cherries")
	m.ID = "fruit"
	m.Right = []string{"Dates"}
	m.Mouse, m.Bounds = true, hittest.Rect{X: 10, Y: 5, W: 30, H: 4}
	press := func(x, y int, b tui.MouseButton, a tui.MouseAction) (Model, tui.Cmd) {
		return m.Update(tui.MouseEvent{X: x, Y: y, Button: b, Action: a})
	}
	next, cmd := press(12, 7, tui.MouseButtonLeft, tui.MouseActionPress)
	if c, ok := changed(cmd); !ok || !slices.Equal(c.Moved, []string{"Bananas"}) || c.To != Right || len(next.Left) != 2 {
		t.Fatalf("click on Bananas: %+v", c)
	}
	next, cmd = press(30, 6, tui.MouseButtonLeft, tui.MouseActionPress)
	if c, ok := changed(cmd); !ok || !slices.Equal(c.Moved, []string{"Dates"}) || c.To != Left || len(next.Left) != 4 {
		t.Fatalf("click on Dates: %+v", c)
	}
	for name, at := range map[string][2]int{
		"a title": {12, 5}, "the rule": {26, 6}, "a blank row": {30, 7}, "outside": {70, 6}, "past the right list": {45, 6},
	} {
		if _, cmd := press(at[0], at[1], tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
			t.Errorf("a click on %s moved an item", name)
		}
	}
	if _, cmd := press(12, 7, tui.MouseButtonRight, tui.MouseActionPress); cmd != nil {
		t.Error("the right button moved an item")
	}
	if _, cmd := press(12, 7, tui.MouseButtonLeft, tui.MouseActionRelease); cmd != nil {
		t.Error("a release moved an item")
	}
	m.Bounds.W = 60
	if _, cmd := press(45, 6, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Error("a click past the lists moved an item")
	}
	m.Mouse = false
	if _, cmd := press(12, 7, tui.MouseButtonLeft, tui.MouseActionPress); cmd != nil {
		t.Error("the pointer moved an item with Mouse off")
	}
}

func TestTheWheelScrollsTheListUnderIt(t *testing.T) {
	m := New("a", "b", "c", "d", "e")
	m.Height = 2
	m.Focus()
	m.Mouse, m.Bounds = true, hittest.Rect{W: 30, H: 3}
	wheel := func(b tui.MouseButton) {
		var cmd tui.Cmd
		m, cmd = m.Update(tui.MouseEvent{X: 2, Y: 1, Button: b, Action: tui.MouseActionPress})
		if cmd != nil {
			t.Fatal("the wheel moved an item")
		}
	}
	wheel(tui.MouseButtonWheelDown)
	wheel(tui.MouseButtonWheelDown)
	// The cursor was on a, which scrolled out; it follows to the first row.
	if got := rows(m); got[1] != "<c>             │" || got[2] != " d              │" {
		t.Errorf("after two notches down = %q", got)
	}
	for range 5 {
		wheel(tui.MouseButtonWheelDown)
	}
	if got := rows(m); got[1] != "<d>             │" || got[2] != " e              │" {
		t.Errorf("at the end = %q", got)
	}
	m, _ = m.Update(key(tui.KeyDown))
	for range 9 {
		wheel(tui.MouseButtonWheelUp)
	}
	// The cursor was on e; it follows to the last row.
	if got := rows(m); got[1] != " a              │" || got[2] != "<b>             │" {
		t.Errorf("back at the top = %q", got)
	}
}

func TestACustomKeyMapReplacesTheDefault(t *testing.T) {
	m := fruit()
	m.KeyMap = KeyMap{Move: keymap.NewBinding("send", "s")}
	if _, cmd := m.Update(key(tui.KeyEnter)); cmd != nil {
		t.Fatal("Enter still moved with a custom KeyMap")
	}
	if _, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "s"}); cmd == nil {
		t.Fatal("the custom key did not move the item")
	}
	z := Model{Left: []string{"A", "B"}, Theme: theme.DarkTheme()}
	z.Focus()
	if next, _ := z.Update(key(tui.KeyDown)); next.Cursor(Left) != 1 {
		t.Fatal("a struct-literal Model with no KeyMap ignored Down")
	}
}

func TestLinearizeListsEveryItem(t *testing.T) {
	m := New("a", "b", "c")
	m.Right = []string{"d"}
	m.Height = 1
	want := "Available (3)\n  a\n  b\n  c\nChosen (1)\n  d"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s", got)
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
	if s := n.Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded}); s != (layout.Size{W: 30, H: 3}) {
		t.Fatalf("Measure = %+v, want 30x3", s)
	}
}
