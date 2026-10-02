package streamtext

import (
	"context"
	"github.com/ows4444/tui"
	"runtime"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// tickFor builds the tick Msg a Cmd from m would deliver.
func tickFor(m Model) tickMsg { return tickMsg{id: m.id} }

func TestNewIsEmptyAndIdle(t *testing.T) {
	m := New()
	if m.Running() {
		t.Error("New() should not be running")
	}
	if m.Text() != "" || m.View() != "" {
		t.Errorf("New() Text=%q View=%q, want empty", m.Text(), m.View())
	}
	if !m.Done() {
		t.Error("an empty model has nothing to reveal, so Done() should be true")
	}
}

func TestStartTicksOnlyWhenThereIsSomethingToReveal(t *testing.T) {
	m := New()
	if cmd := m.Start(); cmd != nil {
		t.Error("Start with no text should return nil (no idle ticking)")
	}
	if !m.Running() {
		t.Error("Start should mark the model running even with nothing to reveal")
	}

	m = New()
	m.SetText("hello")
	cmd := m.Start()
	if cmd == nil {
		t.Fatal("Start with text should return a tick Cmd")
	}
	if got, ok := tui.RunCmd(context.Background(), cmd).(tickMsg); !ok || got.id != m.id {
		t.Errorf("Start's Cmd produced %#v, want tickMsg for this model", tui.RunCmd(context.Background(), cmd))
	}
}

func TestTickRevealsCharsPerTickAndReschedules(t *testing.T) {
	m := New()
	m.CharsPerTick = 2
	m.Cursor = ""
	m.SetText("abcde")
	m.Start()

	m, c := m.Update(tickFor(m))
	if got := m.View(); got != "ab" {
		t.Errorf("after 1 tick View = %q, want %q", got, "ab")
	}
	if c == nil {
		t.Error("should reschedule while text remains")
	}
	m, c = m.Update(tickFor(m))
	if got := m.View(); got != "abcd" || c == nil {
		t.Errorf("after 2 ticks View = %q (cmd nil=%v), want %q and a reschedule", got, c == nil, "abcd")
	}
	m, c = m.Update(tickFor(m))
	if got := m.View(); got != "abcde" {
		t.Errorf("after 3 ticks View = %q, want full text", got)
	}
	if c != nil {
		t.Error("caught up: Update should return nil, ending the chain")
	}
	if !m.Done() {
		t.Error("Done() should be true once everything is revealed")
	}
}

func TestCharsPerTickBelowOneStillAdvances(t *testing.T) {
	m := New()
	m.CharsPerTick = 0
	m.Cursor = ""
	m.SetText("ab")
	m.Start()
	m, _ = m.Update(tickFor(m))
	if got := m.View(); got != "a" {
		t.Errorf("View = %q, want %q (minimum 1 per tick)", got, "a")
	}
}

func TestAppendSchedulesOnlyWhenIdle(t *testing.T) {
	m := New()
	m.Cursor = ""
	m.Start() // running, idle

	if cmd := m.Append("hi"); cmd == nil {
		t.Fatal("Append while running and idle should return a tick Cmd")
	}
	if cmd := m.Append(" there"); cmd != nil {
		t.Error("Append while a tick is in flight must return nil (one chain only)")
	}

	// Drain to idle, then Append again: needs a fresh tick.
	m.CharsPerTick = 100
	m, cmd := m.Update(tickFor(m))
	if cmd != nil || !m.Done() {
		t.Fatalf("expected caught up and idle; done=%v cmd nil=%v", m.Done(), cmd == nil)
	}
	if cmd := m.Append("!"); cmd == nil {
		t.Error("Append after going idle should restart ticking")
	}

	// Not running: Append just stores text.
	n := New()
	if cmd := n.Append("x"); cmd != nil {
		t.Error("Append while not running should not schedule a tick")
	}
	if n.Text() != "x" {
		t.Errorf("Text = %q, want x", n.Text())
	}
}

func TestStopEndsTheChain(t *testing.T) {
	m := New()
	m.Cursor = ""
	m.SetText("abcdef")
	m.Start()
	m.Stop()
	next, cmd := m.Update(tickFor(m))
	if cmd != nil {
		t.Error("a tick after Stop should not reschedule")
	}
	if next.View() != "" {
		t.Errorf("a tick after Stop revealed %q, want nothing", next.View())
	}

	// Start again while the old tick is still in flight: no second chain.
	m = New()
	m.SetText("abcdef")
	m.Start() // tick #1 in flight
	m.Stop()
	if cmd := m.Start(); cmd != nil {
		t.Error("Start while a tick is still in flight must not create a second chain")
	}
	// The in-flight tick then carries the animation on.
	_, cmd = m.Update(tickFor(m))
	if cmd == nil {
		t.Error("the in-flight tick should keep the restarted animation going")
	}
}

func TestTickForAnotherModelIsIgnored(t *testing.T) {
	a, b := New(), New()
	a.Cursor, b.Cursor = "", ""
	a.SetText("aaaa")
	b.SetText("bbbb")
	a.Start()
	b.Start()
	if a.id == b.id {
		t.Fatal("models must have distinct ids")
	}
	a2, cmd := a.Update(tickFor(b))
	if cmd != nil || a2.View() != "" {
		t.Errorf("model advanced on another model's tick: View=%q cmd nil=%v", a2.View(), cmd == nil)
	}
}

func TestSetTextRestartsAndSkipFinishes(t *testing.T) {
	m := New()
	m.Cursor = ""
	m.CharsPerTick = 3
	m.SetText("abcdef")
	m.Start()
	m, _ = m.Update(tickFor(m))
	if m.View() != "abc" {
		t.Fatalf("View = %q", m.View())
	}
	m.SetText("xyz123")
	if m.View() != "" {
		t.Errorf("SetText should restart the reveal, View = %q", m.View())
	}
	m.Skip()
	if !m.Done() || m.View() != "xyz123" {
		t.Errorf("Skip: Done=%v View=%q, want full text", m.Done(), m.View())
	}
}

func TestCursorOnlyWhileRevealing(t *testing.T) {
	th := theme.DarkTheme()
	m := New()
	m.Theme = th
	m.Cursor = "▌"
	m.CharsPerTick = 2
	m.SetText("abcd")
	m.Start()
	m, _ = m.Update(tickFor(m))
	cursor := ansi.NewStyle().Foreground(th.Primary).Render("▌")
	if got, want := m.View(), "ab"+cursor; got != want {
		t.Errorf("mid-reveal View = %q, want %q", got, want)
	}
	m.Skip()
	if got := m.View(); got != "abcd" {
		t.Errorf("done View = %q, want no cursor", got)
	}
	m.Cursor = ""
	m.SetText("abcd")
	m, _ = m.Update(tickFor(m))
	if strings.Contains(m.View(), "▌") {
		t.Error("empty Cursor should draw nothing")
	}
}

func TestRevealNeverBreaksEscapesOrWideRunes(t *testing.T) {
	styled := ansi.NewStyle().Bold().Render("bold") + " plain"
	wide := "你好世界"
	for _, text := range []string{styled, wide, "a" + wide + "\nb"} {
		m := New()
		m.Cursor = ""
		m.CharsPerTick = 1
		m.SetText(text)
		m.Start()
		total := ansi.Width(text)
		for i := 0; i <= total; i++ {
			v := m.View()
			// No dangling escape: every ESC starts a full CSI ... final byte.
			for j := 0; j < len(v); j++ {
				if v[j] == 0x1b {
					k := j + 2
					for k < len(v) && !(v[k] >= 0x40 && v[k] <= 0x7E) {
						k++
					}
					if j+1 >= len(v) || v[j+1] != '[' || k >= len(v) {
						t.Fatalf("text %q step %d: truncated escape in %q", text, i, v)
					}
				}
			}
			// Wide runes appear whole or not at all.
			if strings.ContainsRune(v, '�') {
				t.Fatalf("text %q step %d: broken rune in %q", text, i, v)
			}
			// Cut inside the bold span: styling must be closed again.
			if text == styled && i >= 1 && i <= 4 && !strings.HasSuffix(v, ansi.Reset) {
				t.Fatalf("step %d: style left open in %q", i, v)
			}
			m, _ = m.Update(tickFor(m))
		}
		if !m.Done() || m.View() != text {
			t.Errorf("text %q: final View = %q, want full text", text, m.View())
		}
	}
}

func TestWrapIsStableAsTextIsRevealed(t *testing.T) {
	text := "the quick brown fox jumps over the lazy dog and keeps running far away"
	for _, cursor := range []string{"", "▌"} {
		m := New()
		m.Cursor = cursor
		m.Width = 16
		m.CharsPerTick = 1
		m.SetText(text)
		m.Start()

		final := New()
		final.Cursor = cursor
		final.Width = 16
		final.SetText(text)
		final.Skip()
		want := final.View()

		total := ansi.Width(want)
		for i := 0; i <= total; i++ {
			got := ansi.StripANSI(m.View())
			got = strings.TrimSuffix(got, cursor) // cursor sits after the text
			if !strings.HasPrefix(want, got) {
				t.Fatalf("cursor %q step %d: %q is not a prefix of the final layout %q (text reflowed)", cursor, i, got, want)
			}
			for _, l := range strings.Split(got, "\n") {
				if ansi.Width(l)+ansi.Width(cursor) > 16 {
					t.Fatalf("cursor %q step %d: line %q too wide", cursor, i, l)
				}
			}
			m, _ = m.Update(tickFor(m))
		}
	}
}

func wrapTestModel(width int) Model {
	m := New()
	m.Width = width
	m.Cursor = ""
	return m
}

// Criterion: View after Append at unchanged width rewraps only the last paragraph.
func TestAppendRewrapsOnlyLastParagraph(t *testing.T) {
	m := wrapTestModel(20)
	m.SetText(strings.Repeat("alpha beta gamma delta epsilon zeta\n", 200)[:0])
	para := "alpha beta gamma delta epsilon zeta eta theta\n"
	for i := 0; i < 200; i++ {
		m.Append(para)
		_ = m.View()
	}
	m.Append("tail words here")
	m.Skip()
	_ = m.View()
	before := m.cache.wrapped
	m.Append(" more")
	m.Skip()
	_ = m.View()
	delta := m.cache.wrapped - before
	if delta > len("tail words here more")*3 {
		t.Errorf("wrapped %d bytes after one append, want about the last paragraph only", delta)
	}
}

// Criterion: a width change rewraps the full text once.
func TestWidthChangeRewrapsOnce(t *testing.T) {
	m := wrapTestModel(20)
	m.SetText(strings.Repeat("one two three four five six\n", 50) + "last")
	m.Skip()
	_ = m.View()
	m.Width = 12
	before := m.cache.wrapped
	_ = m.View()
	_ = m.View()
	_ = m.total()
	full := len(m.Text())
	if d := m.cache.wrapped - before; d < full-len("\n")*60 || d > full+30 {
		t.Errorf("wrapped %d bytes over width change, want ~%d (once)", d, full)
	}
}

// Incremental wrap must equal a full ansi.Wrap.
func TestIncrementalWrapMatchesFullWrap(t *testing.T) {
	chunks := []string{"hello wor", "ld this is", " long\n", "\n", "\x1b[31mred text ", "here\x1b[0m\r\n", "wide 世界世界 ok", "\n\n", "end  spaces   "}
	for _, w := range []int{5, 9, 20} {
		m := wrapTestModel(w)
		for _, c := range chunks {
			m.Append(c)
			if got, want := m.display(), ansi.Wrap(m.Text(), w); got != want {
				t.Fatalf("w=%d after %q: got %q want %q", w, c, got, want)
			}
			if m.total() != ansi.Width(ansi.Wrap(m.Text(), w)) {
				t.Fatalf("total mismatch")
			}
		}
	}
}

func TestCopiedModelAppendDoesNotCorrupt(t *testing.T) {
	a := wrapTestModel(0)
	a.Append("abc")
	b := a
	b.Append("X")
	a.Append("Y")
	if a.Text() != "abcY" || b.Text() != "abcX" {
		t.Errorf("a=%q b=%q", a.Text(), b.Text())
	}
}

func BenchmarkStream10kTokens(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := wrapTestModel(80)
		m.Start()
		for j := 0; j < 10000; j++ {
			m.Append("tok ")
			if j%20 == 19 {
				m.Append("\n")
			}
		}
	}
}

// Criterion: allocation grows linearly with text length (Stream10kTokens gate).
func TestStream10kTokensAllocLinear(t *testing.T) {
	run := func(n int) float64 {
		return testing.AllocsPerRun(3, func() {
			m := wrapTestModel(80)
			m.Start()
			for j := 0; j < n; j++ {
				m.Append("tok ")
				if j%20 == 19 {
					m.Append("\n")
				}
			}
		})
	}
	a, b := run(2500), run(10000)
	if b > a*6 {
		t.Errorf("allocs 4x text: %v vs %v, want ~linear", b, a)
	}
	bytesFor := func(n int) uint64 {
		var m0, m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m0)
		m := wrapTestModel(80)
		m.Start()
		for j := 0; j < n; j++ {
			m.Append("tok ")
			if j%20 == 19 {
				m.Append("\n")
			}
		}
		runtime.ReadMemStats(&m1)
		return m1.TotalAlloc - m0.TotalAlloc
	}
	x, y := bytesFor(2500), bytesFor(10000)
	if y > x*8 {
		t.Errorf("bytes 4x text: %d vs %d, want ~linear", y, x)
	}
}

// singleParagraphTokens are LLM-like tokens: no newlines, words split
// across tokens, the odd doubled space.
var singleParagraphTokens = []string{"tok", "en ", "the", " quick ", "brown", "  fox ", "jumps ", "ov", "er ", "a ", "lazy", " dog. "}

// Criterion: appended text yields the same View as the uncached wrap.
func TestIncrementalViewMatchesUncachedWrap(t *testing.T) {
	chunks := []string{"a", " ", "  ", "bb cc", "\n", "  \n", "\n", "\n", "x", "\x1b[31mred ", "wor", "d\x1b[0m", " \t ", "世界 wide", "\r\n", "  ", "tail", "\n  ", "  "}
	for _, w := range []int{1, 3, 5, 9, 20} {
		for _, cursor := range []string{"", "|"} {
			m := wrapTestModel(w)
			m.Cursor = cursor
			for i := 0; i < 60; i++ {
				m.Append(chunks[(i*7+i/3)%len(chunks)])
				m.Append(singleParagraphTokens[i%len(singleParagraphTokens)])
				ww := w - ansi.Width(m.cursor())
				if ww < 1 {
					ww = 1
				}
				want := ansi.Wrap(m.Text(), ww)
				if got := m.display(); got != want {
					t.Fatalf("w=%d step %d: display %q want %q", w, i, got, want)
				}
				if got := m.total(); got != ansi.Width(want) {
					t.Fatalf("w=%d step %d: total %d want %d", w, i, got, ansi.Width(want))
				}
				for _, shown := range []int{0, 1, ansi.Width(want) / 2, ansi.Width(want), ansi.Width(want) + 3} {
					m.shown = shown
					wantView := ansi.Truncate(want, shown)
					if c := m.cursor(); c != "" && shown < ansi.Width(want) {
						wantView += ansi.NewStyle().Foreground(m.Theme.Primary).Render(c)
					}
					if got := m.View(); got != wantView {
						t.Fatalf("w=%d step %d shown=%d: View %q want %q", w, i, shown, got, wantView)
					}
				}
			}
		}
	}
}

// Criterion: one more token on a large single paragraph lays out only the
// appended text plus the open line, and View allocates a small constant.
func TestViewOnLargeSingleParagraphIsIncremental(t *testing.T) {
	m := wrapTestModel(80)
	for m.total() < 40000 {
		for _, tok := range singleParagraphTokens {
			m.Append(tok)
		}
		m.Skip()
		_ = m.View()
	}
	m.Append("more ")
	m.Skip()
	before := m.cache.wrapped
	_ = m.View()
	if d := m.cache.wrapped - before; d > 128 {
		t.Errorf("View after one token examined %d bytes of a %d byte paragraph, want about the open line", d, len(m.Text()))
	}
	empty := wrapTestModel(80)
	base := testing.AllocsPerRun(20, func() { _ = empty.View() })
	if base < 1 {
		base = 1
	}
	i := 0
	got := testing.AllocsPerRun(20, func() {
		m.Append(singleParagraphTokens[i%len(singleParagraphTokens)])
		i++
		m.Skip()
		_ = m.View()
	})
	if got > 10*base+10 {
		t.Errorf("append+View on 40 KB allocs %v, empty View allocs %v", got, base)
	}
}

func singleParagraphRun(n int) {
	m := wrapTestModel(80)
	for j := 0; j < n; j++ {
		m.Append(singleParagraphTokens[j%len(singleParagraphTokens)])
		m.Skip()
		_ = m.View()
	}
}

// BenchmarkStreamSingleParagraph10k streams 10,000 tokens into one
// paragraph (no newline) and calls View after every token, the LLM reply
// case Stream10kTokens misses.
func BenchmarkStreamSingleParagraph10k(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		singleParagraphRun(10000)
	}
}

// Criterion: 4x the tokens with View per token allocates well under the
// 16x a quadratic rewrap would (the open line rebuilds once per line).
func TestStreamSingleParagraphBytesLinear(t *testing.T) {
	bytesFor := func(n int) uint64 {
		var m0, m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m0)
		singleParagraphRun(n)
		runtime.ReadMemStats(&m1)
		return m1.TotalAlloc - m0.TotalAlloc
	}
	x, y := bytesFor(2500), bytesFor(10000)
	// Quadratic would be ~16x. Bytes are not strictly linear: disp is copied
	// once per line when a growing word moves to the next one. The fixed
	// per-call garbage is gone, so the ratio is higher than it once was while
	// the absolute bytes are lower (10000 tokens was ~4.58 MB).
	if float64(y) > 12*float64(x) || y > 3_500_000 {
		t.Errorf("bytes for 10000 tokens %d vs 2500 tokens %d (%.1fx), want at most 12x and 3.5 MB (quadratic is ~16x)", y, x, float64(y)/float64(x))
	}
}

// Criterion #85: streaming one long paragraph with a View per token costs
// at most a quarter of the allocations it did before the layout was made
// allocation-free (36709 allocs per 10000 tokens, M0).
func TestStreamSingleParagraphAllocs(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are not meaningful under -race")
	}
	const m0 = 36709
	got := testing.AllocsPerRun(3, func() { singleParagraphRun(10000) })
	if got > m0/4 {
		t.Errorf("10000 tokens allocate %v, want at most %d (1/4 of M0 %d)", got, m0/4, m0)
	}
}

// Criterion: the allocation count grows linearly with the chunks streamed.
func TestStreamSingleParagraphAllocsLinear(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are not meaningful under -race")
	}
	n := testing.AllocsPerRun(3, func() { singleParagraphRun(5000) })
	n2 := testing.AllocsPerRun(3, func() { singleParagraphRun(10000) })
	if n2 >= 2.5*n {
		t.Errorf("allocs for 10000 chunks %v vs 5000 chunks %v (%.2fx), want under 2.5x", n2, n, n2/n)
	}
}

// Criterion: default SetText/Append drop non-SGR sequences; Raw keeps text.
func TestSanitisesByDefaultAndRaw(t *testing.T) {
	const evil = "a\x1b]52;c;ZXZpbA==\x07b\x1b[2Jc\x1b[31md\x1b[0m"
	m := New()
	m.SetText(evil)
	m.Append(evil)
	want := "abc\x1b[31md\x1b[0m"
	if got := m.Text(); got != want+want {
		t.Errorf("Text() = %q, want %q", got, want+want)
	}
	r := New()
	r.Raw = true
	r.SetText(evil)
	r.Append(evil)
	if got := r.Text(); got != evil+evil {
		t.Errorf("Raw Text() = %q, want unchanged", got)
	}
}
