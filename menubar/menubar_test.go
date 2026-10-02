package menubar

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }
func alt(r rune) tui.Key {
	return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r}), Mod: input.ModAlt}
}
func runeKey(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func sample() Model {
	return New(
		Menu{Title: "File", Accel: 'f', Items: []Item{
			{Label: "New", Value: "new", Accel: 'n'},
			{Label: "Open", Value: "open"},
			{Separator: true},
			{Label: "Quit", Value: "quit", Shortcut: "ctrl+q"},
		}},
		Menu{Title: "Edit", Accel: 'e', Items: []Item{
			{Label: "Undo", Value: "undo", Disabled: true},
			{Label: "Redo", Value: "redo"},
		}},
		Menu{Title: "Help", Items: []Item{{Label: "About", Value: "about"}}},
	)
}

func send(m Model, msgs ...tui.Msg) (Model, tui.Msg) {
	var out tui.Msg
	for _, msg := range msgs {
		var cmd tui.Cmd
		m, cmd = m.Update(msg)
		if cmd != nil {
			out = cmd()
		}
	}
	return m, out
}

func TestFocusOpensAndNavigates(t *testing.T) {
	m := sample()
	m, _ = send(m, key(tui.KeyF10))
	if !m.Open() || m.Active() != 0 {
		t.Fatal("f10 did not open first menu")
	}
	m, _ = send(m, key(tui.KeyRight))
	if m.Active() != 1 {
		t.Fatalf("right: %d", m.Active())
	}
	m, _ = send(m, key(tui.KeyRight), key(tui.KeyRight)) // wraps
	if m.Active() != 0 {
		t.Fatalf("wrap right: %d", m.Active())
	}
	m, _ = send(m, key(tui.KeyLeft))
	if m.Active() != 2 {
		t.Fatalf("wrap left: %d", m.Active())
	}
}

func TestEnterActivates(t *testing.T) {
	m := sample()
	m, msg := send(m, key(tui.KeyF10), key(tui.KeyDown), key(tui.KeyDown), key(tui.KeyEnter))
	sel, ok := msg.(SelectedMsg)
	if !ok || sel.Menu != 0 || sel.Item.Value != "quit" || sel.Index != 3 {
		t.Fatalf("got %+v", msg)
	}
	if m.Open() {
		t.Fatal("still open")
	}
}

func TestEnterOnSwitchedMenu(t *testing.T) {
	_, msg := send(sample(), key(tui.KeyF10), key(tui.KeyRight), key(tui.KeyEnter)) // Undo disabled -> Redo first selectable
	if sel := msg.(SelectedMsg); sel.Menu != 1 || sel.Item.Value != "redo" {
		t.Fatalf("got %+v", sel)
	}
}

func TestEscCloses(t *testing.T) {
	m, msg := send(sample(), key(tui.KeyF10), key(tui.KeyEsc))
	if _, ok := msg.(ClosedMsg); !ok || m.Open() {
		t.Fatal("esc did not close")
	}
	m, msg = send(m, key(tui.KeyEsc)) // idle: no-op
	if msg != nil || m.Open() {
		t.Fatal("idle esc reacted")
	}
}

func TestAltAccelerator(t *testing.T) {
	m, _ := send(sample(), alt('E'))
	if !m.Open() || m.Active() != 1 {
		t.Fatal("alt+e did not open Edit")
	}
	m, _ = send(m, alt('f')) // switches while open
	if m.Active() != 0 {
		t.Fatal("alt+f did not switch")
	}
	if m2, _ := send(sample(), runeKey('f')); m2.Open() {
		t.Fatal("plain f opened a menu")
	}
	// item accelerator inside the dropdown
	_, msg := send(m, runeKey('n'))
	if sel := msg.(SelectedMsg); sel.Item.Value != "new" {
		t.Fatalf("got %+v", sel)
	}
}

func TestRebind(t *testing.T) {
	m := sample()
	m.KeyMap.Right.Keys = []string{"l"}
	m.KeyMap.Focus.Keys = []string{"f2"}
	m, _ = send(m, key(tui.KeyF10))
	if m.Open() {
		t.Fatal("f10 still focuses")
	}
	m, _ = send(m, key(tui.KeyF2), key(tui.KeyRight))
	if m.Active() != 0 {
		t.Fatal("arrow moved after rebind")
	}
	m, _ = send(m, runeKey('l'))
	if m.Active() != 1 {
		t.Fatal("l did not move")
	}
}

func TestZeroKeyMapAndBindings(t *testing.T) {
	m := sample()
	m.KeyMap = KeyMap{}
	if m, _ = send(m, key(tui.KeyF10)); !m.Open() {
		t.Fatal("zero KeyMap not defaulted")
	}
	for _, mm := range []Model{sample(), m} {
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
}

func TestEmptyBar(t *testing.T) {
	m, cmd := New().Update(key(tui.KeyF10))
	if m.Open() || cmd != nil {
		t.Fatal("empty bar reacted")
	}
	if New().Linearize() != "No menus" {
		t.Fatal("linearize empty")
	}
	m.Show(3) // no-op
	if m.Open() {
		t.Fatal("Show on empty opened")
	}
}

func press(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
}

func TestMouse(t *testing.T) {
	m := sample()
	m.Bounds.X, m.Bounds.Y, m.Bounds.W = 2, 1, 40
	m.ScreenW, m.ScreenH = 60, 20
	// " File " = x 2..7, " Edit " = x 8..13
	if m, _ = send(m, press(3, 1)); m.Open() {
		t.Fatal("Mouse off opened")
	}
	m.Mouse = true
	m, _ = send(m, press(3, 1))
	if !m.Open() || m.Active() != 0 {
		t.Fatal("click did not open File")
	}
	// hover Edit switches
	m, _ = send(m, tui.MouseEvent{X: 9, Y: 1, Action: tui.MouseActionMotion})
	if m.Active() != 1 {
		t.Fatal("hover did not switch")
	}
	// dropdown of Edit is at x=8, y=2; border row y=2, rows from y=3: Undo(disabled), Redo y=4
	m2, msg := send(m, press(10, 4))
	if sel, ok := msg.(SelectedMsg); !ok || sel.Item.Value != "redo" || sel.Menu != 1 || m2.Open() {
		t.Fatalf("item click: %+v", msg)
	}
	// clicking the open title closes
	m3, msg := send(m, press(9, 1))
	if _, ok := msg.(ClosedMsg); !ok || m3.Open() {
		t.Fatal("title click did not close")
	}
	// click outside closes
	m4, msg := send(m, press(50, 15))
	if _, ok := msg.(ClosedMsg); !ok || m4.Open() {
		t.Fatal("outside click did not close")
	}
	// click on row beyond titles / other row: idle ignores
	if m5, _ := send(sample(), press(30, 1)); m5.Open() {
		t.Fatal("idle")
	}
}

func TestRenderOverlay(t *testing.T) {
	m := sample()
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat(".", 30)+"\n", 8), "\n")
	if m.Render(base) != base {
		t.Fatal("idle Render changed base")
	}
	m, _ = send(m, alt('e'))
	out := strings.Split(ansi.StripANSI(m.Render(base)), "\n")
	if !strings.Contains(out[1], "┌") && !strings.Contains(out[1], "╭") {
		t.Fatalf("dropdown not under bar: %q", out)
	}
	if !strings.Contains(strings.Join(out, ""), "Redo") {
		t.Fatal("items missing")
	}
	for _, l := range out {
		if ansi.Width(l) != 30 {
			t.Fatalf("width %d", ansi.Width(l))
		}
	}
}

func TestViewMarksActiveWithoutColour(t *testing.T) {
	m := sample()
	if v := ansi.StripANSI(m.View()); v != " File  Edit  Help " {
		t.Fatalf("idle %q", v)
	}
	m, _ = send(m, key(tui.KeyF10))
	if v := ansi.StripANSI(m.View()); v != "[File] Edit  Help " {
		t.Fatalf("open %q", v)
	}
}

func TestLinearize(t *testing.T) {
	var _ tui.Linearizer = Model{}
	m := sample()
	want := "Menu bar\nFile, menu 1 of 3\nEdit, menu 2 of 3\nHelp, menu 3 of 3"
	if got := m.Linearize(); got != want {
		t.Fatalf("idle:\n%s", got)
	}
	m, _ = send(m, key(tui.KeyF10), key(tui.KeyRight))
	want = "Menu bar\nFile, menu 1 of 3\nEdit, menu 2 of 3, expanded\nHelp, menu 3 of 3\n" +
		"Edit menu\nUndo, item 1 of 2, disabled\nRedo, item 2 of 2, selected"
	if got := m.Linearize(); got != want {
		t.Fatalf("open:\n%s\nwant\n%s", got, want)
	}
}

func TestLayoutNode(t *testing.T) {
	m := sample()
	sizes := []layout.Size{{W: 30, H: 8}, {W: 12, H: 3}, {W: 5, H: 1}, {W: 1, H: 1}, {W: 40, H: 20}}
	check := func(m Model) {
		for _, s := range sizes {
			lines := strings.Split(m.LayoutNode().Render(s), "\n")
			if len(lines) != s.H {
				t.Fatalf("%v: %d rows", s, len(lines))
			}
			for _, l := range lines {
				if ansi.Width(l) != s.W {
					t.Fatalf("%v: width %d", s, ansi.Width(l))
				}
			}
		}
	}
	check(m)
	if _, ok := m.LayoutNode().(layout.CellNode); !ok {
		t.Fatal("not a CellNode")
	}
	m, _ = send(m, key(tui.KeyF10))
	check(m)
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 30, H: 8}))
	if !strings.Contains(out, "Quit") || !strings.HasPrefix(out, "[File]") {
		t.Fatalf("open node:\n%s", out)
	}
	if got := m.LayoutNode().Measure(layout.Loose(layout.Size{W: 100, H: 100})); got.H < 5 {
		t.Fatalf("measure %v", got)
	}
}

func TestSetTheme(t *testing.T) {
	var _ tui.ThemeSetter[Model] = Model{}
	th := theme.DraculaTheme()
	m, _ := send(sample(), key(tui.KeyF10))
	m = m.SetTheme(th)
	if m.Theme.Primary != th.Primary || m.drop.Theme.Primary != th.Primary {
		t.Fatal("theme not applied to bar and dropdown")
	}
	// a menu opened later gets it too
	m, _ = send(m, key(tui.KeyRight))
	if m.drop.Theme.Primary != th.Primary {
		t.Fatal("later dropdown not themed")
	}
	if sample().View() == func() string { x, _ := send(sample(), key(tui.KeyF10)); return x.SetTheme(th).View() }() {
		t.Fatal("no visual change")
	}
}

func TestSanitisedTitle(t *testing.T) {
	m := New(Menu{Title: "a\x1b[31mb"})
	if strings.Contains(m.View(), "\x1b[31m") {
		t.Fatal("escape in title")
	}
}
