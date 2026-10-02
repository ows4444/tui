package filepicker

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/fsutil"
	"github.com/ows4444/tui/layout"
)

func manyEntries(n int) Model {
	m := New(".")
	m.entries = make([]fsutil.Entry, n)
	for i := range m.entries {
		m.entries[i] = fsutil.Entry{Name: fmt.Sprintf("file%05d", i)}
	}
	return m
}

// #29: 10,000 entries with Height 20 render 20 entry rows, and the window
// follows the cursor from top to bottom.
func TestViewRendersOnlyHeightRowsOfTenThousandEntries(t *testing.T) {
	m := manyEntries(10000)
	m.Height = 20
	if rows := strings.Split(m.View(), "\n"); len(rows) != 20 {
		t.Fatalf("View has %d rows, want 20", len(rows))
	}
	for i := 0; i < 5000; i++ {
		m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	}
	out := m.View()
	rows := strings.Split(out, "\n")
	if len(rows) != 20 {
		t.Fatalf("after scrolling View has %d rows, want 20", len(rows))
	}
	if !strings.Contains(ansi.StripANSI(out), "> file05000") {
		t.Fatalf("cursor row not in the window:\n%s", ansi.StripANSI(out))
	}
	for i := 0; i < 20000; i++ {
		m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	}
	rows = strings.Split(m.View(), "\n")
	if len(rows) != 20 || !strings.Contains(ansi.StripANSI(rows[19]), "> file09999") {
		t.Fatalf("at the bottom: %d rows, last %q", len(rows), rows[len(rows)-1])
	}
}

// Scrolling up moves the window only once the cursor leaves it.
func TestWindowScrollsOnlyWhenCursorLeavesIt(t *testing.T) {
	m := manyEntries(100)
	m.Height = 5
	for i := 0; i < 7; i++ { // cursor 7: window 3..7
		m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	}
	if s, e := m.window(); s != 3 || e != 8 {
		t.Fatalf("window = [%d,%d), want [3,8)", s, e)
	}
	for i := 0; i < 3; i++ { // cursor 4: still inside 3..7
		m, _ = m.Update(tui.Key{Type: tui.KeyUp})
	}
	if s, _ := m.window(); s != 3 {
		t.Fatalf("window moved to %d while the cursor was inside it", s)
	}
	for i := 0; i < 2; i++ { // cursor 2: above the window
		m, _ = m.Update(tui.Key{Type: tui.KeyUp})
	}
	if s, _ := m.window(); s != 2 {
		t.Fatalf("window start = %d, want 2", s)
	}
}

func TestHeightZeroShowsEveryEntry(t *testing.T) {
	m := manyEntries(50)
	if rows := strings.Split(m.View(), "\n"); len(rows) != 50 {
		t.Fatalf("%d rows, want all 50", len(rows))
	}
}

// LayoutNode windows by index too, without building rows it will not show.
func TestLayoutNodeWindowKeepsCursorRow(t *testing.T) {
	m := manyEntries(10000)
	for i := 0; i < 500; i++ {
		m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	}
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 20, H: 6}))
	if rows := strings.Split(out, "\n"); len(rows) != 6 || !strings.Contains(out, "> file00500") {
		t.Fatalf("got:\n%s", out)
	}
}

// #27/#28 for filepicker: a file name carrying escapes is drawn clean unless
// Raw is set.
func TestFileNamesAreSanitisedUnlessRaw(t *testing.T) {
	for _, name := range []string{"a\x1b]52;c;ZXZpbA==\x07b", "a\x1b[2Jb"} {
		m := New(".")
		m.entries = []fsutil.Entry{{Name: name}}
		for path, out := range map[string]string{
			"View": m.View(), "LayoutNode": m.LayoutNode().Render(layout.Size{W: 40, H: 3}), "Linearize": m.Linearize(),
		} {
			if strings.Contains(out, "\x1b]52") || strings.Contains(out, "\x1b[2J") {
				t.Errorf("%s: escape reached the output: %q", path, out)
			}
		}
		m.Raw = true
		if out := m.View(); !strings.Contains(out, name) {
			t.Errorf("Raw View %q does not carry %q", out, name)
		}
	}
}
