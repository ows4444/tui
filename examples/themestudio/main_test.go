package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func send(m model, msg tui.Msg) model {
	next, _ := m.Update(msg)
	return next.(model)
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model { return send(newModel(), tui.ResizeMsg{Width: 110, Height: 32}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func press(m model, x, y int) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 1})
}

func click(t testing.TB, m model, id string) model {
	r := rect(t, m, id)
	return press(m, r.X+min(1, r.W-1), r.Y)
}

func TestFirstScreen(t *testing.T) {
	out := text(ready())
	for _, want := range []string{"themestudio", "tokyo-night", "Primary", "#7aa2f7", "Background", "Overlay", "R ", "contrast", "Preview", "done", "A card on the surface", "x reset", "s save"} {
		if want == "contrast" {
			want = ":1 on background"
		}
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if len(tokens) != 15 || len(ready().names) != 20 {
		t.Fatalf("%d tokens, %d themes", len(tokens), len(ready().names))
	}
}

func TestEveryPresetBuildsAndRoundTrips(t *testing.T) {
	m := ready()
	for range m.names {
		m = runes(m, "]")
		if m.err != nil {
			t.Fatalf("%s: %v", m.sp.base, m.err)
		}
		want, _ := theme.Preset(m.sp.base)
		if toRGB(m.t.Primary) != toRGB(want.Primary) || toRGB(m.t.Background) != toRGB(want.Background) {
			t.Fatalf("%s differs from its preset", m.sp.base)
		}
		if !strings.Contains(text(m), m.sp.base) {
			t.Fatalf("%s not shown", m.sp.base)
		}
	}
}

func TestSlidersChangeTheColourAndThePreview(t *testing.T) {
	m := ready()
	before := m.current()
	m = key(m, tui.KeyRight)
	if m.current().R != before.R+1 || !strings.Contains(text(m), hex(m.current())) || !m.changedMark(0) {
		t.Fatalf("red %d -> %d", before.R, m.current().R)
	}
	m = key(key(m, tui.KeyTab), tui.KeyPgUp)
	if m.focus != fG || m.current().G != before.G+16 {
		t.Fatalf("green %d", m.current().G)
	}
	m = key(m, tui.KeyPgDown)
	m = key(key(m, tui.KeyLeft), tui.KeyLeft)
	if m.current().G != before.G-2 {
		t.Fatalf("green %d", m.current().G)
	}
	for i := 0; i < 20; i++ {
		m = key(m, tui.KeyPgUp)
	}
	if m.current().G != 255 {
		t.Fatalf("green past 255: %d", m.current().G)
	}
	if m.sp.over["primary"] != hex(m.current()) {
		t.Fatalf("override %q", m.sp.over["primary"])
	}
	if toRGB(m.t.Primary) != m.current() {
		t.Fatal("the preview theme did not take the colour")
	}
}

func TestSettingTheBaseColourBackRemovesTheOverride(t *testing.T) {
	m := key(ready(), tui.KeyRight)
	m = key(m, tui.KeyLeft)
	if len(m.sp.over) != 0 || m.changedMark(0) {
		t.Fatalf("override kept: %v", m.sp.over)
	}
}

func TestHexTypingAppliesAndRejects(t *testing.T) {
	m := key(ready(), tui.KeyEnter)
	if !m.typing || string(m.edit) != "#7aa2f7" {
		t.Fatalf("edit %q", string(m.edit))
	}
	for range 8 {
		m = key(m, tui.KeyBackspace)
	}
	m = runes(m, "zz#ff8800")
	if string(m.edit) != "#ff8800" {
		t.Fatalf("edit %q: letters outside a-f must be ignored", string(m.edit))
	}
	if !strings.Contains(text(m), "#ff8800▏") {
		t.Fatalf("no caret:\n%s", text(m))
	}
	m = key(m, tui.KeyEnter)
	if m.typing || m.current() != (ansi.RGB{R: 0xff, G: 0x88, B: 0}) {
		t.Fatalf("typing %v colour %v", m.typing, m.current())
	}
	m = runes(key(m, tui.KeyEnter), "")
	m = key(m, tui.KeyBackspace)
	m = key(m, tui.KeyEnter)
	if !strings.Contains(m.msg, "not a colour") {
		t.Fatalf("msg %q", m.msg)
	}
	m = key(key(m, tui.KeyEnter), tui.KeyEsc)
	if m.typing {
		t.Fatal("Esc did not cancel")
	}
	if c, ok := parseHex("#f80"); !ok || c != (ansi.RGB{R: 0xff, G: 0x88, B: 0}) {
		t.Fatalf("short hex: %v %v", c, ok)
	}
}

func TestClickAndDragOnASlider(t *testing.T) {
	m := ready()
	r := rect(t, m, "bar-0")
	m = press(m, r.X+r.W-1, r.Y)
	if m.current().R != 255 || m.focus != fR {
		t.Fatalf("red %d", m.current().R)
	}
	m = send(m, tui.MouseEvent{X: r.X, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionMotion})
	if m.current().R != 0 {
		t.Fatalf("drag to the left end: red %d", m.current().R)
	}
	m = send(m, tui.MouseEvent{X: r.X, Y: r.Y, Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.current().R != 4 {
		t.Fatalf("wheel: red %d", m.current().R)
	}
}

func TestClickingAColourPicksIt(t *testing.T) {
	m := click(t, ready(), "tok-12")
	if m.sel != 12 || !strings.Contains(text(m), "▸ ") || !strings.Contains(text(m), "Background") {
		t.Fatalf("sel %d", m.sel)
	}
	m = runes(runes(m, "j"), "j")
	if m.sel != 14 {
		t.Fatalf("sel %d", m.sel)
	}
	if m = runes(runes(m, "j"), "k"); m.sel != 13 {
		t.Fatalf("sel %d", m.sel)
	}
}

func TestBaseChangeClearsChangesAndSaysSo(t *testing.T) {
	m := key(ready(), tui.KeyRight)
	m = runes(m, "]")
	if len(m.sp.over) != 0 || !strings.Contains(m.msg, "cleared") {
		t.Fatalf("over %v msg %q", m.sp.over, m.msg)
	}
	i := m.baseIndex()
	m = click(t, m, "next")
	if m.baseIndex() != (i+1)%20 {
		t.Fatalf("next: %d", m.baseIndex())
	}
	m = click(t, click(t, m, "prev"), "prev")
	if m.baseIndex() != (i+19)%20 {
		t.Fatalf("prev: %d", m.baseIndex())
	}
}

func TestBorderAndGlyphsCycle(t *testing.T) {
	m := ready()
	for range 5 {
		m = runes(m, "b")
	}
	if m.sp.border != 5 || toRGB(m.t.Primary) != toRGB(ready().t.Primary) || m.t.Border.TopLeft != "┏" {
		t.Fatalf("border %d %q", m.sp.border, m.t.Border.TopLeft)
	}
	if !strings.Contains(text(m), "┏") || !strings.Contains(text(m), "border: thick") {
		t.Fatalf("preview lacks the thick border:\n%s", text(m))
	}
	m = runes(runes(m, "b"), "g")
	if m.sp.border != 0 || m.sp.glyphs != 1 {
		t.Fatalf("border %d glyphs %d", m.sp.border, m.sp.glyphs)
	}
	m = runes(m, "g")
	if !strings.Contains(text(m), "#########") || !strings.Contains(text(m), "glyphs: ascii") {
		t.Fatalf("ascii glyphs not drawn:\n%s", text(m))
	}
	m = click(t, m, "btn-border")
	if m.sp.border != 1 {
		t.Fatalf("click: border %d", m.sp.border)
	}
}

func TestResetColourAndAll(t *testing.T) {
	m := key(key(ready(), tui.KeyRight), tui.KeyDown)
	m = key(m, tui.KeyRight)
	m = runes(m, "x")
	if m.changedMark(1) || !m.changedMark(0) {
		t.Fatalf("over %v", m.sp.over)
	}
	m = runes(runes(m, "b"), "0")
	if m.sp.changes() != 0 || !strings.Contains(m.msg, "Back to") {
		t.Fatalf("changes %d", m.sp.changes())
	}
	m = key(m, tui.KeyRight)
	if m = click(t, m, "btn-all"); m.sp.changes() != 0 {
		t.Fatal("button did not reset")
	}
	m = key(m, tui.KeyRight)
	if m = click(t, m, "btn-reset"); m.changedMark(0) {
		t.Fatal("reset button did not reset the colour")
	}
}

func TestDocumentIsOrderedAndParsesBack(t *testing.T) {
	m := ready()
	m = key(m, tui.KeyRight)
	m = click(t, m, "tok-8")
	m = key(m, tui.KeyLeft)
	m = runes(runes(m, "b"), "g")
	doc := string(m.sp.document())
	want := "{\n  \"base\": \"tokyo-night\",\n  \"primary\": \"#7ba2f7\",\n  \"text_inverse\": \"#191b26\",\n  \"border\": \"normal\",\n  \"glyphs\": \"unicode\"\n}\n"
	if doc != want {
		t.Fatalf("%q", doc)
	}
	back, err := theme.Parse(strings.NewReader(doc))
	if err != nil || toRGB(back.Primary) != toRGB(m.t.Primary) || toRGB(back.TextInverse) != toRGB(m.t.TextInverse) {
		t.Fatalf("round trip: %v", err)
	}
	if got := string(spec{base: "nord", over: map[string]string{}}.document()); got != "{\n  \"base\": \"nord\"\n}\n" {
		t.Fatalf("%q", got)
	}
}

func TestSaveWritesTheFile(t *testing.T) {
	m := key(ready(), tui.KeyRight)
	m.path = filepath.Join(t.TempDir(), "t.json")
	m = runes(m, "s")
	got, err := os.ReadFile(m.path)
	if err != nil || string(got) != string(m.sp.document()) || !strings.Contains(m.msg, "Saved") {
		t.Fatalf("%q %v", got, err)
	}
	m.path = filepath.Join(t.TempDir(), "gone", "t.json")
	if m = click(t, m, "btn-save"); !strings.Contains(m.msg, "Could not save") {
		t.Fatalf("msg %q", m.msg)
	}
}

func TestContrastIsReportedAndFlagged(t *testing.T) {
	m := click(t, ready(), "tok-6") // Muted
	m = runes(m, "0")
	for i := 0; i < 40; i++ {
		m = key(m, tui.KeyPgDown)
	}
	m = runes(key(m, tui.KeyTab), "")
	m.sp.over = map[string]string{"muted": "#1a1b26"}
	m = m.rebuild()
	out := text(m)
	if !strings.Contains(out, "✗ below 4.5:1") || !strings.Contains(out, "Below 4.5:1") {
		t.Fatalf("a colour equal to the background should fail:\n%s", out)
	}
	if !strings.Contains(text(runes(ready(), "0")), "✓ AA") {
		t.Fatal("the stock primary should pass AA")
	}
}

func TestSmallWindowsKeepJustTheColourList(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 60, Height: 12})
	out := text(m)
	if strings.Contains(out, "Preview") || !strings.Contains(out, "Primary") {
		t.Fatalf("\n%s", out)
	}
	m = send(ready(), tui.ResizeMsg{Width: 300, Height: 80})
	if !strings.Contains(text(m), "Overlay: menu or dialog") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := runes(ready(), "?")
	if !strings.Contains(text(m), "change the base theme") {
		t.Fatalf("no help:\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	if _, c := ready().Update(tui.Key{Type: tui.KeyCtrlC}); c == nil {
		t.Fatal("Ctrl+C did not quit")
	}
	if _, c := ready().Update(tui.Key{Type: tui.KeyRunes, Text: "q"}); c == nil {
		t.Fatal("q did not quit")
	}
}

func TestSaveIsOffWithoutOut(t *testing.T) {
	m := key(ready(), tui.KeyRight)
	if m.path != "" {
		t.Fatalf("path %q: the default must write nothing", m.path)
	}
	if m = runes(m, "s"); !strings.Contains(m.msg, "-out FILE") {
		t.Fatalf("msg %q", m.msg)
	}
}
