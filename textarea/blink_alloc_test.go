package textarea

import (
	"runtime"
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

// Focus, Blur, Focus inside one blink interval leaves exactly one live loop:
// the tick of the first loop is dropped and does not reschedule.
func TestFocusBlurFocusRunsOneBlinkLoop(t *testing.T) {
	m := New()
	if m.Focus() == nil {
		t.Fatal("Focus returned no blink Cmd")
	}
	first := m.blinkID
	m.Blur()
	if m.Focus() == nil {
		t.Fatal("second Focus returned no blink Cmd")
	}
	second := m.blinkID
	if first == second {
		t.Fatalf("both loops share id %d", first)
	}

	next, cmd := m.Update(blinkMsg{id: first})
	if cmd != nil || next.cursorVisible != m.cursorVisible {
		t.Fatalf("stale tick was not dropped (cmd %v, visible %v -> %v)", cmd != nil, m.cursorVisible, next.cursorVisible)
	}
	next, cmd = m.Update(blinkMsg{id: second})
	if cmd == nil || next.cursorVisible == m.cursorVisible {
		t.Fatal("the live loop's tick should toggle and reschedule")
	}
}

// A tick from before a Blur is dropped, so the loop stops.
func TestBlurOrphansBlinkTick(t *testing.T) {
	m := New()
	m.Focus()
	id := m.blinkID
	m.Blur()
	m.Focus()
	if _, cmd := m.Update(blinkMsg{id: id}); cmd != nil {
		t.Fatal("orphaned tick rescheduled itself")
	}
}

// One keystroke into a 100,000-line, Height-limited textarea allocates less
// than 64 KB, so it does not scale with the buffer.
func TestTypingIntoHugeBufferAllocatesLittle(t *testing.T) {
	m := bigModel()
	key := tui.Key{Type: tui.KeyRunes, Text: "x"}
	const n = 50
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < n; i++ {
		m, _ = m.Update(key)
	}
	runtime.ReadMemStats(&after)
	if per := (after.TotalAlloc - before.TotalAlloc) / n; per >= 64<<10 {
		t.Fatalf("a keystroke allocated %d bytes, want < 65536", per)
	}
	if !strings.Contains(m.Value(), strings.Repeat("x", n)) {
		t.Fatal("typed text missing")
	}
}

// The doc stays consistent across group boundaries.
func TestDocSpansGroups(t *testing.T) {
	text := strings.Repeat("ab\n", grpLines*3+5) + "tail"
	d := newDoc([]rune(text))
	if d.String() != text || d.len() != len([]rune(text)) {
		t.Fatal("round trip changed the text")
	}
	if want := grpLines*3 + 6; d.lineCount() != want {
		t.Fatalf("lineCount = %d, want %d", d.lineCount(), want)
	}
	a, b := grpLines*3-2, grpLines*3*2
	if got := d.str(a, b); got != string([]rune(text)[a:b]) {
		t.Fatalf("slice across groups = %q", got)
	}
	e := d.replace(a, a, []rune("Z"))
	want := text[:a] + "Z" + text[a:]
	if e.String() != want || d.String() != text {
		t.Fatal("in-line insert wrong or mutated the original")
	}
	if got := docStarts(e); len(got) != len(flatStarts(want)) {
		t.Fatal("index length changed")
	}
	f := e.replace(0, e.len(), nil)
	if f.len() != 0 || f.lineCount() != 1 {
		t.Fatal("clearing left text or lines")
	}
}
