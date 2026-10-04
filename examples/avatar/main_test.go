package main

import (
	"context"
	"math"
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
		if !strings.Contains(status(m), ", "+want+", blocks, none, own, still]") {
			t.Errorf("after b: %s, want %s", status(m), want)
		}
	}
	m, _ = send(m, key("a"))
	if !strings.Contains(status(m), ", ascii, none, own, still]") {
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
	if !strings.Contains(status(m), ", blocks, none, own, still]") {
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
		if !strings.HasSuffix(status(m), ", "+want+", own, still]") {
			t.Errorf("after e: %s, want %s", status(m), want)
		}
		wall := m.View()
		if seen[wall] {
			t.Errorf("%s draws a wall already seen", want)
		}
		seen[wall] = true
	}
}

// i steps through the idle modes: on hover only the avatar under the
// pointer idles, then all of them, then none.
func TestIdleKeyStepsThroughTheModes(t *testing.T) {
	idling := func(m model) (n int, which int) {
		which = -1
		for i, a := range m.wall {
			if a.Idling() {
				n, which = n+1, i
			}
		}
		return n, which
	}
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 80, Height: 24})
	_, _, tileW := m.grid()
	over := func(tile int) tui.MouseEvent {
		return tui.MouseEvent{X: tile*(tileW+tileGap) + tileW/2, Y: wallTop + 1, Action: tui.MouseActionMotion}
	}

	// Hover mode with the pointer nowhere: nothing idles and no Cmd.
	m, cmd := send(m, key("i"))
	if n, _ := idling(m); n != 0 || cmd != nil || !strings.HasSuffix(status(m), ", hover]") {
		t.Fatalf("hover mode before the pointer moved: %d idling, Cmd %v, %s", n, cmd != nil, status(m))
	}
	// The pointer reaches the second tile: that avatar idles, alone.
	m, cmd = send(m, over(1))
	if n, which := idling(m); n != 1 || which != 1 || cmd == nil {
		t.Fatalf("pointer over tile 1: %d idling (avatar %d), Cmd %v", n, which, cmd != nil)
	}
	// Moving within the tile restarts nothing.
	still, cmd := send(m, tui.MouseEvent{X: over(1).X + 1, Y: wallTop + 2, Action: tui.MouseActionMotion})
	if n, which := idling(still); n != 1 || which != 1 || cmd != nil {
		t.Errorf("a move inside the tile: %d idling, Cmd %v", n, cmd != nil)
	}
	// On to the third tile: the second stops and the third starts.
	m, cmd = send(m, over(2))
	if n, which := idling(m); n != 1 || which != 2 || cmd == nil {
		t.Errorf("pointer over tile 2: %d idling (avatar %d), Cmd %v", n, which, cmd != nil)
	}
	// Off the wall: everything rests.
	off, cmd := send(m, tui.MouseEvent{X: 3, Y: 0, Action: tui.MouseActionMotion})
	if n, _ := idling(off); n != 0 || cmd != nil {
		t.Errorf("pointer off the wall: %d idling, Cmd %v", n, cmd != nil)
	}

	// Always: every avatar idles, wherever the pointer is.
	m, cmd = send(m, key("i"))
	if n, _ := idling(m); n != len(names) || cmd == nil || !strings.HasSuffix(status(m), ", idle]") {
		t.Errorf("always: %d idling, Cmd %v, %s", n, cmd != nil, status(m))
	}
	if moved, _ := send(m, over(0)); func() int { n, _ := idling(moved); return n }() != len(names) {
		t.Error("a mouse move in the always mode stopped an avatar")
	}
	// Off: none.
	m, cmd = send(m, key("i"))
	if n, _ := idling(m); n != 0 || cmd != nil || !strings.HasSuffix(status(m), ", still]") {
		t.Errorf("off: %d idling, Cmd %v, %s", n, cmd != nil, status(m))
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
		if !strings.HasSuffix(status(m), ", "+want+", still]") {
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

// g draws the wall as images when the terminal reported a graphics protocol,
// through that protocol, in the same cells; with none it keeps the cells and
// says so.
func TestImageModeFollowsTheTerminal(t *testing.T) {
	// Each case gets a model of its own: copies of one share their avatars.
	fresh := func() model {
		m, _ := send(initialModel(), tui.ResizeMsg{Width: 80, Height: 24})
		return m
	}
	cells := fresh().View()
	for name, c := range map[string]struct {
		caps   tui.Capabilities
		escape string
	}{
		"kitty":     {tui.Capabilities{KittyGraphics: true}, "\x1b_G"},
		"sixel":     {tui.Capabilities{Sixel: true}, "\x1bP"},
		"no images": {tui.Capabilities{}, ""},
	} {
		m, cmd := send(fresh(), tui.CapabilitiesMsg{Capabilities: c.caps})
		if cmd != nil || m.View() != cells {
			t.Errorf("%s: the probe's answer alone changed the wall", name)
		}
		m, _ = send(m, key("g"))
		// The selected avatar blinks at the key; put it back at rest.
		m.wall[0].StopIdle()
		view := m.View()
		if !strings.Contains(status(m), ", "+name+", ") {
			t.Errorf("%s: status is %s", name, status(m))
		}
		if c.escape == "" {
			if strings.Contains(view, "\x1b_G") || strings.Contains(view, "\x1bP") {
				t.Errorf("%s: an image was sent to a terminal that cannot draw one", name)
			}
			continue
		}
		cols, rows, _ := m.grid()
		if n := strings.Count(view, c.escape); n != cols*rows {
			t.Errorf("%s: %d images sent for %d tiles", name, n, cols*rows)
		}
		for _, l := range strings.Split(view, "\n") {
			if w := ansi.Width(l); w > 80 {
				t.Errorf("%s: a line is %d cells wide", name, w)
			}
		}
		// A second frame of the same wall sends the same bytes.
		if m.View() != view {
			t.Errorf("%s: an unchanged wall drew differently", name)
		}
		off, _ := send(m, key("g"))
		if strings.Contains(off.View(), c.escape) {
			t.Errorf("%s: g did not switch back to cells", name)
		}
	}
	// Kitty wins when a terminal has both.
	both, _ := send(fresh(), tui.CapabilitiesMsg{Capabilities: tui.Capabilities{KittyGraphics: true, Sixel: true}}, key("g"))
	if !strings.Contains(status(both), ", kitty, ") {
		t.Errorf("with both protocols: %s", status(both))
	}
}

// In image mode the eyes turn in steps, so a small move of the pointer
// redraws few avatars, where in cells it may redraw them all.
func TestImageModeRedrawsFewAvatarsOnASmallMove(t *testing.T) {
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 80, Height: 24}, tui.CapabilitiesMsg{Capabilities: tui.Capabilities{Sixel: true}})
	m.graphics = true
	looks := func(m model) [][2]float64 {
		cols, rows, tileW := m.grid()
		var out [][2]float64
		for i := 0; i < cols*rows; i++ {
			a := m.avatar(i, i%cols*(tileW+tileGap), wallTop+i/cols*(sizes[m.size][1]+2))
			out = append(out, [2]float64{a.LookX, a.LookY})
		}
		return out
	}
	at := func(x, y int) [][2]float64 {
		n, _ := send(m, tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionMotion})
		return looks(n)
	}
	before, after := at(40, 10), at(41, 10)
	moved := 0
	for i := range before {
		if before[i] != after[i] {
			moved++
		}
		for _, v := range before[i] {
			if r := v * lookSteps; math.Abs(r-math.Round(r)) > 1e-9 {
				t.Fatalf("avatar %d looks %v, not on a step", i, before[i])
			}
		}
	}
	if moved > len(before)/3 {
		t.Errorf("a one-cell move of the pointer redrew %d of %d avatars", moved, len(before))
	}
	// Across the screen the eyes do follow.
	far := at(0, 10)
	turned := 0
	for i := range before {
		if far[i] != before[i] {
			turned++
		}
	}
	if turned < len(before)/2 {
		t.Errorf("a move across the screen turned only %d of %d avatars", turned, len(before))
	}
}
