package textarea

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

// referenceView is the View algorithm as it was before Height existed, kept
// verbatim so Height == 0 can be checked against it.
func referenceView(m Model) string {
	if m.value.len() == 0 {
		var b strings.Builder
		if m.focused && m.cursorVisible {
			b.WriteString(m.CursorStyle.Render(" "))
		}
		b.WriteString(m.PlaceholderStyle.Render(m.Placeholder))
		return b.String()
	}

	cursorLine, _ := m.lineCol()
	lines := strings.Split(m.value.String(), "\n")
	lineStart := make([]int, len(lines))
	offset := 0
	for i, l := range lines {
		lineStart[i] = offset
		offset += len([]rune(l)) + 1
	}

	rendered := make([]string, len(lines))
	for i, l := range lines {
		lr := []rune(l)
		lineCursor := -1
		if i == cursorLine {
			lineCursor = m.cursor - lineStart[i]
		}
		start, end := 0, len(lr)
		if m.Width > 0 && len(lr) > m.Width {
			pos := 0
			if lineCursor >= 0 {
				pos = lineCursor
			}
			start, end = visibleWindow(len(lr), pos, m.Width)
		}
		var b strings.Builder
		for j := start; j < end; j++ {
			if m.focused && m.cursorVisible && j == lineCursor {
				b.WriteString(m.CursorStyle.Render(string(lr[j])))
				continue
			}
			b.WriteString(m.TextStyle.Render(string(lr[j])))
		}
		if m.focused && m.cursorVisible && lineCursor == len(lr) && lineCursor == end {
			b.WriteString(m.CursorStyle.Render(" "))
		}
		rendered[i] = b.String()
	}
	return strings.Join(rendered, "\n")
}

func numbered(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("L%d", i)
	}
	return strings.Join(lines, "\n")
}

// visibleLines returns the view's lines with styling removed.
func visibleLines(m Model) []string {
	return strings.Split(ansi.StripANSI(m.View()), "\n")
}

func firstVisible(t *testing.T, m Model) int {
	t.Helper()
	var n int
	if _, err := fmt.Sscanf(visibleLines(m)[0], "L%d", &n); err != nil {
		t.Fatalf("first visible line %q: %v", visibleLines(m)[0], err)
	}
	return n
}

func pressKey(m Model, typ tui.KeyType, times int) Model {
	for i := 0; i < times; i++ {
		m, _ = m.Update(tui.Key{Type: typ})
	}
	return m
}

// Height 0 must render exactly what View rendered before Height existed.
func TestHeightZeroMatchesTheUnwindowedView(t *testing.T) {
	for _, width := range []int{0, 6} {
		for _, focus := range []bool{false, true} {
			for _, cursor := range []int{0, 1, 7, 20, 60, 1 << 20} {
				m := New()
				m.Width = width
				m.SetValue(numbered(30) + "\nsome much longer line here\n\nend")
				if focus {
					m.Focus()
				}
				m.SetCursor(cursor)
				if got, want := m.View(), referenceView(m); got != want {
					t.Fatalf("width=%d focus=%v cursor=%d:\n got %q\nwant %q", width, focus, cursor, got, want)
				}
			}
		}
	}
}

// With Height N the view is at most N lines and always includes the cursor
// line, wherever the cursor is.
func TestHeightLimitsLinesAndKeepsTheCursorLineVisible(t *testing.T) {
	const total = 40
	value := numbered(total)
	for _, height := range []int{1, 2, 5, 39, 40, 100} {
		for line := 0; line < total; line++ {
			m := New()
			m.Height = height
			m.SetValue(value)
			m.SetCursor(strings.Index(value, fmt.Sprintf("L%d", line)))
			lines := visibleLines(m)
			if len(lines) > height {
				t.Fatalf("height=%d cursor line=%d: %d lines, want at most %d", height, line, len(lines), height)
			}
			want := fmt.Sprintf("L%d", line)
			found := false
			for _, l := range lines {
				if l == want {
					found = true
				}
			}
			if !found {
				t.Fatalf("height=%d: cursor line %s not in view %q", height, want, lines)
			}
		}
	}
}

// Moving the cursor off the window scrolls it by the least distance; moving
// within it does not scroll.
func TestWindowScrollsTheMinimumDistance(t *testing.T) {
	m := focused()
	m.Height = 5
	m.SetValue(numbered(30)) // cursor on the last line
	if got := firstVisible(t, m); got != 25 {
		t.Fatalf("after SetValue the window starts at L%d, want L25", got)
	}
	m = pressKey(m, tui.KeyUp, 4) // cursor on L25, still the first visible line
	if got := firstVisible(t, m); got != 25 {
		t.Errorf("cursor inside the window moved it to L%d, want L25", got)
	}
	m = pressKey(m, tui.KeyUp, 1) // L24 is just above the window
	if got := firstVisible(t, m); got != 24 {
		t.Errorf("one line above: window starts at L%d, want L24", got)
	}
	m = pressKey(m, tui.KeyUp, 1)
	if got := firstVisible(t, m); got != 23 {
		t.Errorf("two lines above: window starts at L%d, want L23", got)
	}
	m = pressKey(m, tui.KeyDown, 4) // cursor on L27, the last visible line (window L23..L27)
	if got := firstVisible(t, m); got != 23 {
		t.Errorf("cursor on the last visible line moved the window to L%d, want L23", got)
	}
	m = pressKey(m, tui.KeyDown, 1) // L28 is just below the window
	if got := firstVisible(t, m); got != 24 {
		t.Errorf("one line below: window starts at L%d, want L24", got)
	}
}

func TestSetCursorScrollsTheWindow(t *testing.T) {
	m := New()
	m.Height = 4
	m.SetValue(numbered(20))
	m.SetCursor(0)
	if got := firstVisible(t, m); got != 0 {
		t.Errorf("after SetCursor(0) the window starts at L%d, want L0", got)
	}
}

func TestTypingAtTheBottomKeepsTheNewLineVisible(t *testing.T) {
	m := focused()
	m.Height = 3
	for i := 0; i < 10; i++ {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "x"})
		m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	}
	if lines := visibleLines(m); len(lines) != 3 {
		t.Fatalf("got %d lines %q, want 3", len(lines), lines)
	}
	if got := ansi.StripANSI(m.View()); !strings.HasSuffix(got, "\n ") {
		t.Errorf("last visible line should be the empty cursor line, got %q", got)
	}
}

func TestEmptyValueWithHeightStillShowsThePlaceholder(t *testing.T) {
	m := New()
	m.Placeholder = "type here"
	m.Height = 3
	if got, want := m.View(), referenceView(m); got != want {
		t.Errorf("View = %q, want %q", got, want)
	}
}

func TestShrinkingHeightKeepsTheCursorVisible(t *testing.T) {
	m := New()
	m.Height = 20
	m.SetValue(numbered(30))
	m.Height = 3
	lines := visibleLines(m)
	if len(lines) != 3 || lines[len(lines)-1] != "L29" {
		t.Errorf("view = %q, want 3 lines ending at L29", lines)
	}
}

// The cached line index must equal a fresh scan after every kind of edit.
func TestCachedLineIndexMatchesAFreshScan(t *testing.T) {
	keys := []tui.Key{
		{Type: tui.KeyRunes, Text: "a"},
		{Type: tui.KeyRunes, Text: "éx"},
		{Type: tui.KeySpace},
		{Type: tui.KeyEnter},
		{Type: tui.KeyEnter},
		{Type: tui.KeyBackspace},
		{Type: tui.KeyBackspace},
		{Type: tui.KeyDelete},
		{Type: tui.KeyLeft},
		{Type: tui.KeyRight},
		{Type: tui.KeyUp},
		{Type: tui.KeyDown},
		{Type: tui.KeyHome},
		{Type: tui.KeyEnd},
		{Type: tui.KeyCtrl, Code: 'u'},
		{Type: tui.KeyCtrl, Code: 'k'},
		{Type: tui.KeyCtrl, Code: 'w'},
	}
	seed := uint32(1)
	next := func(n int) int { // small deterministic generator
		seed = seed*1664525 + 1013904223
		return int(seed>>8) % n
	}
	m := focused()
	m.Height = 4
	m.SetValue(numbered(12))
	check := func(step string) {
		t.Helper()
		want := flatStarts(m.value.String())
		if got := docStarts(m.value); !slices.Equal(got, want) {
			t.Fatalf("%s: index %v, fresh %v", step, got, want)
		}
	}
	check("SetValue")
	for i := 0; i < 2000; i++ {
		switch next(20) {
		case 0:
			m, _ = m.Update(tui.PasteEvent{Text: "p\nq\r\nr"})
		case 1:
			m.SetCursor(next(m.value.len() + 1))
		default:
			m, _ = m.Update(keys[next(len(keys))])
		}
		check(fmt.Sprintf("step %d", i))
	}
	m.Reset()
	if got := docStarts(m.value); !slices.Equal(got, []int{0}) {
		t.Errorf("Reset left an index %v", got)
	}
	m.Height = 0
	m.SetValue(numbered(5))
	if got, want := docStarts(m.value), flatStarts(m.Value()); !slices.Equal(got, want) {
		t.Errorf("Height 0: index %v, want %v", got, want)
	}
}

// View over a huge buffer must not allocate per line of the buffer.
func TestWindowedViewAllocationsDoNotScaleWithTheBuffer(t *testing.T) {
	m := New()
	m.Height = 40
	m.SetValue(strings.Repeat("the quick brown fox jumps over the lazy dog\n", 100000))
	m.Focus()
	if allocs := testing.AllocsPerRun(20, func() { _ = m.View() }); allocs >= 100 {
		t.Errorf("View at 100,000 lines with Height 40 made %.0f allocations, want under 100", allocs)
	}
}
