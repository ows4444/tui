package main

import (
	"context"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/avatar"
)

func send(m model, msgs ...tui.Msg) (model, tui.Cmd) {
	var cmd tui.Cmd
	for _, msg := range msgs {
		var next tui.Model
		next, cmd = m.Update(msg)
		m = next.(model)
	}
	return m, cmd
}

func key(s string) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: s} }

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func status(m model) string {
	lines := strings.Split(ansi.StripANSI(m.View()), "\n")
	return lines[len(lines)-2]
}

func TestArrowsAndLettersMoveTheSelection(t *testing.T) {
	m := initialModel()
	if m.Init() != nil {
		t.Error("Init returned a Cmd")
	}
	m, _ = send(m, tui.Key{Type: tui.KeyRight}, key("l"))
	if !strings.HasPrefix(status(m), names[2]+":") {
		t.Errorf("after two steps right: %s", status(m))
	}
	m, _ = send(m, tui.Key{Type: tui.KeyLeft}, key("h"), key("h"))
	if !strings.HasPrefix(status(m), names[len(names)-1]+":") {
		t.Errorf("stepping left of the first name should wrap: %s", status(m))
	}
}

func TestSizeKeysStepAndStopAtTheEnds(t *testing.T) {
	m := initialModel()
	for range len(sizes) + 2 {
		m, _ = send(m, key("+"))
	}
	if !strings.Contains(status(m), "[24x12,") {
		t.Errorf("largest size: %s", status(m))
	}
	m, _ = send(m, key("-"), tui.Key{Type: tui.KeyDown})
	if !strings.Contains(status(m), "[12x6,") {
		t.Errorf("two steps down: %s", status(m))
	}
	for range len(sizes) + 2 {
		m, _ = send(m, key("-"))
	}
	m, _ = send(m, tui.Key{Type: tui.KeyUp}, key("="))
	if !strings.Contains(status(m), "[8x4,") {
		t.Errorf("two steps up from the smallest: %s", status(m))
	}
}

func TestBackgroundAndGlyphKeys(t *testing.T) {
	m := initialModel()
	for _, want := range []string{"squircle", "circle", "square", "none"} {
		m, _ = send(m, key("b"))
		if !strings.Contains(status(m), ", "+want+", blocks, none, own]") {
			t.Errorf("after b: %s, want %s", status(m), want)
		}
	}
	m, _ = send(m, key("a"))
	if !strings.Contains(status(m), ", ascii, none, own]") {
		t.Errorf("after a: %s", status(m))
	}
	// The help line names keys with arrows; the wall above it must be ASCII.
	lines := strings.Split(ansi.StripANSI(m.View()), "\n")
	for _, l := range lines[:len(lines)-1] {
		for _, r := range l {
			if r > 127 {
				t.Fatalf("non-ASCII rune %q in the ASCII wall: %s", r, l)
			}
		}
	}
	m, _ = send(m, key("a"))
	if !strings.Contains(status(m), ", blocks, none, own]") {
		t.Errorf("a did not toggle back: %s", status(m))
	}
}

func TestQuitKeys(t *testing.T) {
	for name, k := range map[string]tui.Key{"q": key("q"), "esc": {Type: tui.KeyEsc}, "ctrl+c": {Type: tui.KeyCtrlC}} {
		if _, cmd := send(initialModel(), k); !isQuit(cmd) {
			t.Errorf("%s did not quit", name)
		}
	}
	if _, cmd := send(initialModel(), key("x")); cmd == nil || isQuit(cmd) {
		t.Error("an unbound key should blink, not quit")
	}
}

// The wall shows the page that holds the selection, and marks only that name.
func TestWallPagesWithTheSelection(t *testing.T) {
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 40, Height: 10})
	cols, rows, _ := m.grid()
	if cols != 3 || rows != 1 {
		t.Fatalf("40x10 grid is %dx%d", cols, rows)
	}
	view := ansi.StripANSI(m.View())
	if !strings.Contains(view, names[0]) || strings.Contains(view, names[3]) {
		t.Errorf("first page should hold the first three names:\n%s", view)
	}
	m, _ = send(m, key("l"), key("l"), key("l"))
	view = ansi.StripANSI(m.View())
	if strings.Contains(view, " "+names[0]+" ") || !strings.Contains(view, names[3]) {
		t.Errorf("after three steps the second page should show:\n%s", view)
	}
	if n := strings.Count(m.View(), selectedStyle.Render(names[3])); n != 1 {
		t.Errorf("the selected name is marked %d times", n)
	}
	// The last page is short; the wall stops at the last name.
	m.selected = len(names) - 1
	if view = ansi.StripANSI(m.View()); !strings.Contains(view, names[len(names)-1]) {
		t.Errorf("last page:\n%s", view)
	}
}

// A terminal too small for one tile still draws one, clipped to its width.
func TestTinyTerminalStillDrawsOneTile(t *testing.T) {
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 5, Height: 3})
	if cols, rows, _ := m.grid(); cols != 1 || rows != 1 {
		t.Fatalf("grid is %dx%d", cols, rows)
	}
	for _, l := range strings.Split(m.View(), "\n") {
		if w := ansi.Width(l); w > 5 {
			t.Errorf("line is %d wide: %q", w, l)
		}
	}
}

// Every key press blinks the selected avatar: the Cmd it returns is the
// blink's tick, and feeding the ticks back plays it to rest.
func TestAKeyPressBlinksTheSelectedAvatar(t *testing.T) {
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 80, Height: 24})
	rest := m.View()
	m, cmd := send(m, key(" "))
	if cmd == nil || !m.wall[0].Blinking() {
		t.Fatal("space did not start a blink")
	}
	if m.View() == rest {
		t.Error("the wall did not change when the blink started")
	}
	for steps := 0; cmd != nil; steps++ {
		if steps > 10 {
			t.Fatal("the blink does not end")
		}
		m, cmd = send(m, tui.RunCmd(context.Background(), cmd))
	}
	if m.wall[0].Blinking() || m.View() != rest {
		t.Error("the wall did not return to rest after the blink")
	}
	// Moving the selection blinks the newly selected avatar.
	m, cmd = send(m, key("l"))
	if cmd == nil || !m.wall[1].Blinking() || m.wall[0].Blinking() {
		t.Error("after moving, the newly selected avatar is not the one blinking")
	}
}

// The avatars look at the pointer: moving it from one side of the wall to
// the other redraws them, and before it moves they are at rest.
func TestAvatarsLookAtThePointer(t *testing.T) {
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 80, Height: 24})
	rest := m.View()
	left, cmd := send(m, tui.MouseEvent{X: 0, Y: 4, Action: tui.MouseActionMotion})
	if cmd != nil {
		t.Error("a mouse move returned a Cmd")
	}
	right, _ := send(m, tui.MouseEvent{X: 79, Y: 4, Action: tui.MouseActionMotion})
	if left.View() == rest || right.View() == rest || left.View() == right.View() {
		t.Error("the wall does not follow the pointer")
	}
	for _, v := range []model{left, right} {
		for _, l := range strings.Split(v.View(), "\n") {
			if w := ansi.Width(l); w > 80 {
				t.Errorf("a line is %d wide while looking", w)
			}
		}
	}
}

// e steps through every expression and back to none, and each one redraws
// the wall.
func TestExpressionKeyCyclesThePoses(t *testing.T) {
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 120, Height: 40}, key("+"))
	seen := map[string]bool{}
	for _, want := range []string{"happy", "sad", "mad", "surprised", "wink", "sleepy", "thinking",
		"smug", "unsure", "scared", "love", "shy", "sick", "none"} {
		m, _ = send(m, key("e"))
		if !strings.HasSuffix(status(m), ", "+want+", own]") {
			t.Errorf("after e: %s, want %s", status(m), want)
		}
		wall := m.View()
		if seen[wall] {
			t.Errorf("%s draws a wall already seen", want)
		}
		seen[wall] = true
	}
}

// i starts every avatar idling and a second i stops them all; the Cmd it
// returns carries a tick for each.
func TestIdleKeyTogglesTheWholeWall(t *testing.T) {
	m, cmd := send(initialModel(), key("i"))
	if cmd == nil {
		t.Fatal("i returned no Cmd")
	}
	for i, a := range m.wall {
		if !a.Idling() {
			t.Fatalf("avatar %d is not idling", i)
		}
	}
	m, cmd = send(m, key("i"))
	if cmd != nil {
		t.Error("stopping returned a Cmd")
	}
	for i, a := range m.wall {
		if a.Idling() {
			t.Fatalf("avatar %d is still idling", i)
		}
	}
}

// c, t and s pin the hue, the tone and the silhouette for the whole wall and
// step back to each name's own.
func TestOverrideKeysPinTheWall(t *testing.T) {
	m := initialModel()
	own := m.View()
	for _, k := range []string{"c", "t", "s"} {
		n, _ := send(m, key(k))
		if n.View() == own {
			t.Errorf("%s did not change the wall", k)
		}
	}
	m, _ = send(m, key("s"))
	if !strings.HasPrefix(status(m), names[0]+": round,") {
		t.Errorf("after s the selected avatar is not round: %s", status(m))
	}
	for range avatar.SilhouetteTriangle {
		m, _ = send(m, key("s"))
	}
	for range len(hues) {
		m, _ = send(m, key("c"))
	}
	for range avatar.ToneInk + 1 {
		m, _ = send(m, key("t"))
	}
	// Each key press also blinks the selected avatar, so compare what is
	// pinned, not the frame.
	if m.hue != 0 || m.tone != avatar.ToneAuto || m.silhouette != avatar.SilhouetteAuto {
		t.Errorf("a full round of each key left hue %d, tone %v, silhouette %v", m.hue, m.tone, m.silhouette)
	}
}

func click(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
}

// A click on an avatar selects it and makes it pull a face; the Cmd releases
// it. A click between tiles, or outside the wall, does nothing.
func TestClickingAnAvatarMakesItReact(t *testing.T) {
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 80, Height: 24})
	_, _, tileW := m.grid()
	// The second tile of the second row: name index cols+1.
	cols, _, _ := m.grid()
	x, y := (tileW+tileGap)+tileW/2, wallTop+(sizes[m.size][1]+2)+1
	n, cmd := send(m, click(x, y))
	want := cols + 1
	if cmd == nil || n.selected != want || !n.wall[want].Reacting() {
		t.Fatalf("a click at (%d, %d) selected %d (reacting %v), want %d", x, y, n.selected, n.wall[want].Reacting(), want)
	}
	for i, a := range n.wall {
		if i != want && a.Reacting() {
			t.Errorf("avatar %d reacted to a click on %d", i, want)
		}
	}
	if !strings.HasPrefix(status(n), names[want]+":") {
		t.Errorf("the clicked avatar is not the selected one: %s", status(n))
	}
	for where, at := range map[string][2]int{
		"the gap between tiles":  {tileW, wallTop + 1},
		"the title":              {3, 0},
		"the row between tiles":  {3, wallTop + sizes[m.size][1] + 1},
		"right of the last tile": {79, wallTop + 1},
		"below the wall":         {3, 23},
	} {
		if o, cmd := send(m, click(at[0], at[1])); cmd != nil || o.selected != m.selected {
			t.Errorf("a click on %s did something", where)
		}
	}
	// Other buttons and a release do not react.
	if _, cmd := send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonRight, Action: tui.MouseActionPress}); cmd != nil {
		t.Error("a right click reacted")
	}
	if _, cmd := send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionMotion}); cmd != nil {
		t.Error("a drag reacted")
	}
	// On the last page a click past the last name does nothing.
	last, _ := send(m, tui.ResizeMsg{Width: 40, Height: 10})
	last.selected = len(names) - 1
	c, _, w := last.grid()
	if o, cmd := send(last, click((c-1)*(w+tileGap)+1, wallTop+1)); cmd != nil || o.selected != len(names)-1 {
		t.Error("a click on an empty tile of the last page did something")
	}
}

// r makes the selected avatar react.
func TestReactKey(t *testing.T) {
	m, cmd := send(initialModel(), key("r"))
	if cmd == nil || !m.wall[0].Reacting() {
		t.Error("r did not make the selected avatar react")
	}
}

// p steps through the trait presets, each redrawing the wall, and back to
// each name's own avatar.
func TestPinsKeyStepsThroughPresets(t *testing.T) {
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 120, Height: 40}, key("+"), key("+"))
	seen := map[string]bool{}
	for _, want := range []string{"big eyes", "wide-set", "square", "own"} {
		m, _ = send(m, key("p"))
		if !strings.HasSuffix(status(m), ", "+want+"]") {
			t.Errorf("after p: %s, want %s", status(m), want)
		}
		// Compare the unselected part of the wall: the selected avatar
		// blinks at every key.
		wall := strings.Join(strings.Split(m.View(), "\n")[wallTop+sizes[m.size][1]+2:], "\n")
		if seen[wall] {
			t.Errorf("%s draws a wall already seen", want)
		}
		seen[wall] = true
	}
}
