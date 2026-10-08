package hovercard

import (
	"slices"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// grid is a w x h rectangle of dots, a base to draw over.
func grid(w, h int) string {
	rows := make([]string, h)
	for i := range rows {
		rows[i] = strings.Repeat(".", w)
	}
	return strings.Join(rows, "\n")
}

func rows(s string) []string { return strings.Split(ansi.StripANSI(s), "\n") }

func move(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionMotion, Button: tui.MouseButtonNone}
}

func press(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionPress, Button: tui.MouseButtonLeft}
}

func closed(cmd tui.Cmd) (ClosedMsg, bool) {
	if cmd == nil {
		return ClosedMsg{}, false
	}
	c, ok := cmd().(ClosedMsg)
	return c, ok
}

// card describes a name drawn at columns 2..4 of row 0 of a 22 x 8 view.
// Open, it takes columns 2..18 of rows 1..6.
func card() Model {
	m := New("Ada", "Wrote the\nfirst program", hittest.Rect{X: 2, Y: 0, W: 3, H: 1})
	m.ID = "ada"
	m.Mouse, m.Bounds = true, hittest.Rect{W: 22, H: 8}
	return m
}

func TestItIsDrawnUnderItsTarget(t *testing.T) {
	m := card()
	base := grid(22, 8)
	if m.Open() || m.Render(base) != base || m.Rect() != (hittest.Rect{}) || m.Linearize() != "" || m.Bindings() != nil {
		t.Fatal("a new card is not closed")
	}
	m.Show()
	want := []string{
		"......................",
		"..┌───────────────┐...",
		"..│ Ada           │...",
		"..│               │...",
		"..│ Wrote the     │...",
		"..│ first program │...",
		"..└───────────────┘...",
		"......................",
	}
	if got := rows(m.Render(base)); !slices.Equal(got, want) {
		t.Errorf("Render =\n%s", strings.Join(got, "\n"))
	}
	if got := m.Rect(); got != (hittest.Rect{X: 2, Y: 1, W: 17, H: 6}) {
		t.Errorf("Rect = %+v", got)
	}
	if got := m.Linearize(); got != "Ada\n\nWrote the\nfirst program" {
		t.Errorf("Linearize = %q", got)
	}
	if len(m.Bindings()) != 1 {
		t.Errorf("Bindings() = %+v", m.Bindings())
	}
	m.Hide()
	if m.Open() {
		t.Error("Hide left it open")
	}
}

func TestTitleAndContentAreEachOptional(t *testing.T) {
	m := card()
	m.Show()
	m.Title = ""
	if got := rows(m.Render(grid(22, 8))); got[2] != "..│ Wrote the     │..." || got[4] != "..└───────────────┘..." {
		t.Errorf("no title =\n%s", strings.Join(got, "\n"))
	}
	m.Title, m.Content = "Ada", ""
	if got := rows(m.Render(grid(22, 8))); got[2] != "..│ Ada │............." || got[3] != "..└─────┘............." {
		t.Errorf("no content =\n%s", strings.Join(got, "\n"))
	}
}

// With no room to the right it moves left; with none below it goes above.
func TestItStaysInsideTheBase(t *testing.T) {
	m := card()
	m.Show()
	m.Target = hittest.Rect{X: 19, Y: 7, W: 3, H: 1}
	got := rows(m.Render(grid(22, 8)))
	if got[1] != ".....┌───────────────┐" || got[6] != ".....└───────────────┘" || got[7] != strings.Repeat(".", 22) {
		t.Errorf("Render =\n%s", strings.Join(got, "\n"))
	}
	m.Bounds.H = 3 // neither fits: drawn from the top and cut
	m.Target.Y = 1
	if got := rows(m.Render(grid(22, 3))); got[0] != ".....┌───────────────┐" || len(got) != 3 {
		t.Errorf("no room = %q", got)
	}
}

// The content keeps its styling and loses any other escape sequence; Width
// wraps it.
func TestContentIsStyledSafeAndWrapped(t *testing.T) {
	m := card()
	m.Show()
	m.Content = ansi.NewStyle().Bold().Render("bold") + "\x1b[2J and\tmore words here"
	m.Width = 10
	out := m.Render(grid(22, 9))
	if !strings.Contains(out, "\x1b[1m") || strings.Contains(out, "\x1b[2J") || strings.Contains(out, "\t") {
		t.Errorf("content = %q", out)
	}
	for _, r := range rows(out)[4:6] {
		if w := ansi.Width(strings.Trim(r, ".")); w > 14 {
			t.Errorf("a wrapped row is %d cells: %q", w, r)
		}
	}
}

func TestItStaysOpenWhileThePointerIsOnTheTargetOrTheCard(t *testing.T) {
	m := card()
	m, cmd := m.Update(move(3, 0))
	if !m.Open() || cmd != nil {
		t.Fatal("the pointer on the target did not open it")
	}
	for _, at := range [][2]int{{4, 0}, {3, 1}, {18, 6}} {
		m, cmd = m.Update(move(at[0], at[1]))
		if !m.Open() || cmd != nil {
			t.Fatalf("the pointer at %v closed it", at)
		}
	}
	m, cmd = m.Update(press(5, 3)) // a press on the card
	if !m.Open() || cmd != nil {
		t.Fatal("a press on the card closed it")
	}
	m, cmd = m.Update(tui.MouseEvent{X: 20, Y: 7, Action: tui.MouseActionRelease, Button: tui.MouseButtonLeft})
	if !m.Open() || cmd != nil {
		t.Fatal("a release outside closed it")
	}
	m, cmd = m.Update(move(20, 7))
	if c, ok := closed(cmd); !ok || c.ID != "ada" || m.Open() {
		t.Fatalf("the pointer leaving both did not close it: %+v", c)
	}
	m, cmd = m.Update(move(20, 7))
	if cmd != nil || m.Open() {
		t.Fatal("a closed card reported closing again")
	}
	m, _ = m.Update(move(3, 0))
	m, cmd = m.Update(press(20, 7))
	if _, ok := closed(cmd); !ok || m.Open() {
		t.Fatal("a press outside did not close it")
	}
	// A drag across the target is not a hover.
	m, _ = m.Update(tui.MouseEvent{X: 3, Y: 0, Action: tui.MouseActionMotion, Button: tui.MouseButtonLeft})
	if m.Open() {
		t.Fatal("a drag opened it")
	}
	m.Mouse = false
	m, _ = m.Update(move(3, 0))
	if m.Open() {
		t.Fatal("the pointer opened it with Mouse off")
	}
}

func TestEscClosesIt(t *testing.T) {
	m := card()
	if _, cmd := m.Update(tui.Key{Type: tui.KeyEsc}); cmd != nil {
		t.Fatal("Esc on a closed card returned a Cmd")
	}
	m.Show()
	if next, cmd := m.Update(tui.Key{Type: tui.KeyEnter}); cmd != nil || !next.Open() {
		t.Fatal("Enter closed it")
	}
	next, cmd := m.Update(tui.Key{Type: tui.KeyEsc})
	if c, ok := closed(cmd); !ok || c.ID != "ada" || next.Open() {
		t.Fatalf("Esc did not close it: %+v", c)
	}
	m.KeyMap = KeyMap{Close: keymap.NewBinding("close", "q")}
	if _, cmd := m.Update(tui.Key{Type: tui.KeyEsc}); cmd != nil {
		t.Fatal("Esc still closed it with a custom KeyMap")
	}
	if _, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "q"}); cmd == nil {
		t.Fatal("the custom key did not close it")
	}
	z := Model{Title: "T", Theme: theme.DarkTheme()}
	z.Show()
	if _, cmd := z.Update(tui.Key{Type: tui.KeyEsc}); cmd == nil {
		t.Fatal("a struct-literal Model with no KeyMap ignored Esc")
	}
}

var _ tui.Linearizer = Model{}

func TestThemeTokensAndLayoutNode(t *testing.T) {
	m := card().SetTheme(theme.LightTheme())
	if m.Theme.Primary != theme.LightTheme().Primary {
		t.Error("SetTheme did not apply the theme")
	}
	red := ansi.RGB{R: 255}
	m = m.WithTokens(theme.Tokens{Border: red})
	if m.Tokens().Border != red {
		t.Error("WithTokens did not override the border colour")
	}
	free := layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded}
	if s := m.LayoutNode().Measure(free); s.W != 0 {
		t.Fatalf("closed: Measure = %+v", s)
	}
	m.Show()
	if s := m.LayoutNode().Measure(free); s != (layout.Size{W: 17, H: 6}) {
		t.Fatalf("Measure = %+v, want 17x6", s)
	}
}
