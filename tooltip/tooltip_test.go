package tooltip

import (
	"slices"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
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

// hint describes a thing drawn at columns 2..5 of row 0 of a 20 x 5 view.
func hint() Model {
	m := New("Save file", hittest.Rect{X: 2, Y: 0, W: 4, H: 1})
	m.Mouse, m.Bounds = true, hittest.Rect{W: 20, H: 5}
	return m
}

func TestItIsDrawnUnderItsTarget(t *testing.T) {
	m := hint()
	base := grid(20, 5)
	if m.Open() || m.Render(base) != base || m.Rect() != (hittest.Rect{}) || m.Linearize() != "" {
		t.Fatal("a new tooltip is not closed")
	}
	m.Show()
	want := []string{
		"....................",
		"..┌───────────┐.....",
		"..│ Save file │.....",
		"..└───────────┘.....",
		"....................",
	}
	if got := rows(m.Render(base)); !slices.Equal(got, want) {
		t.Errorf("Render =\n%s", strings.Join(got, "\n"))
	}
	if got := m.Rect(); got != (hittest.Rect{X: 2, Y: 1, W: 13, H: 3}) {
		t.Errorf("Rect = %+v", got)
	}
	if got := m.Linearize(); got != "Hint: Save file" {
		t.Errorf("Linearize = %q", got)
	}
	m.Hide()
	if m.Open() {
		t.Error("Hide left it open")
	}
}

// With no room to the right it moves left; with none below it goes above.
func TestItStaysInsideTheBase(t *testing.T) {
	m := hint()
	m.Show()
	m.Target = hittest.Rect{X: 16, Y: 4, W: 4, H: 1}
	want := []string{
		"....................",
		".......┌───────────┐",
		".......│ Save file │",
		".......└───────────┘",
		"....................",
	}
	if got := rows(m.Render(grid(20, 5))); !slices.Equal(got, want) {
		t.Errorf("Render =\n%s", strings.Join(got, "\n"))
	}
	// Neither fits in two rows: it is drawn from the top and cut.
	m.Bounds.H = 2
	m.Target.Y = 1
	if got := rows(m.Render(grid(20, 2))); got[0] != ".......┌───────────┐" || len(got) != 2 {
		t.Errorf("no room = %q", got)
	}
	// The base need not start at the screen's corner.
	m = hint()
	m.Show()
	m.Bounds = hittest.Rect{X: 2, Y: 0, W: 20, H: 5}
	if got := rows(m.Render(grid(20, 5)))[1]; got != "┌───────────┐......." {
		t.Errorf("offset base = %q", got)
	}
	if got := m.Rect(); got != (hittest.Rect{X: 2, Y: 1, W: 13, H: 3}) {
		t.Errorf("Rect over an offset base = %+v", got)
	}
}

func TestWidthWrapsTheText(t *testing.T) {
	m := hint()
	m.Text = "Save the file now"
	m.Width = 8
	m.Show()
	got := rows(m.Render(grid(20, 6)))
	if got[2] != "..│ Save the │......" || got[3] != "..│ file now │......" {
		t.Errorf("wrapped =\n%s", strings.Join(got, "\n"))
	}
	m.Width = 0
	m.Text = "one\nthree"
	if got := rows(m.Render(grid(20, 6))); got[2] != "..│ one   │........." || got[3] != "..│ three │........." {
		t.Errorf("two lines =\n%s", strings.Join(got, "\n"))
	}
}

func TestThePointerOverTheTargetShowsIt(t *testing.T) {
	m := hint()
	m, cmd := m.Update(move(3, 0))
	if !m.Open() || cmd != nil {
		t.Fatal("the pointer on the target did not show it")
	}
	m, _ = m.Update(move(5, 0))
	if !m.Open() {
		t.Fatal("moving within the target hid it")
	}
	m, _ = m.Update(move(6, 0))
	if m.Open() {
		t.Fatal("the pointer leaving the target did not hide it")
	}
	m, _ = m.Update(move(3, 0))
	m, _ = m.Update(tui.MouseEvent{X: 3, Y: 0, Action: tui.MouseActionPress, Button: tui.MouseButtonLeft})
	if m.Open() {
		t.Fatal("a press on the target did not hide it")
	}
	// A drag across the target is not a hover.
	m, _ = m.Update(tui.MouseEvent{X: 3, Y: 0, Action: tui.MouseActionMotion, Button: tui.MouseButtonLeft})
	if m.Open() {
		t.Fatal("a drag showed it")
	}
	m.Mouse = false
	m, _ = m.Update(move(3, 0))
	if m.Open() {
		t.Fatal("the pointer showed it with Mouse off")
	}
	m.Show()
	m, _ = m.Update(move(15, 3))
	if !m.Open() {
		t.Fatal("the pointer hid it with Mouse off")
	}
}

func TestAKeyHidesIt(t *testing.T) {
	m := hint()
	m.Show()
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "a", Action: tui.KeyRelease})
	if !m.Open() {
		t.Fatal("a key release hid it")
	}
	m, _ = m.Update(struct{}{})
	if !m.Open() {
		t.Fatal("a message that is neither key nor mouse hid it")
	}
	m, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "a"})
	if m.Open() || cmd != nil {
		t.Fatal("a key did not hide it")
	}
}

var _ tui.Linearizer = Model{}

func TestThemeTokensAndLayoutNode(t *testing.T) {
	m := hint().SetTheme(theme.LightTheme())
	if m.Theme.Primary != theme.LightTheme().Primary {
		t.Error("SetTheme did not apply the theme")
	}
	red := ansi.RGB{R: 255}
	m = m.WithTokens(theme.Tokens{Info: red})
	if m.Tokens().Info != red {
		t.Error("WithTokens did not override the info colour")
	}
	free := layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded}
	if s := m.LayoutNode().Measure(free); s.W != 0 {
		t.Fatalf("closed: Measure = %+v", s)
	}
	m.Show()
	if s := m.LayoutNode().Measure(free); s != (layout.Size{W: 13, H: 3}) {
		t.Fatalf("Measure = %+v, want 13x3", s)
	}
}
