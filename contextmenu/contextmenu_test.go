package contextmenu

import (
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
func runeKey(r rune) tui.Key    { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func sample() Model {
	m := New(
		Item{Label: "Cut", Value: "cut", Accel: 't', Shortcut: "ctrl+x"},
		Item{Label: "Copy", Value: "copy", Accel: 'c', Shortcut: "ctrl+c"},
		Item{Separator: true},
		Item{Label: "Paste", Value: "paste", Disabled: true, Accel: 'p'},
		Item{Label: "Delete", Value: "delete"},
	)
	m.Show(5, 3)
	return m
}

func msgOf(t *testing.T, cmd tui.Cmd) tui.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("nil cmd")
	}
	return cmd()
}

func TestCursorSkipsSeparatorAndDisabled(t *testing.T) {
	m := sample()
	if m.Cursor() != 0 {
		t.Fatalf("start %d", m.Cursor())
	}
	m, _ = m.Update(key(tui.KeyDown))
	m, _ = m.Update(key(tui.KeyDown)) // skips separator and disabled
	if m.Cursor() != 4 {
		t.Fatalf("cursor %d, want 4", m.Cursor())
	}
	m, _ = m.Update(key(tui.KeyDown)) // clamps
	if m.Cursor() != 4 {
		t.Fatalf("cursor %d after clamp", m.Cursor())
	}
	m, _ = m.Update(key(tui.KeyHome))
	if m.Cursor() != 0 {
		t.Fatalf("home: %d", m.Cursor())
	}
	m, _ = m.Update(key(tui.KeyEnd))
	if m.Cursor() != 4 {
		t.Fatalf("end: %d", m.Cursor())
	}
}

func TestEnterSelectsAndCloses(t *testing.T) {
	m := sample()
	m, _ = m.Update(key(tui.KeyDown))
	m, cmd := m.Update(key(tui.KeyEnter))
	sel, ok := msgOf(t, cmd).(SelectedMsg)
	if !ok || sel.Item.Value != "copy" || sel.Index != 1 {
		t.Fatalf("got %+v", sel)
	}
	if m.Open() {
		t.Fatal("still open")
	}
}

func TestEscCloses(t *testing.T) {
	m, cmd := sample().Update(key(tui.KeyEsc))
	if _, ok := msgOf(t, cmd).(ClosedMsg); !ok || m.Open() {
		t.Fatal("esc did not close")
	}
}

func TestAccelerator(t *testing.T) {
	m, cmd := sample().Update(runeKey('C')) // case-insensitive
	if sel := msgOf(t, cmd).(SelectedMsg); sel.Item.Value != "copy" || m.Open() {
		t.Fatalf("got %+v", sel)
	}
	// disabled item's accelerator does nothing
	m, cmd = sample().Update(runeKey('p'))
	if cmd != nil || !m.Open() {
		t.Fatal("disabled accelerator acted")
	}
}

func TestClosedIgnoresKeysExceptOpen(t *testing.T) {
	m := New(Item{Label: "a"})
	m2, cmd := m.Update(key(tui.KeyEnter))
	if cmd != nil || m2.Open() {
		t.Fatal("closed menu reacted")
	}
	m.AnchorX, m.AnchorY = 2, 2
	m, _ = m.Update(tui.Key{Type: tui.KeyF10, Mod: input.ModShift}) // shift+f10
	if !m.Open() {
		t.Fatal("shift+f10 did not open")
	}
}

func TestRebind(t *testing.T) {
	m := sample()
	m.KeyMap.Down.Keys = []string{"j"}
	m.KeyMap.Close.Keys = []string{"q"}
	m, _ = m.Update(key(tui.KeyDown))
	if m.Cursor() != 0 {
		t.Fatal("arrow moved after rebind")
	}
	m, _ = m.Update(runeKey('j'))
	if m.Cursor() != 1 {
		t.Fatal("j did not move")
	}
	if m2, cmd := m.Update(key(tui.KeyEsc)); cmd != nil || !m2.Open() {
		t.Fatal("esc still closes")
	}
	if _, cmd := m.Update(runeKey('q')); cmd == nil {
		t.Fatal("q did not close")
	}
}

func TestZeroKeyMapFallsBack(t *testing.T) {
	m := sample()
	m.KeyMap = KeyMap{}
	m, _ = m.Update(key(tui.KeyDown))
	if m.Cursor() != 1 {
		t.Fatal("zero KeyMap not defaulted")
	}
}

func TestBindings(t *testing.T) {
	m := sample()
	for _, mm := range []Model{m, New()} {
		bs := mm.Bindings()
		if len(bs) == 0 {
			t.Fatal("no bindings")
		}
		for _, b := range bs {
			if b.Desc == "" || len(b.Keys) == 0 {
				t.Errorf("incomplete %+v", b)
			}
		}
	}
	var _ []keymap.Binding = m.Bindings()
}

func TestMouse(t *testing.T) {
	m := New(Item{Label: "One", Value: "1"}, Item{Label: "Two", Value: "2"}, Item{Label: "Off", Disabled: true})
	m.ScreenW, m.ScreenH = 40, 20
	right := tui.MouseEvent{X: 10, Y: 5, Button: tui.MouseButtonRight, Action: tui.MouseActionPress}
	if m, _ = m.Update(right); m.Open() {
		t.Fatal("Mouse off opened")
	}
	m.Mouse = true
	m, _ = m.Update(right)
	if !m.Open() || m.Bounds().X != 10 || m.Bounds().Y != 5 {
		t.Fatalf("not opened at pointer: %+v", m.Bounds())
	}
	b := m.Bounds()
	// hover row 1 (border 1 + index 1)
	m, _ = m.Update(tui.MouseEvent{X: b.X + 2, Y: b.Y + 2, Action: tui.MouseActionMotion})
	if m.Cursor() != 1 {
		t.Fatalf("hover cursor %d", m.Cursor())
	}
	// click disabled row: nothing
	m2, cmd := m.Update(tui.MouseEvent{X: b.X + 2, Y: b.Y + 3, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if cmd != nil || !m2.Open() {
		t.Fatal("disabled row clicked")
	}
	// click item 0
	m3, cmd := m.Update(tui.MouseEvent{X: b.X + 2, Y: b.Y + 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if sel := msgOf(t, cmd).(SelectedMsg); sel.Item.Value != "1" || m3.Open() {
		t.Fatal("click did not select")
	}
	// click outside closes
	m4, cmd := m.Update(tui.MouseEvent{X: 0, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if _, ok := msgOf(t, cmd).(ClosedMsg); !ok || m4.Open() {
		t.Fatal("outside click did not close")
	}
	// wheel
	m5, _ := m.Update(tui.MouseEvent{X: b.X + 1, Y: b.Y + 1, Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m5.Cursor() != 0 {
		t.Fatalf("wheel up %d", m5.Cursor())
	}
}

func blank(w, h int) string {
	row := strings.Repeat(".", w)
	rows := make([]string, h)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

func TestRenderClipsToScreen(t *testing.T) {
	m := sample()
	base := blank(30, 12)
	for _, a := range [][2]int{{0, 0}, {28, 0}, {29, 11}, {5, 10}, {100, 100}} {
		m.Show(a[0], a[1])
		out := m.Render(base)
		lines := strings.Split(out, "\n")
		if len(lines) != 12 {
			t.Fatalf("anchor %v: %d rows", a, len(lines))
		}
		for i, l := range lines {
			if w := ansi.Width(l); w != 30 {
				t.Fatalf("anchor %v row %d width %d", a, i, w)
			}
		}
		if !strings.Contains(ansi.StripANSI(out), "Copy") {
			t.Fatalf("anchor %v: menu not on screen:\n%s", a, ansi.StripANSI(out))
		}
	}
	// flip above near the bottom
	m.Show(2, 11)
	if got := strings.Split(ansi.StripANSI(m.Render(base)), "\n"); !strings.Contains(strings.Join(got[:6], ""), "Cut") {
		t.Fatalf("not flipped:\n%s", strings.Join(got, "\n"))
	}
	// a tiny screen cuts, never panics
	small := blank(6, 3)
	m.Show(0, 0)
	if lines := strings.Split(m.Render(small), "\n"); len(lines) != 3 {
		t.Fatalf("small: %d rows", len(lines))
	}
}

func TestClosedRenderIsIdentity(t *testing.T) {
	if got := New(Item{Label: "a"}).Render("base"); got != "base" {
		t.Fatalf("got %q", got)
	}
}

func TestBoundsClippedToScreen(t *testing.T) {
	m := sample()
	m.ScreenW, m.ScreenH = 12, 4
	m.Show(0, 0)
	b := m.Bounds()
	if b.X+b.W > 12 || b.Y+b.H > 4 {
		t.Fatalf("bounds %+v exceed screen", b)
	}
	if (New().Bounds() != hittest.Rect{}) {
		t.Fatal("closed bounds not empty")
	}
}

func TestSanitised(t *testing.T) {
	m := New(Item{Label: "a\x1b[31mb"})
	m.Show(0, 0)
	if strings.Contains(m.View(), "\x1b[31m") {
		t.Fatal("escape drawn")
	}
	m.Raw = true
	if !strings.Contains(m.View(), "\x1b[31m") {
		t.Fatal("Raw changed label")
	}
}

func TestLinearize(t *testing.T) {
	var _ tui.Linearizer = Model{}
	if got := New(Item{Label: "a"}).Linearize(); got != "" {
		t.Fatalf("closed = %q", got)
	}
	want := "Context menu\n" +
		"Cut, item 1 of 4, shortcut ctrl+x, selected\n" +
		"Copy, item 2 of 4, shortcut ctrl+c\n" +
		"Paste, item 3 of 4, disabled\n" +
		"Delete, item 4 of 4"
	if got := sample().Linearize(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	e := New()
	e.Show(0, 0)
	if got := e.Linearize(); got != "Context menu\nNo items" {
		t.Fatalf("empty = %q", got)
	}
}

func TestLayoutNode(t *testing.T) {
	m := sample()
	sizes := []layout.Size{{W: 30, H: 8}, {W: 12, H: 3}, {W: 5, H: 1}, {W: 1, H: 1}, {W: 40, H: 20}}
	for _, s := range sizes {
		lines := strings.Split(m.LayoutNode().Render(s), "\n")
		if len(lines) != s.H {
			t.Fatalf("%v: %d rows", s, len(lines))
		}
		for _, l := range lines {
			if ansi.Width(l) != s.W {
				t.Fatalf("%v: row width %d", s, ansi.Width(l))
			}
		}
	}
	if _, ok := m.LayoutNode().(layout.CellNode); !ok {
		t.Fatal("open node is not a CellNode")
	}
	if got := New(Item{Label: "a"}).LayoutNode().Render(layout.Size{W: 4, H: 2}); strings.TrimSpace(ansi.StripANSI(got)) != "" {
		t.Fatalf("closed node not blank: %q", got)
	}
	if !strings.Contains(ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 30, H: 8})), "Delete") {
		t.Fatal("content missing")
	}
}

func TestSetTheme(t *testing.T) {
	th := theme.DraculaTheme()
	m := sample().SetTheme(th)
	if m.Theme.Primary != th.Primary {
		t.Fatal("theme not applied")
	}
	var _ tui.ThemeSetter[Model] = Model{}
}

func TestThemeChangesView(t *testing.T) {
	a := sample()
	b := sample().SetTheme(theme.DraculaTheme())
	if a.View() == b.View() {
		t.Fatal("theme had no effect on the view")
	}
}

func TestNoBorderTheme(t *testing.T) {
	m := sample()
	m.Theme.Border = theme.Border{}
	m.ScreenW, m.ScreenH = 40, 20
	b := m.Bounds()
	if b.H != len(m.Items) {
		t.Fatalf("height %d", b.H)
	}
	if m.rowAt(b.X, b.Y) != 0 {
		t.Fatal("first row not at top without a border")
	}
}
