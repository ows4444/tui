package render

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/vtscreen"
)

var logWords = []string{"connected", "retrying", "timeout", "cache", "miss", "hit", "flushed", "queue", "worker", "started", "stopped", "request", "response", "upstream", "latency", "error", "warning", "disk", "memory", "token"}

// logLine is entry k of a log, padded to width, with a coloured counter so the
// rows carry styles.
func logLine(k, width int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\x1b[32m%05d\x1b[0m ", k)
	x := uint32(k)*2654435761 + 12345
	for b.Len() < width-14 {
		x = x*1664525 + 1013904223
		b.WriteString(logWords[(x>>16)%uint32(len(logWords))])
		b.WriteByte(' ')
	}
	l := b.String()
	return l + strings.Repeat(" ", max(0, width-ansi.Width(l)))
}

func logLines(from, n, width int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = logLine(from+i, width)
	}
	return out
}

// screenRunner draws frames with one Cells onto a vtscreen, the way a Program in
// the alternate screen does: every frame starts at the top-left.
type screenRunner struct {
	c      *Cells
	scr    *vtscreen.Screen
	w, h   int
	prev   int
	scroll bool
	bytes  int
	last   string
}

func newRunner(w, h int, scroll bool) *screenRunner {
	return &screenRunner{c: New(), scr: vtscreen.NewScreen(w, h), w: w, h: h, scroll: scroll}
}

func (r *screenRunner) draw(t *testing.T, lines []string) {
	t.Helper()
	out, _, ok := r.c.Frame(Frame{
		Lines: append([]string(nil), lines...), Max: len(lines), Width: r.w, Height: r.h, PrevRows: r.prev,
		RegionTop: func() string { return "\x1b[H" }, FitLine: func(s string) string { return s },
		HardwareScroll: r.scroll,
	})
	if !ok {
		t.Fatal("the cell renderer refused the frame")
	}
	r.prev = len(lines)
	r.bytes += len(out)
	r.last = out
	r.scr.Write([]byte(out))
}

// same fails t unless the two screens hold the same runes and styles.
func same(t *testing.T, got, want *vtscreen.Screen, w, h int, what string) {
	t.Helper()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if g, e := got.Cell(x, y), want.Cell(x, y); g != e {
				t.Fatalf("%s: cell (%d,%d) = %+v, want %+v\ngot:\n%s\nwant:\n%s", what, x, y, g, e,
					strings.Join(got.Lines(), "\n"), strings.Join(want.Lines(), "\n"))
			}
		}
	}
}

func TestScrollingALogByOneLineWritesAboutOneRow(t *testing.T) {
	const w, h = 100, 40
	on, off := newRunner(w, h, true), newRunner(w, h, false)
	on.draw(t, logLines(0, h, w))
	off.draw(t, logLines(0, h, w))
	startOn, startOff := on.bytes, off.bytes
	for step := 1; step <= 10; step++ {
		before := on.bytes
		on.draw(t, logLines(step, h, w))
		off.draw(t, logLines(step, h, w))
		same(t, on.scr, off.scr, w, h, fmt.Sprintf("step %d", step))
		if got := on.bytes - before; got > 400 {
			t.Errorf("step %d: %d bytes with hardware scrolling, want about one row (%d wide)", step, got, w)
		}
		if !strings.Contains(on.last, "\x1b[1;40r\x1b[1S\x1b[r") {
			t.Errorf("step %d: no scroll sequence in %q", step, on.last[:min(60, len(on.last))])
		}
	}
	gotOn, gotOff := on.bytes-startOn, off.bytes-startOff
	t.Logf("10 one-line scrolls at %dx%d: %d bytes with hardware scrolling, %d without", w, h, gotOn, gotOff)
	if gotOn*10 > gotOff {
		t.Errorf("hardware scrolling saved less than 10x: %d against %d bytes", gotOn, gotOff)
	}
}

func TestScrollBothWaysAndByManyLines(t *testing.T) {
	const w, h = 60, 40
	for _, delta := range []int{1, 2, 5, -1, -3, 8, -8, 16, -16} {
		on, off := newRunner(w, h, true), newRunner(w, h, false)
		on.draw(t, logLines(100, h, w))
		off.draw(t, logLines(100, h, w))
		on.draw(t, logLines(100+delta, h, w))
		off.draw(t, logLines(100+delta, h, w))
		same(t, on.scr, off.scr, w, h, fmt.Sprintf("delta %d", delta))
		if want := fmt.Sprintf("\x1b[%d%s", abs(delta), map[bool]string{true: "S", false: "T"}[delta > 0]); !strings.Contains(on.last, want) {
			t.Errorf("delta %d: no %q in the output", delta, want)
		}
		// Only the rows that opened up are written, plus the few bytes of the scroll.
		if limit := abs(delta)*(w+50) + 60; len(on.last) > limit {
			t.Errorf("delta %d: %d bytes written, want at most %d (about %d new rows)", delta, len(on.last), limit, abs(delta))
		}
		if want := fmt.Sprintf("\x1b[1;%dr", h); !strings.Contains(on.last, want) {
			t.Errorf("delta %d: the whole screen should scroll (%q)", delta, want)
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// A header and a footer that do not move stay put; only the rows between scroll.
func TestScrollOnlyTheBlockThatMoved(t *testing.T) {
	const w, h = 60, 20
	frame := func(from int) []string {
		lines := logLines(from, h, w)
		lines[0] = "HEADER" + strings.Repeat(" ", w-6)
		lines[h-1] = "footer: scroll to see more" + strings.Repeat(" ", w-26)
		return lines
	}
	on, off := newRunner(w, h, true), newRunner(w, h, false)
	on.draw(t, frame(0))
	off.draw(t, frame(0))
	on.draw(t, frame(2))
	off.draw(t, frame(2))
	same(t, on.scr, off.scr, w, h, "header and footer")
	// The block is rows 1 to h-2; the log moved up by two lines.
	if !strings.Contains(on.last, "\x1b[2;19r\x1b[2S\x1b[r") {
		t.Errorf("scroll region is not the middle block: %q", on.last[:min(70, len(on.last))])
	}
	if on.bytes >= off.bytes {
		t.Errorf("no saving: %d bytes against %d", on.bytes, off.bytes)
	}
}

func TestRandomisedFramesAlwaysEndUpOnTheSameScreen(t *testing.T) {
	const w, h = 40, 16
	r := rand.New(rand.NewSource(3))
	on, off := newRunner(w, h, true), newRunner(w, h, false)
	base := 0
	lines := logLines(base, h, w)
	on.draw(t, lines)
	off.draw(t, lines)
	scrolls := 0
	for step := 0; step < 400; step++ {
		next := append([]string(nil), lines...)
		switch r.Intn(6) {
		case 0, 1: // scroll up
			s := 1 + r.Intn(6)
			base += s
			next = append(next[s:], logLines(base+h-s, s, w)...)
		case 2: // scroll down
			s := 1 + r.Intn(6)
			base -= s
			next = append(logLines(base, s, w), next[:h-s]...)
		case 3: // edit some rows
			for k := 0; k < 1+r.Intn(4); k++ {
				next[r.Intn(h)] = logLine(1000+r.Intn(50), w)
			}
		case 4: // a moved block with a fixed header and some edits
			next[0] = logLine(7777, w)
			s := 1 + r.Intn(4)
			next = append(append([]string{next[0]}, next[1+s:]...), logLines(base+h+step, s, w)...)
		default: // no change
		}
		on.draw(t, next)
		off.draw(t, next)
		if strings.Contains(on.last, "r\x1b[") && strings.Contains(on.last, "S") || strings.Contains(on.last, "T\x1b[r") {
			scrolls++
		}
		same(t, on.scr, off.scr, w, h, fmt.Sprintf("step %d", step))
		lines = next
	}
	if scrolls < 50 {
		t.Errorf("only %d of 400 frames scrolled: the test does not exercise the feature", scrolls)
	}
	if on.bytes >= off.bytes {
		t.Errorf("no saving over a random run: %d bytes against %d", on.bytes, off.bytes)
	}
}

func TestAShiftOfMoreThanHalfTheRowsIsNotLookedFor(t *testing.T) {
	const w, h = 60, 24
	r := newRunner(w, h, true)
	r.draw(t, logLines(0, h, w))
	r.draw(t, logLines(20, h, w))
	if strings.Contains(r.last, "\x1b[r") {
		t.Error("scrolled by 20 of 24 rows")
	}
}

func TestNoScrollWhenItWouldNotHelpOrIsNotAllowed(t *testing.T) {
	const w, h = 40, 12
	check := func(name string, scroll bool, a, b []string) {
		t.Helper()
		r := newRunner(w, h, scroll)
		r.draw(t, a)
		r.draw(t, b)
		if strings.Contains(r.last, "r") && strings.Contains(r.last, "\x1b[1S") || strings.Contains(r.last, "\x1b[r") {
			t.Errorf("%s: scrolled: %q", name, r.last)
		}
	}
	log := logLines(0, h, w)
	scrolled := append(append([]string(nil), log[1:]...), logLine(99, w))
	check("not allowed", false, log, scrolled)
	// Two rows moved only: not worth the bytes.
	few := append([]string(nil), log...)
	few[3], few[4] = log[4], log[3]
	check("a small change", true, log, few)
	// Every row the same: shifted equals unshifted.
	flat := make([]string, h)
	for i := range flat {
		flat[i] = strings.Repeat("x", w)
	}
	check("identical rows", true, flat, flat)
	// A different number of rows.
	r := newRunner(w, h, true)
	r.draw(t, log)
	r.draw(t, scrolled[:h-2])
	if strings.Contains(r.last, "\x1b[r") {
		t.Error("scrolled although the number of rows changed")
	}
}

func TestRowsWithImagesAreNeverScrolled(t *testing.T) {
	const w, h = 40, 12
	r := newRunner(w, h, true)
	log := logLines(0, h, w)
	withImage := append([]string(nil), log...)
	withImage[5] = "\x1b_Ga=T,f=100,i=7;AAAA\x1b\\" + withImage[5]
	r.draw(t, withImage)
	next := append(append([]string(nil), withImage[1:]...), logLine(99, w))
	r.draw(t, next)
	if strings.Contains(r.last, "\x1b[r") {
		t.Error("a block holding an image row was scrolled")
	}
}

func TestMoveSequences(t *testing.T) {
	if got := string(scrollMove{top: 0, bottom: 40, n: 1}.appendSeq(nil)); got != "\x1b[1;40r\x1b[1S\x1b[r" {
		t.Errorf("up: %q", got)
	}
	if got := string(scrollMove{top: 3, bottom: 11, n: -4}.appendSeq(nil)); got != "\x1b[4;11r\x1b[4T\x1b[r" {
		t.Errorf("down: %q", got)
	}
}

// A move that explains only a few of the changed rows is not worth a scroll.
func TestAMoveThatExplainsLittleIsNotScrolled(t *testing.T) {
	const w, h = 40, 20
	log := logLines(0, h, w)
	next := make([]string, h)
	copy(next, log)
	// Rows 0 to 4 are rows 1 to 5 moved up (a run of 5 that did change by index) ...
	copy(next[0:5], log[1:6])
	// ... while the other 15 rows are all different text.
	for i := 5; i < h; i++ {
		next[i] = logLine(5000+i, w)
	}
	r := newRunner(w, h, true)
	r.draw(t, log)
	r.draw(t, next)
	if strings.Contains(r.last, "\x1b[r") {
		t.Errorf("scrolled to save 5 of 20 changed rows: %q", r.last[:min(50, len(r.last))])
	}
}

// BenchmarkFrameChangedNoScroll is the worst case for the detection: every row
// changes and none of it is a move, so every shift is tried and none matches.
func BenchmarkFrameChangedNoScroll(b *testing.B) {
	for _, scroll := range []bool{false, true} {
		b.Run(fmt.Sprintf("scroll=%v", scroll), func(b *testing.B) {
			const w, h = 200, 60
			c := New()
			frames := [2][]string{logLines(0, h, w), logLines(1000, h, w)}
			f := Frame{Max: h, Width: w, Height: h, RegionTop: func() string { return "\x1b[H" }, FitLine: func(s string) string { return s }, HardwareScroll: scroll}
			f.Lines = frames[0]
			c.Frame(f)
			f.PrevRows = h
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				f.Lines = frames[(i+1)%2]
				c.Frame(f)
			}
		})
	}
}
