package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func key(m model, t tui.KeyType) (model, tui.Cmd) { return update(m, tui.Key{Type: t}) }

func runes(m model, s string) (model, tui.Cmd) {
	return update(m, tui.Key{Type: tui.KeyRunes, Text: s})
}

func ready() model {
	m, _ := update(newModel(), tui.ResizeMsg{Width: 100, Height: 24})
	return m
}

func text(m model) string { return ansi.StripANSI(m.View()) }

// divider is the column of the rule between the sidebar and the screen.
func divider(m model) int {
	line := []rune(strings.Split(text(m), "\n")[2])
	for i, r := range line {
		if r == '│' {
			return i
		}
	}
	return -1
}

func crumbs(m model) string {
	var out []string
	for _, r := range m.stack {
		out = append(out, m.crumb(r))
	}
	return strings.Join(out, " › ")
}

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func click(t testing.TB, m model, id string, clicks int) (model, tui.Cmd) {
	r := rect(t, m, id)
	return update(m, tui.MouseEvent{X: r.X + min(1, r.W-1), Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: clicks})
}

// open walks Home > Customers > Ada > her first order with the keys.
func openOrder(m model) model {
	m, _ = key(m, tui.KeyEnter) // Customers, the first shortcut on Home
	m, _ = key(m, tui.KeyEnter) // Ada
	m, _ = key(m, tui.KeyEnter) // her first order
	return m
}

func TestHomeShowsSidebarCrumbsAndShortcuts(t *testing.T) {
	out := text(ready())
	for _, want := range []string{"‹ Back", "Home", "Welcome back", "12 customers, 36 orders, 2 shipped", "☺ Customers", "▤ Orders", "⚙ Settings", "│", "[tab] sidebar"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestOpeningPushesAndBackPopsWithTheSelectionKept(t *testing.T) {
	m := openOrder(ready())
	if crumbs(m) != "Home › Customers › Ada Lovelace › #1001" || len(m.stack) != 4 {
		t.Fatalf("%s", crumbs(m))
	}
	out := text(m)
	for _, want := range []string{"Order #1001", "Keyboard", "Customer  Ada Lovelace", "Depth 4", "Esc goes back to Ada Lovelace"} {
		if !strings.Contains(out, want) {
			t.Errorf("order screen lacks %q:\n%s", want, out)
		}
	}
	m, _ = key(m, tui.KeyEsc)
	m, _ = key(m, tui.KeyEsc)
	if crumbs(m) != "Home › Customers" {
		t.Fatalf("%s", crumbs(m))
	}
	m, _ = key(m, tui.KeyDown)
	m, _ = key(m, tui.KeyDown)
	m, _ = key(m, tui.KeyEnter)
	m, _ = key(m, tui.KeyBackspace)
	if m.top().cur != 2 || !strings.Contains(text(m), "▸ ") {
		t.Fatalf("the customer list lost its place: cur %d", m.top().cur)
	}
}

func TestSamePlaceReachedTwoWaysHasTwoTrails(t *testing.T) {
	m := openOrder(ready())
	a := crumbs(m)
	m, _ = click(t, ready(), "side-2", 1)
	m, _ = key(m, tui.KeyEnter)
	if b := crumbs(m); b != "Home › Orders › #1001" || b == a {
		t.Fatalf("trail %q vs %q", b, a)
	}
}

func TestBackAtTheTopSaysSoAndExpires(t *testing.T) {
	m, cmd := key(ready(), tui.KeyEsc)
	if len(m.toasts) != 1 || cmd == nil || !strings.Contains(text(m), "Already at the top") {
		t.Fatalf("toasts %v", m.toasts)
	}
	for i := 0; i < toastTicks-1; i++ {
		m, cmd = update(m, tickMsg{})
		if cmd == nil || len(m.toasts) != 1 {
			t.Fatalf("tick %d: toasts %d", i, len(m.toasts))
		}
	}
	m, cmd = update(m, tickMsg{})
	if len(m.toasts) != 0 || cmd != nil || m.ticking || strings.Contains(text(m), "Already at the top") {
		t.Fatal("the notice should be gone and the clock stopped")
	}
}

func TestClockStartsOnceForSeveralNotices(t *testing.T) {
	m, c1 := key(ready(), tui.KeyEsc)
	m, c2 := key(m, tui.KeyEsc)
	if c1 == nil || c2 != nil || len(m.toasts) != 2 {
		t.Fatalf("c1 %v c2 %v toasts %d", c1, c2, len(m.toasts))
	}
	for range 5 {
		m, _ = key(m, tui.KeyEsc)
	}
	if len(m.toasts) != 3 {
		t.Fatalf("toasts are capped at three, got %d", len(m.toasts))
	}
}

func TestClickingANoticeDismissesIt(t *testing.T) {
	m, _ := key(ready(), tui.KeyEsc)
	m, _ = click(t, m, "toast-0", 1)
	if len(m.toasts) != 0 {
		t.Fatal("notice still there")
	}
}

func TestStarAndShipChangeStateAndNotify(t *testing.T) {
	m, _ := runes(ready(), "s")
	if !strings.Contains(m.toasts[0].text, "Open a customer") {
		t.Fatalf("%v", m.toasts)
	}
	m = openOrder(ready())
	m, _ = key(m, tui.KeyEsc)
	m, _ = runes(m, "s")
	if !m.starred[1] || !strings.Contains(text(m), "Starred Ada Lovelace") || !strings.Contains(text(m), "★ starred") {
		t.Fatalf("\n%s", text(m))
	}
	m, _ = runes(m, "s")
	if m.starred[1] {
		t.Fatal("s did not unstar")
	}
	m, _ = key(m, tui.KeyDown)
	m, _ = key(m, tui.KeyEnter) // order 1002 is already shipped
	m, _ = runes(m, "m")
	if !strings.Contains(m.toasts[len(m.toasts)-1].text, "already shipped") {
		t.Fatalf("%v", m.toasts)
	}
	m, _ = key(m, tui.KeyEsc)
	m, _ = key(m, tui.KeyDown)
	m, _ = key(m, tui.KeyEnter) // 1003, new
	m, _ = runes(m, "m")
	if !m.shipped[1003] || !strings.Contains(text(m), "Order #1003 shipped") || !strings.Contains(text(m), "Status    shipped") {
		t.Fatalf("\n%s", text(m))
	}
	if _, cmd := update(ready(), tui.Key{Type: tui.KeyRunes, Text: "m"}); cmd == nil {
		t.Fatal("m on Home should warn")
	}
}

func TestSidebarByKeyAndMouse(t *testing.T) {
	m, _ := key(ready(), tui.KeyTab)
	if !m.side {
		t.Fatal("Tab did not move to the sidebar")
	}
	m, _ = key(m, tui.KeyDown)
	m, _ = key(m, tui.KeyDown)
	m, _ = key(m, tui.KeyEnter)
	if crumbs(m) != "Home › Orders" || m.side {
		t.Fatalf("%s side %v", crumbs(m), m.side)
	}
	m, _ = click(t, m, "side-3", 1)
	if crumbs(m) != "Home › Settings" || !strings.Contains(text(m), "[ ] Compact sidebar") {
		t.Fatalf("%s", crumbs(m))
	}
	m, _ = click(t, m, "side-0", 1)
	if len(m.stack) != 1 {
		t.Fatal("Home should clear the stack")
	}
}

func TestRowsClickSelectsThenOpensAndDoubleClickOpens(t *testing.T) {
	m, _ := click(t, ready(), "row-1", 1) // Orders shortcut on Home: first click selects
	if len(m.stack) != 1 || m.top().cur != 1 {
		t.Fatalf("first click: depth %d cur %d", len(m.stack), m.top().cur)
	}
	m, _ = click(t, m, "row-1", 1)
	if crumbs(m) != "Home › Orders" {
		t.Fatalf("second click: %s", crumbs(m))
	}
	m, _ = click(t, m, "row-4", 2)
	if crumbs(m) != "Home › Orders › #1005" {
		t.Fatalf("double click: %s", crumbs(m))
	}
}

func TestBreadcrumbAndBackButtonNavigate(t *testing.T) {
	m := openOrder(ready())
	m, _ = click(t, m, "crumb-1", 1)
	if crumbs(m) != "Home › Customers" {
		t.Fatalf("%s", crumbs(m))
	}
	m = openOrder(ready())
	m, _ = click(t, m, "back", 1)
	if crumbs(m) != "Home › Customers › Ada Lovelace" {
		t.Fatalf("%s", crumbs(m))
	}
	m, _ = click(t, m, "crumb-2", 1)
	if len(m.stack) != 3 {
		t.Fatal("clicking the current crumb should do nothing")
	}
	m, _ = click(t, m, "crumb-0", 1)
	if len(m.stack) != 1 {
		t.Fatal("Home crumb")
	}
}

func TestSettingsToggleTheSidebarAndNotices(t *testing.T) {
	m, _ := click(t, ready(), "side-3", 1)
	m, _ = key(m, tui.KeyEnter)
	if !m.compact || m.sideW() != 5 || divider(m) != 5 {
		t.Fatalf("sidebar should be icons only:\n%s", text(m))
	}
	m, _ = key(m, tui.KeyDown)
	m, _ = key(m, tui.KeyEnter)
	if m.notify {
		t.Fatal("notices still on")
	}
	m, cmd := key(m, tui.KeyEsc)
	m, _ = key(m, tui.KeyEsc)
	if len(m.toasts) != 0 || cmd != nil {
		t.Fatal("a notice appeared with notices off")
	}
}

func TestNarrowWindowsUseTheCompactSidebar(t *testing.T) {
	m, _ := update(ready(), tui.ResizeMsg{Width: 50, Height: 14})
	if m.sideW() != 5 || divider(m) != 5 {
		t.Fatalf("\n%s", text(m))
	}
}

func TestLongListsScrollToTheSelection(t *testing.T) {
	m, _ := click(t, ready(), "side-2", 1)
	m, _ = update(m, tui.ResizeMsg{Width: 100, Height: 12})
	for range 20 {
		m, _ = key(m, tui.KeyDown)
	}
	if !strings.Contains(text(m), "#1021") {
		t.Fatalf("selection not in view:\n%s", text(m))
	}
	m, _ = update(m, tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.top().cur != 19 {
		t.Fatalf("wheel: cur %d", m.top().cur)
	}
}

func TestHelpAndQuit(t *testing.T) {
	m, _ := runes(ready(), "?")
	if !strings.Contains(text(m), "jump back to it") {
		t.Fatalf("no help:\n%s", text(m))
	}
	if m, _ = key(m, tui.KeyEsc); m.help || len(m.stack) != 1 {
		t.Fatal("Esc should close help and not go back")
	}
	if _, c := key(ready(), tui.KeyCtrlC); c == nil {
		t.Fatal("Ctrl+C did not quit")
	}
	if _, c := runes(ready(), "q"); c == nil {
		t.Fatal("q did not quit")
	}
}

// An open customer shows its avatar beside the name, at every size, never wider
// than the terminal; it is the same picture each time and smiles once starred.
func TestCustomerShowsItsAvatar(t *testing.T) {
	for _, size := range [][2]int{{40, 10}, {80, 24}, {120, 40}, {300, 80}} {
		m, _ := update(newModel(), tui.ResizeMsg{Width: size[0], Height: size[1]})
		m = openOrder(m)
		m, _ = key(m, tui.KeyEsc)
		if m.top().kind != rCustomer {
			t.Fatalf("%dx%d: the customer did not open (%v)", size[0], size[1], crumbs(m))
		}
		view := text(m)
		for _, line := range strings.Split(view, "\n") {
			if ansi.Width(line) > size[0] {
				t.Errorf("%dx%d: a line is %d wide", size[0], size[1], ansi.Width(line))
			}
		}
		face := m.face(m.top(), m.theme())
		if size[0] >= 40 && size[1] >= 14 && !strings.Contains(view, ansi.StripANSI(strings.Split(face, "\n")[1])) {
			t.Errorf("%dx%d: the avatar is not on screen:\n%s", size[0], size[1], view)
		}
		if face != m.face(m.top(), m.theme()) {
			t.Error("the same customer drew two avatars")
		}
		starred, _ := runes(m, "s")
		if starred.face(starred.top(), starred.theme()) == face {
			t.Errorf("%dx%d: a starred customer looks the same", size[0], size[1])
		}
	}
}

// Moving over the customer's avatar wakes it and a click makes it react; moving
// away lets it rest.
func TestAvatarFollowsTheMouse(t *testing.T) {
	m := openOrder(ready())
	m, _ = key(m, tui.KeyEsc)
	a := m.avatarOf(m.top())
	in := a.Bounds()
	move := func(x, y int) tui.MouseEvent { return tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionMotion} }

	m, _ = update(m, move(in.X+2, in.Y+1))
	if !m.faceModel.Idling() {
		t.Fatal("the pointer over the avatar did not wake it")
	}
	m, cmd := update(m, tui.MouseEvent{X: in.X + 2, Y: in.Y + 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if !m.faceModel.Reacting() || cmd == nil {
		t.Fatal("a click on the avatar did not make it react")
	}
	m, _ = update(m, move(0, 0))
	if m.faceModel.Idling() {
		t.Error("moving away left the avatar awake")
	}
}

// On a terminal that draws images the avatar is an image; elsewhere it is cells.
func TestAvatarIsAnImageWhereTheTerminalDrawsImages(t *testing.T) {
	m := openOrder(ready())
	m, _ = key(m, tui.KeyEsc)
	if strings.Contains(m.View(), "1337;File") {
		t.Error("an image was sent before the terminal said it draws them")
	}
	m, _ = update(m, tui.CapabilitiesMsg{Capabilities: tui.Capabilities{InlineImages: true}})
	if !strings.Contains(m.View(), "1337;File") {
		t.Errorf("no inline image on iTerm:\n%s", text(m))
	}
	m, _ = update(m, tui.CapabilitiesMsg{Capabilities: tui.Capabilities{KittyGraphics: true}})
	if !strings.Contains(m.View(), "\x1b_G") {
		t.Error("no kitty image on kitty")
	}
}
