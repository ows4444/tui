package window

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func rect(x, y, w, h int) hittest.Rect { return hittest.Rect{X: x, Y: y, W: w, H: h} }

func desk() Model {
	m := New(
		Window{ID: "a", Title: "Alpha", Rect: rect(0, 0, 20, 6), Content: layout.Block("one\ntwo")},
		Window{ID: "b", Title: "Beta", Rect: rect(10, 2, 20, 6), Content: layout.Block("three")},
	)
	m.Mouse = true
	m.Bounds = rect(5, 3, 60, 20) // the desktop is not at the screen origin
	return m
}

func press(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
}
func motion(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionMotion}
}
func release(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease}
}

// send runs the Cmd Update returns and gives back its Msg.
func send(m Model, msg tui.Msg) (Model, tui.Msg) {
	m, cmd := m.Update(msg)
	if cmd == nil {
		return m, nil
	}
	return m, cmd()
}

func dragAcross(m Model, from, to [2]int) (Model, tui.Msg) {
	m, _ = send(m, press(from[0], from[1]))
	m, _ = send(m, motion(to[0], to[1]))
	return send(m, release(to[0], to[1]))
}

func TestRenderOverlapsFrontOverBack(t *testing.T) {
	m := New(
		Window{ID: "a", Title: "Alpha", Rect: rect(0, 0, 12, 4), Content: layout.Block("one\ntwo")},
		Window{ID: "b", Title: "Beta", Rect: rect(6, 2, 12, 4), Content: layout.Block("three")},
	)
	m.Bounds = rect(0, 0, 20, 7)
	m.Theme = theme.DarkTheme().ASCII()
	got := strings.Split(ansi.StripANSI(m.View()), "\n")
	want := []string{
		"+ Alpha [x]+        ",
		"|one       |        ",
		"|two  + Beta -[x]+  ",
		"+-----|three     |  ",
		"      |          |  ",
		"      +----------+  ",
		"                    ",
	}
	if len(got) != len(want) {
		t.Fatalf("%d rows:\n%s", len(got), strings.Join(got, "\n"))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestHitAndContentRect(t *testing.T) {
	m := desk()
	// Window b is at desktop (10,2), so on screen at (15,5).
	for _, c := range []struct {
		x, y int
		id   string
		part Part
	}{
		{20, 5, "b", PartTitle},
		{15, 5, "b", PartTitle}, // the corner
		{15 + 16, 5, "b", PartClose},
		{15, 6, "b", PartEdge},
		{20, 5 + 5, "b", PartEdge},
		{20, 7, "b", PartBody},
		{6, 4, "a", PartBody},
		{5, 3, "a", PartTitle},
		{1, 1, "", PartNone},
		{60, 20, "", PartNone},
	} {
		if id, part := m.Hit(c.x, c.y); id != c.id || part != c.part {
			t.Errorf("Hit(%d,%d) = %q,%v; want %q,%v", c.x, c.y, id, part, c.id, c.part)
		}
	}
	if r, ok := m.ContentRect("b"); !ok || r != rect(16, 6, 18, 4) {
		t.Errorf("ContentRect(b) = %v, %v", r, ok)
	}
	if _, ok := m.ContentRect("zzz"); ok {
		t.Error("ContentRect found a window that is not there")
	}
}

func TestDragTitleMovesAndReports(t *testing.T) {
	m := desk()
	// Hold b by the cell 4 columns into its title bar and move the pointer 3 right, 2 down.
	m, msg := dragAcross(m, [2]int{19, 5}, [2]int{22, 7})
	b, _ := m.Window("b")
	if b.Rect != rect(13, 4, 20, 6) {
		t.Fatalf("b at %v", b.Rect)
	}
	if c, ok := msg.(ChangedMsg); !ok || c.ID != "b" || c.Rect != b.Rect {
		t.Fatalf("msg = %#v", msg)
	}
	if m.Dragging() {
		t.Error("still dragging after the release")
	}
}

func TestDragIsClampedToTheDesktop(t *testing.T) {
	m := desk()
	m, _ = dragAcross(m, [2]int{19, 5}, [2]int{500, 500})
	if b, _ := m.Window("b"); b.Rect.X != 40 || b.Rect.Y != 14 {
		t.Fatalf("b at %v; want the bottom right corner of a 60x20 desktop", b.Rect)
	}
	m, _ = dragAcross(m, [2]int{5 + 40 + 4, 3 + 14}, [2]int{-50, -50})
	if b, _ := m.Window("b"); b.Rect.X != 0 || b.Rect.Y != 0 {
		t.Fatalf("b at %v; want the top left", b.Rect)
	}
}

func TestPressRaisesAndFocuses(t *testing.T) {
	m := desk()
	m, msg := send(m, press(7, 3)) // the title bar of a, which b does not cover
	if m.Focused() != "a" {
		t.Fatalf("focused %q", m.Focused())
	}
	if f, ok := msg.(FocusedMsg); !ok || f.ID != "a" {
		t.Fatalf("msg = %#v", msg)
	}
	if _, msg := send(m, press(7, 3)); msg != nil {
		t.Errorf("a press on the focused window reported %#v", msg)
	}
	// A press in the body raises but does not start a drag.
	m, _ = send(desk(), press(6, 4))
	if m.Focused() != "a" || m.Dragging() {
		t.Errorf("focused %q dragging %v", m.Focused(), m.Dragging())
	}
}

func TestCloseBox(t *testing.T) {
	m := desk()
	m, msg := send(m, press(15+16, 5))
	if c, ok := msg.(ClosedMsg); !ok || c.ID != "b" {
		t.Fatalf("msg = %#v", msg)
	}
	if len(m.Windows) != 1 || m.Focused() != "a" {
		t.Fatalf("windows %d, focused %q", len(m.Windows), m.Focused())
	}
	m = desk()
	m.Windows[1].NoClose = true
	if _, msg := send(m, press(15+16, 5)); msg != nil {
		t.Errorf("NoClose window reported %#v", msg)
	}
}

func TestResizeFromEdges(t *testing.T) {
	// The bottom right corner: b is 20x6 at screen (15,5); its corner is (34,10).
	m, _ := dragAcross(desk(), [2]int{34, 10}, [2]int{39, 12})
	if b, _ := m.Window("b"); b.Rect != rect(10, 2, 25, 8) {
		t.Fatalf("b = %v", b.Rect)
	}
	// The left edge keeps the right edge where it was.
	m, _ = dragAcross(desk(), [2]int{15, 7}, [2]int{12, 7})
	if b, _ := m.Window("b"); b.Rect != rect(7, 2, 23, 6) {
		t.Fatalf("b = %v", b.Rect)
	}
	// It stops at the minimum width, with the right edge still fixed.
	m, _ = dragAcross(desk(), [2]int{15, 7}, [2]int{60, 7})
	if b, _ := m.Window("b"); b.Rect.W != 10 || b.Rect.X+b.Rect.W != 30 {
		t.Fatalf("b = %v", b.Rect)
	}
	// The bottom edge only changes the height.
	m, _ = dragAcross(desk(), [2]int{20, 10}, [2]int{20, 3})
	if b, _ := m.Window("b"); b.Rect != rect(10, 2, 20, 3) {
		t.Fatalf("b = %v", b.Rect)
	}
}

func TestMinSizeOfAWindowIsHonoured(t *testing.T) {
	m := desk()
	m.Windows[1].MinW, m.Windows[1].MinH = 18, 5
	m, _ = dragAcross(m, [2]int{34, 10}, [2]int{20, 6})
	if b, _ := m.Window("b"); b.Rect.W != 18 || b.Rect.H != 5 {
		t.Fatalf("b = %v", b.Rect)
	}
}

func TestFixedWindowsAreNotDragged(t *testing.T) {
	m := desk()
	m.Windows[1].Fixed = true
	before, _ := m.Window("b")
	m, msg := dragAcross(m, [2]int{19, 5}, [2]int{30, 12})
	if after, _ := m.Window("b"); after.Rect != before.Rect || msg != nil {
		t.Fatalf("a fixed window moved to %v, msg %#v", after.Rect, msg)
	}
	m, _ = send(m, tui.Key{Type: tui.KeyLeft, Mod: input.ModAlt})
	if after, _ := m.Window("b"); after.Rect != before.Rect {
		t.Fatalf("a fixed window moved by key to %v", after.Rect)
	}
}

func TestMouseOffAndNoBoundsIgnoreTheMouse(t *testing.T) {
	m := desk()
	m.Mouse = false
	if m2, msg := send(m, press(19, 5)); msg != nil || m2.Dragging() || m2.Focused() != "b" {
		t.Error("Mouse off still handled a press")
	}
	m = desk()
	m.Bounds = hittest.Rect{}
	if m2, _ := send(m, press(19, 5)); m2.Dragging() {
		t.Error("an empty Bounds still started a drag")
	}
}

func TestKeysMoveAndResizeTheFocusedWindow(t *testing.T) {
	alt := func(t tui.KeyType) tui.Key { return tui.Key{Type: t, Mod: input.ModAlt} }
	altShift := func(t tui.KeyType) tui.Key { return tui.Key{Type: t, Mod: input.ModAlt | input.ModShift} }
	m := desk()
	m.Step = 2
	for _, c := range []struct {
		key  tui.Key
		want hittest.Rect
	}{
		{alt(tui.KeyRight), rect(12, 2, 20, 6)},
		{alt(tui.KeyDown), rect(12, 4, 20, 6)},
		{alt(tui.KeyLeft), rect(10, 4, 20, 6)},
		{alt(tui.KeyUp), rect(10, 2, 20, 6)},
		{altShift(tui.KeyRight), rect(10, 2, 22, 6)},
		{altShift(tui.KeyDown), rect(10, 2, 22, 8)},
		{altShift(tui.KeyLeft), rect(10, 2, 20, 8)},
		{altShift(tui.KeyUp), rect(10, 2, 20, 6)},
	} {
		var msg tui.Msg
		m, msg = send(m, c.key)
		if b, _ := m.Window("b"); b.Rect != c.want {
			t.Fatalf("after %v b = %v; want %v", c.key.String(), b.Rect, c.want)
		}
		if ch, ok := msg.(ChangedMsg); !ok || ch.Rect != c.want {
			t.Fatalf("after %v msg = %#v", c.key.String(), msg)
		}
	}
	// At the edge a move changes nothing and reports nothing.
	m.SetRect("b", rect(40, 14, 20, 6))
	if _, msg := send(m, alt(tui.KeyRight)); msg != nil {
		t.Errorf("a move into the wall reported %#v", msg)
	}
}

func altKey(r rune) tui.Key {
	return tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r, Mod: input.ModAlt}
}

func TestNextPrevCloseAndMaximize(t *testing.T) {
	m := desk()
	m, msg := send(m, altKey(']'))
	if m.Focused() != "a" || msg != (FocusedMsg{ID: "a"}) {
		t.Fatalf("Next: focused %q, msg %#v", m.Focused(), msg)
	}
	m, msg = send(m, altKey('['))
	if m.Focused() != "b" || msg != (FocusedMsg{ID: "b"}) {
		t.Fatalf("Prev: focused %q, msg %#v", m.Focused(), msg)
	}
	m, msg = send(m, altKey('m'))
	b, _ := m.Window("b")
	if b.Rect != rect(0, 0, 60, 20) || !b.Maximized() {
		t.Fatalf("maximize: %v %v", b.Rect, b.Maximized())
	}
	if _, ok := msg.(ChangedMsg); !ok {
		t.Fatalf("msg = %#v", msg)
	}
	m, _ = send(m, altKey('m'))
	if b, _ := m.Window("b"); b.Rect != rect(10, 2, 20, 6) || b.Maximized() {
		t.Fatalf("restore: %v %v", b.Rect, b.Maximized())
	}
	m, msg = send(m, altKey('w'))
	if msg != (ClosedMsg{ID: "b"}) || len(m.Windows) != 1 {
		t.Fatalf("Close: %d windows, msg %#v", len(m.Windows), msg)
	}
}

func TestModelCopiesAreIndependent(t *testing.T) {
	m := desk()
	moved, _ := send(m, altKey(']'))
	if m.Focused() != "b" || moved.Focused() != "a" {
		t.Fatal("changing a copy changed the original")
	}
	added := m
	added.Add(Window{ID: "c", Rect: rect(1, 1, 12, 4)})
	if len(m.Windows) != 2 || len(added.Windows) != 3 {
		t.Fatalf("Add shared the slice: %d, %d", len(m.Windows), len(added.Windows))
	}
}

func TestAddRemoveRaiseSetRect(t *testing.T) {
	m := desk()
	m.Add(Window{ID: "c", Rect: rect(-5, 100, 3, 1)})
	c, _ := m.Window("c")
	if c.Rect != rect(0, 17, 10, 3) {
		t.Fatalf("Add did not clamp: %v", c.Rect)
	}
	if !m.Raise("a") || m.Focused() != "a" || m.Raise("zzz") {
		t.Fatal("Raise")
	}
	if !m.Remove("a") || m.Remove("a") || len(m.Windows) != 2 {
		t.Fatal("Remove")
	}
	m.SetRect("b", rect(1000, 1000, 1000, 1000))
	if b, _ := m.Window("b"); b.Rect != rect(0, 0, 60, 20) {
		t.Fatalf("SetRect = %v", b.Rect)
	}
}

func TestTitleIsSanitizedAndTruncated(t *testing.T) {
	m := New(Window{ID: "a", Title: "evil\x1b]52;c;AAAA\x07\ntitle that is far too long to fit", Rect: rect(0, 0, 16, 4)})
	m.Bounds = rect(0, 0, 20, 5)
	out := m.View()
	if strings.Contains(out, "\x1b]52") || strings.Contains(out, "\x07") {
		t.Fatalf("the title reached the screen unsanitized: %q", out)
	}
	for i, l := range strings.Split(ansi.StripANSI(out), "\n") {
		if ansi.Width(l) != 20 {
			t.Errorf("row %d is %d wide", i, ansi.Width(l))
		}
	}
}

func TestLayoutNodeFitsTheRoomAndClipsWindows(t *testing.T) {
	m := desk()
	m.Bounds = rect(0, 0, 25, 6)
	for _, s := range []layout.Size{{W: 25, H: 6}, {W: 8, H: 3}, {W: 1, H: 1}, {W: 0, H: 4}} {
		lines := strings.Split(m.LayoutNode().Render(s), "\n")
		if s.W <= 0 {
			continue
		}
		if len(lines) != s.H {
			t.Errorf("%v: %d rows", s, len(lines))
		}
		for _, l := range lines {
			if ansi.Width(l) != s.W {
				t.Errorf("%v: a row is %d wide", s, ansi.Width(l))
			}
		}
	}
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got.W != 30 || got.H != 8 {
		t.Errorf("Measure = %v", got)
	}
}

func TestLinearize(t *testing.T) {
	out := desk().Linearize()
	if !strings.Contains(out, "Windows, 2") || !strings.Contains(out, `Window "Beta" (focused)`) || !strings.Contains(out, `Window "Alpha"`) {
		t.Fatalf("Linearize:\n%s", out)
	}
	if strings.Index(out, "Beta") > strings.Index(out, "Alpha") {
		t.Error("the focused window should be described first")
	}
}

func TestBackgroundIsDrawnBehind(t *testing.T) {
	m := New(Window{ID: "a", Rect: rect(2, 1, 10, 3)})
	m.Bounds = rect(0, 0, 14, 5)
	m.Theme = theme.DarkTheme().ASCII()
	m.Background = layout.Block(strings.TrimRight(strings.Repeat(strings.Repeat(".", 14)+"\n", 5), "\n"))
	rows := strings.Split(ansi.StripANSI(m.View()), "\n")
	if rows[0] != strings.Repeat(".", 14) || !strings.HasPrefix(rows[2], "..|") {
		t.Fatalf("rows:\n%s", strings.Join(rows, "\n"))
	}
}

func TestSetThemeAndTokens(t *testing.T) {
	m := desk()
	if n := len(m.Bindings()); n != 12 {
		t.Fatalf("%d bindings", n)
	}
	if m.WithTokens(m.Tokens()).SetTheme(m.Theme).Theme.Primary == nil {
		t.Fatal("SetTheme lost the theme")
	}
}
