package main

import (
	"fmt"
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

// drain feeds the model what cmd and its successors produce until the source
// has ended, like the Program does.
func drain(m model, cmd tui.Cmd) model {
	for cmd != nil {
		msg := cmd()
		var next tui.Cmd
		m, next = update(m, msg)
		if _, done := msg.(eofMsg); done {
			break
		}
		cmd = next
	}
	return m
}

func lines(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "line %d of the text\n", i)
	}
	return b.String()
}

// load is a model at 80x12 that has read text to the end.
func load(text string) model {
	m := newModel("t.txt").begin(strings.NewReader(text), false)
	m = send(m, tui.ResizeMsg{Width: 80, Height: 12})
	return drain(m, m.start)
}

func text(m model) string { return ansi.StripANSI(m.View()) }

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func TestReadsTheWholeSourceAndShowsTheTop(t *testing.T) {
	m := load(lines(100))
	if m.vp.LineCount() != 100 || !m.eof || m.vp.YOffset() != 0 {
		t.Fatalf("lines=%d eof=%v offset=%d", m.vp.LineCount(), m.eof, m.vp.YOffset())
	}
	out := text(m)
	for _, want := range []string{"line 1 of the text", "line 10 of the text", "t.txt", "lines 1–10 of 100", "0%"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "reading…") {
		t.Error("still says reading after the end")
	}
}

func TestShowsReadingUntilTheEnd(t *testing.T) {
	m := newModel("p").begin(strings.NewReader(lines(5)), false)
	m = send(m, tui.ResizeMsg{Width: 80, Height: 12})
	if !strings.Contains(text(m), "reading…") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestScrollingKeys(t *testing.T) {
	m := load(lines(100))
	m = runes(m, "j")
	if m.vp.YOffset() != 1 {
		t.Fatalf("j: offset %d", m.vp.YOffset())
	}
	m = runes(m, " ")
	if m.vp.YOffset() != 1+m.vp.Height {
		t.Fatalf("space: offset %d", m.vp.YOffset())
	}
	m = runes(m, "b")
	if m.vp.YOffset() != 1 {
		t.Fatalf("b: offset %d", m.vp.YOffset())
	}
	m = runes(runes(m, "d"), "u")
	if m.vp.YOffset() != 1 {
		t.Fatalf("d then u: offset %d", m.vp.YOffset())
	}
	m = runes(m, "G")
	if !m.vp.AtBottom() || !strings.Contains(text(m), "100%") || !strings.Contains(text(m), "of 100") {
		t.Fatalf("G:\n%s", text(m))
	}
	m = runes(m, "g")
	if !m.vp.AtTop() {
		t.Fatal("g did not go to the top")
	}
	m = key(key(m, tui.KeyDown), tui.KeyDown)
	m = runes(m, "k")
	if m.vp.YOffset() != 1 {
		t.Fatalf("arrows: offset %d", m.vp.YOffset())
	}
	m = key(m, tui.KeyPgDown)
	if m.vp.YOffset() <= 1 {
		t.Fatal("PgDn did not page")
	}
}

func TestSearchHighlightsAndSteps(t *testing.T) {
	m := load(lines(100))
	m = runes(m, "/")
	if !m.typing || !strings.Contains(text(m), "/ ") {
		t.Fatal("/ did not open the search")
	}
	m = runes(m, "line 7")
	m = key(m, tui.KeyEnter)
	if m.vp.MatchCount() != 11 || !strings.Contains(text(m), "match 1/11") {
		t.Fatalf("matches=%d\n%s", m.vp.MatchCount(), text(m))
	}
	m = runes(m, "n")
	if m.vp.CurrentMatch() != 2 {
		t.Fatalf("n: current %d", m.vp.CurrentMatch())
	}
	m = runes(m, "N")
	m = runes(m, "N")
	if m.vp.CurrentMatch() != 11 {
		t.Fatalf("N did not wrap: %d", m.vp.CurrentMatch())
	}
	m = key(m, tui.KeyEsc)
	if m.query != "" || m.vp.MatchCount() != 0 {
		t.Fatal("Esc did not clear the search")
	}
}

func TestSearchNotFoundAndCancel(t *testing.T) {
	m := load(lines(30))
	m = runes(runes(m, "/"), "zzz")
	m = key(m, tui.KeyEnter)
	if !strings.Contains(text(m), "Pattern not found: zzz") {
		t.Fatalf("\n%s", text(m))
	}
	m = runes(m, "/")
	if m = key(m, tui.KeyEsc); m.typing {
		t.Fatal("Esc did not close the search box")
	}
	if _, cmd := update(runes(m, "/"), tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Fatal("q quit while typing a search")
	}
}

func TestWrapAndNumbers(t *testing.T) {
	long := strings.Repeat("0123456789", 12) + "\nshort\n"
	m := load(long)
	if !m.vp.SoftWrap {
		t.Fatal("wrapping should start on")
	}
	if rows := strings.Count(text(m), "0123456789"); rows < 2 {
		t.Fatalf("a long line did not wrap:\n%s", text(m))
	}
	m = runes(m, "#")
	if m.numbers || !strings.Contains(text(m), "Line numbers need wrapping off") {
		t.Fatalf("numbers on while wrapping:\n%s", text(m))
	}
	m = runes(m, "w")
	if m.vp.SoftWrap {
		t.Fatal("w did not turn wrapping off")
	}
	m = runes(m, "#")
	out := text(m)
	if !m.numbers || !strings.Contains(out, "1 0123456789") || !strings.Contains(out, "2 short") {
		t.Fatalf("numbers:\n%s", out)
	}
	m = key(m, tui.KeyRight)
	if !strings.Contains(text(m), "89012345") {
		t.Fatalf("Right did not scroll sideways:\n%s", text(m))
	}
	m = runes(m, "w")
	if m.vp.SoftWrap && m.numbers {
		t.Fatal("wrapping and numbers both on")
	}
}

func TestPercentBarClickJumps(t *testing.T) {
	m := load(lines(200))
	bar, ok := layout.RectOf(m.screen(), m.size(), "bar")
	if !ok {
		t.Fatal("no bar")
	}
	m = send(m, tui.MouseEvent{X: bar.X + bar.W - 1, Y: bar.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if !m.vp.AtBottom() {
		t.Fatalf("clicking the bar's end: offset %d", m.vp.YOffset())
	}
	m = send(m, tui.MouseEvent{X: bar.X, Y: bar.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if !m.vp.AtTop() {
		t.Fatalf("clicking the bar's start: offset %d", m.vp.YOffset())
	}
	m = send(m, tui.MouseEvent{X: bar.X + bar.W/2, Y: bar.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if o := m.vp.YOffset(); o < 80 || o > 120 {
		t.Fatalf("clicking the middle: offset %d of %d", o, m.vp.LineCount())
	}
}

func TestWheelScrolls(t *testing.T) {
	m := load(lines(100))
	m = send(m, tui.MouseEvent{X: 5, Y: 3, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.vp.YOffset() == 0 {
		t.Fatal("the wheel did not scroll")
	}
	m = send(m, tui.MouseEvent{X: 5, Y: 3, Action: tui.MouseActionMotion})
	if m.hover {
		t.Fatal("hover on the bar while over the text")
	}
	bar, _ := layout.RectOf(m.screen(), m.size(), "bar")
	m = send(m, tui.MouseEvent{X: bar.X, Y: bar.Y, Action: tui.MouseActionMotion})
	if !m.hover {
		t.Fatal("no hover over the bar")
	}
}

func TestFollowKeepsUpWithNewLinesWhileAtTheBottom(t *testing.T) {
	m := newModel("log").begin(strings.NewReader(""), false)
	m = send(m, tui.ResizeMsg{Width: 80, Height: 12})
	ch := make(chan item)
	m = runes(m, "F")
	if !m.sticky || !m.follow.Load() || !strings.Contains(text(m), "▶ following") {
		t.Fatalf("F did not turn following on:\n%s", text(m))
	}
	var batch []string
	for i := 1; i <= 50; i++ {
		batch = append(batch, fmt.Sprintf("entry %d", i))
	}
	m = send(m, batchMsg{lines: batch, ch: ch})
	if !m.vp.AtBottom() || !strings.Contains(text(m), "entry 50") {
		t.Fatalf("not at the end:\n%s", text(m))
	}
	m = runes(m, "g") // scroll away from the end
	m = send(m, batchMsg{lines: []string{"entry 51"}, ch: ch})
	if !m.vp.AtTop() {
		t.Fatal("new lines pulled a reader who scrolled away back to the end")
	}
	m = runes(m, "F")
	if m.sticky || m.follow.Load() {
		t.Fatal("F did not turn following off")
	}
}

func TestOldLinesAreDroppedAndNumbersStayTrue(t *testing.T) {
	m := load(lines(3))
	m.dropped = 0
	var batch []string
	for i := 0; i < maxLines+10; i++ {
		batch = append(batch, "x")
	}
	m = send(m, batchMsg{lines: batch, ch: make(chan item)})
	if m.vp.LineCount() != maxLines || m.dropped != 13 {
		t.Fatalf("lines=%d dropped=%d", m.vp.LineCount(), m.dropped)
	}
	m = runes(runes(m, "w"), "#")
	m = runes(m, "G")
	if !strings.Contains(text(m), fmt.Sprint(maxLines+13)) {
		t.Fatalf("the last line is not numbered %d:\n%s", maxLines+13, text(m))
	}
}

func TestReadErrorIsShown(t *testing.T) {
	m := load("a\n")
	m = send(m, eofMsg{err: fmt.Errorf("disk gone")})
	if !strings.Contains(text(m), "✗ disk gone") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestEmptyInput(t *testing.T) {
	m := load("")
	if !strings.Contains(text(m), "empty") {
		t.Fatalf("\n%s", text(m))
	}
	m = runes(runes(runes(m, "G"), "j"), "/")
	m = key(m, tui.KeyEsc)
}

func TestHelpAndQuit(t *testing.T) {
	m := runes(load(lines(5)), "?")
	if !strings.Contains(text(m), "next page") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	if m = key(m, tui.KeyF1); !m.help {
		t.Fatal("F1 did not open help")
	}
	m = key(m, tui.KeyEnter)
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}} {
		if _, cmd := update(load(lines(5)), k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
}

func TestQuitStopsTheReader(t *testing.T) {
	m := newModel("x").begin(strings.NewReader(lines(5)), true)
	m, cmd := update(m, tui.Key{Type: tui.KeyCtrlC})
	if !isQuit(cmd) {
		t.Fatal("Ctrl+C did not quit")
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := runes(load(lines(200)), "w")
	m = runes(m, "#")
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}, {3, 3}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
