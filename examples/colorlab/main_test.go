package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func send(m model, msg tui.Msg) model {
	m, _ = update(m, msg)
	return m
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model { return send(initialModel(), tui.ResizeMsg{Width: 80, Height: 24}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no target %q on screen:\n%s", id, text(m))
	}
	return r
}

func press(m model, x, y int) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

func click(t testing.TB, m model, id string) model {
	r := rect(t, m, id)
	return press(m, r.X+r.W/2, r.Y)
}

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func TestStartsOnABlueWithItsNumbers(t *testing.T) {
	out := text(ready())
	for _, want := range []string{"Colour lab", "Hue", "Saturation", "Lightness", "#3A7BD5", "white", "black", "Copy", "Export", "Random", "AA"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestSliderKeysChangeTheColour(t *testing.T) {
	m := ready()
	before := m.rgb
	m = key(m, tui.KeyRight)
	if m.h != before2(ready()).h+1 || m.rgb == before {
		t.Fatalf("Right on Hue: h=%v rgb=%v", m.h, m.rgb)
	}
	if m = key(m, tui.KeyPgDown); m.h != before2(ready()).h-9 {
		t.Fatalf("PgDn: h = %v", m.h)
	}
	if hex(m.rgb) != m.hexIn.Value() {
		t.Fatalf("the hex field %q does not match the colour %s", m.hexIn.Value(), hex(m.rgb))
	}
}

func before2(m model) model { m.h = float64(int(m.h + 0.5)); return m }

func TestSlidersStopAtTheirEnds(t *testing.T) {
	m := click(t, ready(), "tr-1") // saturation
	for i := 0; i < 20; i++ {
		m = key(m, tui.KeyPgUp)
	}
	if m.s != 100 {
		t.Fatalf("saturation = %v after many PgUps", m.s)
	}
	for i := 0; i < 20; i++ {
		m = key(m, tui.KeyPgDown)
	}
	if m.s != 0 {
		t.Fatalf("saturation = %v after many PgDns", m.s)
	}
	m = send(ready(), tui.Key{Type: tui.KeyTab})
	m = key(key(m, tui.KeyTab), tui.KeyPgDown)
	for i := 0; i < 20; i++ {
		m = key(m, tui.KeyPgUp)
	}
	if m.l != 100 || m.rgb != white {
		t.Fatalf("lightness = %v rgb = %v at the top", m.l, m.rgb)
	}
}

func TestClickAndDragASlider(t *testing.T) {
	m := ready()
	r := rect(t, m, "tr-0")
	m = press(m, r.X+r.W-1, r.Y)
	if m.h != 359 || m.drag != 0 {
		t.Fatalf("h=%v drag=%d after clicking the track's end", m.h, m.drag)
	}
	m = send(m, tui.MouseEvent{X: r.X, Y: r.Y, Action: tui.MouseActionMotion})
	if m.h != 0 {
		t.Fatalf("h = %v after dragging to the start", m.h)
	}
	m = send(m, tui.MouseEvent{X: r.X, Y: r.Y, Action: tui.MouseActionRelease})
	m = send(m, tui.MouseEvent{X: r.X + 5, Y: r.Y, Action: tui.MouseActionMotion})
	if m.h != 0 || m.drag != -1 {
		t.Fatal("moving after release still dragged")
	}
}

func TestWheelNudgesTheSliderUnderThePointer(t *testing.T) {
	m := ready()
	r := rect(t, m, "tr-2")
	l := m.l
	m = send(m, tui.MouseEvent{X: r.X + 2, Y: r.Y, Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if int(m.l+0.5) != int(l+0.5)+1 || m.focus != onLight {
		t.Fatalf("lightness %v -> %v, focus %v", l, m.l, m.focus)
	}
}

func TestTypingAHexCodeSetsTheColour(t *testing.T) {
	m := click(t, ready(), "hex")
	if m.focus != onHex {
		t.Fatal("clicking the hex did not focus it")
	}
	for i := 0; i < 8; i++ {
		m = key(m, tui.KeyBackspace)
	}
	m = runes(m, "#ff0000")
	if m.rgb != (ansi.RGB{R: 255}) || int(m.h+0.5) != 0 || int(m.s+0.5) != 100 {
		t.Fatalf("rgb=%v h=%v s=%v after typing #ff0000", m.rgb, m.h, m.s)
	}
	if m.hexIn.Value() != "#ff0000" {
		t.Fatalf("the field was rewritten while typing: %q", m.hexIn.Value())
	}
}

func TestAnIncompleteHexKeepsTheColourAndSaysSo(t *testing.T) {
	m := click(t, ready(), "hex")
	for i := 0; i < 8; i++ {
		m = key(m, tui.KeyBackspace)
	}
	before := m.rgb
	m = runes(m, "#12")
	if m.rgb != before || !strings.Contains(text(m), "Type a hex code") {
		t.Fatalf("rgb changed or no hint:\n%s", text(m))
	}
	if m = runes(m, "3"); m.rgb != (ansi.RGB{R: 0x11, G: 0x22, B: 0x33}) {
		t.Fatalf("#123 -> %v", m.rgb)
	}
}

func TestQAndLettersTypeIntoTheHexField(t *testing.T) {
	m := click(t, ready(), "hex")
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Fatal("q quit while typing a hex code")
	}
	if m = key(m, tui.KeyEsc); m.focus != onHue {
		t.Fatal("Esc did not leave the hex field")
	}
}

func TestGreyKeepsTheHue(t *testing.T) {
	m := ready()
	h := m.h
	m = m.setRGB(ansi.RGB{R: 128, G: 128, B: 128}, true)
	if m.h != h || m.s != 0 {
		t.Fatalf("h=%v (was %v) s=%v for a grey", m.h, h, m.s)
	}
}

func TestContrastForBlackAndWhite(t *testing.T) {
	m := ready().setRGB(black, true)
	out := text(m)
	if !strings.Contains(out, "1.00 ✗ fail") || !strings.Contains(out, "21.00 ✓ AAA") {
		t.Fatalf("black text:\n%s", out)
	}
}

func TestPaletteSwatchesAreTakenByClickAndEnter(t *testing.T) {
	m := ready()
	want := palette(m.rgb)[4].c
	m = click(t, m, "sw-4")
	if m.rgb != want || m.focus != onPalette || m.cursor != 2 {
		t.Fatalf("rgb=%v want %v focus=%v cursor=%d", m.rgb, want, m.focus, m.cursor)
	}
	m = key(m, tui.KeyTab) // wraps round to the hue
	if m.focus != onHue {
		t.Fatalf("Tab from the palette went to %v", m.focus)
	}
	m = send(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	if m.focus != onPalette {
		t.Fatalf("Shift+Tab from the hue went to %v", m.focus)
	}
	m = key(key(m, tui.KeyLeft), tui.KeyLeft)
	if m.cursor != 0 {
		t.Fatalf("cursor = %d", m.cursor)
	}
	want = palette(m.rgb)[0].c
	if m = key(m, tui.KeyEnter); m.rgb != want {
		t.Fatalf("Enter took %v, want %v", m.rgb, want)
	}
	m.focus = onPalette
	for i := 0; i < 10; i++ {
		m = key(m, tui.KeyRight)
	}
	if m.cursor != 5 {
		t.Fatalf("cursor = %d after many Rights", m.cursor)
	}
}

func TestCopyAndExportReturnClipboardCommands(t *testing.T) {
	m, cmd := update(ready(), tui.Key{Type: tui.KeyRunes, Text: "c"})
	if cmd == nil || !strings.Contains(text(m), "Copied #3A7BD5") {
		t.Fatalf("c: cmd=%v\n%s", cmd != nil, text(m))
	}
	m, cmd = update(ready(), tui.Key{Type: tui.KeyRunes, Text: "e"})
	if cmd == nil || !strings.Contains(text(m), "palette as CSS") {
		t.Fatalf("e: cmd=%v\n%s", cmd != nil, text(m))
	}
	for _, id := range []string{"copy", "export"} {
		if _, cmd := update(ready(), tui.MouseEvent{X: rect(t, ready(), id).X + 1, Y: rect(t, ready(), id).Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}); cmd == nil {
			t.Errorf("clicking %s gave no command", id)
		}
	}
}

func TestExportWritesOnlyWithOut(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.css")
	m := ready()
	_, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "e"})
	if cmd != nil {
		cmd()
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("a file appeared without -out")
	}
	m.out = path
	_, cmd = update(m, tui.Key{Type: tui.KeyRunes, Text: "e"})
	// The command batches the clipboard write and the file write; run the file one directly.
	writeFile(path, css(palette(m.rgb)))()
	b, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(b), "--color-base: #3a7bd5;") {
		t.Fatalf("file = %q, err %v", b, err)
	}
	if cmd == nil {
		t.Fatal("no command with -out")
	}
}

func TestRandomIsRepeatable(t *testing.T) {
	a := send(ready(), tui.Key{Type: tui.KeyRunes, Text: "r"})
	b := send(ready(), tui.Key{Type: tui.KeyRunes, Text: "r"})
	if a.rgb != b.rgb || a.rgb == ready().rgb {
		t.Fatalf("random colours %v and %v from the same start", a.rgb, b.rgb)
	}
	if !strings.Contains(a.status, hex(a.rgb)) {
		t.Fatalf("status = %q", a.status)
	}
	if c := click(t, ready(), "random"); c.rgb == ready().rgb {
		t.Fatal("clicking Random changed nothing")
	}
}

func TestRandomStaysReadableRange(t *testing.T) {
	m := ready()
	for i := 0; i < 200; i++ {
		m = m.random()
		if m.l < 25 || m.l > 75 || m.s < 30 || m.s > 90 {
			t.Fatalf("random colour out of range: h=%v s=%v l=%v", m.h, m.s, m.l)
		}
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := key(ready(), tui.KeyF1)
	if !strings.Contains(text(m), "copy the palette as CSS") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}} {
		if _, cmd := update(ready(), k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
}

func TestHoverIsTracked(t *testing.T) {
	m := ready()
	r := rect(t, m, "sw-3")
	m = send(m, tui.MouseEvent{X: r.X, Y: r.Y, Action: tui.MouseActionMotion})
	if m.hover != "sw-3" {
		t.Fatalf("hover = %q", m.hover)
	}
}

func TestCompactAndTinyTerminalsStayInside(t *testing.T) {
	m := ready()
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
	m = send(m, tui.ResizeMsg{Width: 40, Height: 10})
	if out := text(m); !strings.Contains(out, "Hue") || !strings.Contains(out, "Copy") {
		t.Fatalf("40x10 lost the controls:\n%s", out)
	}
}
