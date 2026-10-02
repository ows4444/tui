package datatable

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func testRows() [][]string {
	return [][]string{
		{"Alice", "30"},
		{"Bob", "5"},
	}
}

func TestNewStartsAtZero(t *testing.T) {
	m := New([]string{"Name", "Age"}, testRows())
	if m.Cursor() != 0 {
		t.Errorf("Cursor() = %d, want 0", m.Cursor())
	}
}

func TestCursorMovementAndClamp(t *testing.T) {
	m := New([]string{"Name", "Age"}, testRows())

	next, cmd := m.Update(key(tui.KeyDown))
	m = next
	if m.Cursor() != 1 || cmd != nil {
		t.Fatalf("after Down: Cursor()=%d cmd=%v, want 1, nil", m.Cursor(), cmd)
	}
	next, _ = m.Update(key(tui.KeyDown)) // one past the end
	m = next
	if m.Cursor() != 1 {
		t.Fatalf("Down past the end: Cursor() = %d, want 1 (clamped)", m.Cursor())
	}
	next, _ = m.Update(key(tui.KeyUp))
	m = next
	if m.Cursor() != 0 {
		t.Fatalf("after Up: Cursor() = %d, want 0", m.Cursor())
	}
}

func TestEnterEmitsSelectedMsg(t *testing.T) {
	m := New([]string{"Name", "Age"}, testRows())
	m.SetCursor(1)
	_, cmd := m.Update(key(tui.KeyEnter))
	if cmd == nil {
		t.Fatal("Enter should return a non-nil Cmd")
	}
	msg, ok := cmd().(SelectedMsg)
	if !ok {
		t.Fatalf("Cmd produced %T, want SelectedMsg", cmd())
	}
	if msg.Row != 1 || msg.Cells[0] != "Bob" {
		t.Errorf("SelectedMsg = %+v, want Row=1, Cells[0]=Bob", msg)
	}
}

func TestSetCursorClamps(t *testing.T) {
	m := New([]string{"A"}, testRows())
	m.SetCursor(100)
	if m.Cursor() != 1 {
		t.Errorf("SetCursor(100) = %d, want 1", m.Cursor())
	}
	m.SetCursor(-5)
	if m.Cursor() != 0 {
		t.Errorf("SetCursor(-5) = %d, want 0", m.Cursor())
	}
}

func TestEmptyModelDoesNotPanic(t *testing.T) {
	m := New([]string{"A"}, nil)
	m.SetCursor(3)
	if m.Cursor() != 0 {
		t.Errorf("SetCursor on empty rows = %d, want 0", m.Cursor())
	}
	next, cmd := m.Update(key(tui.KeyEnter))
	if cmd != nil || next.Cursor() != 0 {
		t.Error("Update with no rows should be a safe no-op")
	}
}

func TestViewEmptyHeaders(t *testing.T) {
	m := New(nil, testRows())
	if got := m.View(); got != "" {
		t.Errorf("View() with no headers = %q, want empty", got)
	}
}

func TestNonKeyMsgIgnored(t *testing.T) {
	m := New([]string{"A"}, testRows())
	next, cmd := m.Update(tui.ResizeMsg{Width: 10, Height: 10})
	if cmd != nil || next.Cursor() != m.Cursor() {
		t.Error("a non-Key Msg should be a no-op")
	}
}

// TestViewUsesSharedTableRows proves acceptance criterion for
// datatable's half: View's output is exactly what widgets.TableRows
// produces for the same style-per-row function, not an independent
// reimplementation of column-width computation and cell padding.
func TestViewUsesSharedTableRows(t *testing.T) {
	dt := theme.DarkTheme()
	m := New([]string{"Name", "Age"}, testRows())
	m.Theme = dt
	m.SetCursor(1)

	headerStyle := ansi.NewStyle().Bold().Foreground(dt.Primary)
	dividerStyle := ansi.NewStyle().Foreground(dt.Muted)
	cursorStyle := dt.ResolvedStates().Selected.Bold()
	want := widgets.TableRows(m.Headers, m.Rows, headerStyle, dividerStyle, func(i int) ansi.Style {
		if i == m.Cursor() {
			return cursorStyle
		}
		return ansi.Style{}
	})

	if got := m.View(); got != withGutter(want, 0, m.Cursor()) {
		t.Errorf("View() diverged from widgets.TableRows() plus the cursor gutter: View() =\n%q\nTableRows() =\n%q", got, want)
	}
}

func TestViewHighlightsCursorRow(t *testing.T) {
	dt := theme.DarkTheme()
	m := New([]string{"Name", "Age"}, testRows())
	m.Theme = dt
	m.SetCursor(1) // "Bob"

	got := m.View()
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d lines, want 4 (header, divider, 2 rows)", len(lines))
	}

	cursorStyle := dt.ResolvedStates().Selected.Bold()
	// widths: Name col = max(4,5,3) = 5, Age col = max(3,2,1) = 3.
	wantRow1 := cursorStyle.Render("Bob  ") + "  " + cursorStyle.Render("5  ")
	if lines[3] != "> "+wantRow1 {
		t.Errorf("cursor row = %q, want %q", lines[3], wantRow1)
	}
	// The non-cursor row should be unstyled.
	if lines[2] != "  Alice"+"  "+"30 " {
		t.Errorf("non-cursor row = %q, want plain text", lines[2])
	}
}

func tallTable(n, h int) Model {
	m := bigTable(n)
	m.Height = h
	return m
}

func TestCursorScrollsIntoView(t *testing.T) { // #42
	m := tallTable(100, 5)
	for i := 0; i < 50; i++ {
		m, _ = m.Update(key(tui.KeyDown))
	}
	out := m.View()
	if !strings.Contains(out, "row50") || strings.Contains(out, "row00") {
		t.Fatalf("cursor row not in window:\n%s", out)
	}
	if got := len(strings.Split(out, "\n")); got != 5+2 {
		t.Fatalf("lines = %d, want 7", got)
	}
	for i := 0; i < 20; i++ {
		m, _ = m.Update(key(tui.KeyUp))
	}
	if !strings.Contains(m.View(), "row30") {
		t.Fatalf("scroll up lost cursor:\n%s", m.View())
	}
	m.SetCursor(99)
	if !strings.Contains(m.View(), "row99") {
		t.Fatal("SetCursor did not reveal row")
	}
}

func TestPagingKeys(t *testing.T) { // #43
	m := tallTable(100, 5)
	m, _ = m.Update(key(tui.KeyPgDown))
	if m.Cursor() != 5 {
		t.Fatalf("PgDn = %d", m.Cursor())
	}
	m, _ = m.Update(key(tui.KeyPgUp))
	if m.Cursor() != 0 {
		t.Fatalf("PgUp = %d", m.Cursor())
	}
	m, _ = m.Update(key(tui.KeyEnd))
	if m.Cursor() != 99 || !strings.Contains(m.View(), "row99") {
		t.Fatalf("End = %d", m.Cursor())
	}
	m, _ = m.Update(key(tui.KeyHome))
	if m.Cursor() != 0 || !strings.Contains(m.View(), "row00") {
		t.Fatalf("Home = %d", m.Cursor())
	}
	m, _ = m.Update(key(tui.KeyPgUp))
	if m.Cursor() != 0 {
		t.Fatal("PgUp not clamped")
	}
}

func TestHeightZeroRendersAll(t *testing.T) { // #45
	m := bigTable(40)
	out := m.View()
	if !strings.Contains(out, "row00") || !strings.Contains(out, "row39") {
		t.Fatal("height 0 must render all rows")
	}
	if got := len(strings.Split(out, "\n")); got != 42 {
		t.Fatalf("lines = %d", got)
	}
}

func BenchmarkTable10kScroll(b *testing.B) { // #44
	m := tallTable(10000, 24)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m, _ = m.Update(key(tui.KeyDown))
		_ = m.View()
	}
}

func TestViewWorkIsProportionalToHeight(t *testing.T) { // #44
	m := tallTable(10000, 24)
	small := testing.AllocsPerRun(20, func() { _ = m.View() })
	full := bigTable(10000)
	all := testing.AllocsPerRun(3, func() { _ = full.View() })
	if small*20 > all {
		t.Fatalf("windowed allocs %v vs full %v", small, all)
	}
}

func TestHeightZeroWindowsFromResize(t *testing.T) { // #49
	m := bigTable(100)
	m, _ = m.Update(tui.ResizeMsg{Width: 80, Height: 12})
	out := m.View()
	if got := len(strings.Split(out, "\n")); got != 12 {
		t.Fatalf("lines = %d, want 12 (10 rows + header + divider)", got)
	}
	for i := 0; i < 50; i++ {
		m, _ = m.Update(key(tui.KeyDown))
	}
	out = m.View()
	if !strings.Contains(out, "row50") || strings.Contains(out, "row00") {
		t.Fatalf("cursor row not visible:\n%s", out)
	}
	if got := len(strings.Split(out, "\n")); got != 12 {
		t.Fatalf("lines = %d, want 12", got)
	}
	// explicit Height wins over the resize.
	m.Height = 3
	if got := len(strings.Split(m.View(), "\n")); got != 5 {
		t.Fatalf("explicit Height lines = %d, want 5", got)
	}
}

func TestResizeTinyKeepsOneRow(t *testing.T) { // #49
	m := bigTable(10)
	m, _ = m.Update(tui.ResizeMsg{Width: 80, Height: 1})
	m.SetCursor(7)
	out := m.View()
	if !strings.Contains(out, "row07") || len(strings.Split(out, "\n")) != 3 {
		t.Fatalf("want one visible cursor row:\n%s", out)
	}
}
