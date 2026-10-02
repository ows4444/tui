package logview

import (
	"strconv"
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

func appendN(m *Model, n int) {
	for i := 0; i < n; i++ {
		m.Append("line" + strconv.Itoa(i))
	}
}

// Criterion #383: append while at bottom stays pinned to the bottom.
func TestAppendAtBottomStaysPinnedToBottom(t *testing.T) {
	m := New(10, 3)
	appendN(&m, 5) // fills past the 3-row window; still at bottom after each append

	if !m.Viewport.AtBottom() {
		t.Fatalf("expected viewport to be at bottom before final append")
	}

	m.Append("newest")

	if !m.Viewport.AtBottom() {
		t.Errorf("expected viewport to stay pinned to bottom after Append while at bottom")
	}
	got := m.Viewport.View()
	want := "line3\nline4\nnewest"
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

// Criterion #384: append while scrolled up leaves the scroll position
// unchanged, preserving the reader's position.
func TestAppendWhileScrolledUpPreservesPosition(t *testing.T) {
	m := New(10, 3)
	appendN(&m, 5) // "line0".."line4"

	m.Viewport.GotoTop()
	if got := m.Viewport.View(); got != "line0\nline1\nline2" {
		t.Fatalf("setup: View() = %q", got)
	}

	m.Append("line5")

	if m.Viewport.AtBottom() {
		t.Errorf("expected viewport to remain scrolled up after Append, not jump to bottom")
	}
	got := m.Viewport.View()
	want := "line0\nline1\nline2"
	if got != want {
		t.Errorf("View() after Append while scrolled up = %q, want %q (reading position should be preserved)", got, want)
	}
}

// Criterion #385: Update and View delegate to the embedded viewport.Model.
func TestUpdateAndViewDelegateToViewport(t *testing.T) {
	m := New(10, 3)
	appendN(&m, 10)

	// Model is at bottom; a KeyUp should scroll it up by one line, exactly
	// as viewport.Model.Update would.
	before := m.Viewport.View()

	m, _ = m.Update(tui.Key{Type: tui.KeyUp})

	after := m.Viewport.View()
	if after == before {
		t.Errorf("Update() did not delegate to viewport.Model.Update: view unchanged")
	}

	wantView := m.Viewport.View()
	if got := m.View(); got != wantView {
		t.Errorf("View() = %q, want delegated viewport View() %q", got, wantView)
	}
}

// Criterion #386: lines are accumulated in append order.
func TestAppendPreservesOrder(t *testing.T) {
	m := New(10, 3)
	appendN(&m, 6) // "line0".."line5"

	if got := m.Viewport.LineCount(); got != 6 {
		t.Fatalf("LineCount = %d, want 6", got)
	}

	// Also verify readable back through the viewport in order, scrolling
	// from the top.
	m.Viewport.GotoTop()
	gotView := m.Viewport.View() // first Height(=3) lines from top
	wantView := "line0\nline1\nline2"
	if gotView != wantView {
		t.Errorf("View() from top = %q, want %q", gotView, wantView)
	}

	m.Viewport.GotoBottom()
	gotView = m.Viewport.View()
	wantView = "line3\nline4\nline5"
	if gotView != wantView {
		t.Errorf("View() from bottom = %q, want %q", gotView, wantView)
	}
}

// Criterion: log sanitises by default; Viewport.Raw opts out.
func TestAppendSanitisesByDefaultAndRaw(t *testing.T) {
	const evil = "a\x1b]52;c;ZXZpbA==\x07b\x1b[2Jc"
	m := New(40, 3)
	m.Append(evil)
	if got := m.View(); strings.ContainsAny(got, "\x07") || strings.Contains(got, "\x1b") {
		t.Errorf("View() = %q, want no escapes", got)
	}
	r := New(40, 3)
	r.Viewport.Raw = true
	r.Append(evil)
	if got := r.View(); !strings.Contains(got, evil) {
		t.Errorf("Raw View() = %q, want unchanged", got)
	}
}

// Criterion #93: scrolled up + append keeps the view, shows a count; End resumes.
func TestScrolledUpShowsNewLinesIndicatorAndEndResumes(t *testing.T) {
	m := New(20, 3)
	m.NewLinesIndicator = true
	appendN(&m, 10)
	m.Viewport.GotoTop()
	before := strings.Split(m.View(), "\n")
	m.Append("a")
	m.Append("b")
	if m.NewLines() != 2 {
		t.Fatalf("NewLines = %d, want 2", m.NewLines())
	}
	rows := strings.Split(m.View(), "\n")
	if rows[0] != before[0] || rows[1] != before[1] {
		t.Errorf("view moved: %q", rows)
	}
	if !strings.Contains(rows[2], "2 new lines") {
		t.Errorf("no indicator: %q", rows[2])
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyEnd})
	if m.NewLines() != 0 || strings.Contains(m.View(), "new line") {
		t.Errorf("indicator remains after End: %q", m.View())
	}
	m.Append("c")
	if !strings.Contains(m.View(), "c") || m.NewLines() != 0 {
		t.Errorf("not following after End: %q", m.View())
	}
}
