package logview

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/viewport"
)

// Regression: Append re-joined and re-split every line on each call, so n
// appends cost O(n^2): 16,000 appends took about 16 seconds. It must stay
// linear; the bound is generous (the old code needed 16 s, this needs
// milliseconds) so a slow machine does not flake it.
func TestAppendScalesLinearly(t *testing.T) {
	const n = 20000
	m := New(120, 40)
	start := time.Now()
	for i := 0; i < n; i++ {
		m.Append(fmt.Sprintf("2026-09-29T12:00:00Z INFO request %d handled", i))
	}
	if el := time.Since(start); el > 3*time.Second {
		t.Fatalf("%d appends took %v, want well under 3s (quadratic behaviour?)", n, el)
	}
	if !m.Viewport.AtBottom() {
		t.Error("a pinned log drifted off the bottom")
	}
}

// Appending keeps the viewport's content equal to the joined lines, whatever
// mix of single-line and multi-line appends and scrolling came before.
func TestAppendMatchesSetContentOfTheJoinedLines(t *testing.T) {
	appends := []string{"one", "", "two\nthree", "four", "wide line of text that is long", "", "last"}
	m := New(10, 3)
	var all []string
	for i, a := range appends {
		m.Append(a)
		all = append(all, a)
		if i == 3 {
			m.Viewport.GotoTop() // the reader scrolls up part-way through
		}
	}

	want := viewport.New(10, 3)
	want.SetContent(strings.Join(all, "\n"))
	want.GotoTop() // the reader stayed at the top after the scroll above
	if got, w := m.Viewport.View(), want.View(); got != w {
		t.Errorf("view after appends:\n%q\nwant:\n%q", got, w)
	}
	if m.Viewport.ScrollPercent() != want.ScrollPercent() {
		t.Errorf("ScrollPercent = %v, want %v", m.Viewport.ScrollPercent(), want.ScrollPercent())
	}
	m.Viewport.GotoBottom()
	want.GotoBottom()
	if got, w := m.Viewport.View(), want.View(); got != w {
		t.Errorf("bottom view:\n%q\nwant:\n%q", got, w)
	}
	// Horizontal scrolling still sees the widest appended line.
	m.Viewport.LineRight(1000)
	want.LineRight(1000)
	if got, w := m.Viewport.View(), want.View(); got != w {
		t.Errorf("scrolled-right view:\n%q\nwant:\n%q", got, w)
	}
}

// BenchmarkAppend measures one append onto a log that already holds 10,000
// lines: constant time now, about half a millisecond with the old code.
func BenchmarkAppend(b *testing.B) {
	m := New(120, 40)
	for i := 0; i < 10000; i++ {
		m.Append("2026-09-29T12:00:00Z INFO request handled in 12ms path=/api/v1/items")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Append("2026-09-29T12:00:00Z INFO request handled in 12ms path=/api/v1/items")
	}
}
