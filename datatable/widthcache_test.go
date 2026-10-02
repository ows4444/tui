package datatable

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
)

func bigModel(n int) Model {
	rows := make([][]string, n)
	for i := range rows {
		rows[i] = []string{fmt.Sprint("row", i), "x"}
	}
	return New([]string{"Name", "V"}, rows)
}

// countScans returns the total rows scanned by natural-width passes during f.
func countScans(f func()) (rows, passes int) {
	scanHook = func(n int) { rows += n; passes++ }
	defer func() { scanHook = nil }()
	f()
	return
}

func draw(m Model) string {
	n := m.LayoutNode()
	n.Measure(layout.Loose(layout.Size{W: 80, H: 20}))
	return n.Render(layout.Size{W: 80, H: 12})
}

// #52: after the first draw, cursor moves cause no row scan at all.
func TestLayoutNodeWidthsCachedAcrossCursorMoves(t *testing.T) {
	m := bigModel(10000)
	draw(m) // warm
	rows, _ := countScans(func() {
		for _, c := range []int{1, 500, 9999, 0} {
			m.SetCursor(c)
			draw(m)
		}
	})
	if rows != 0 {
		t.Fatalf("scanned %d rows after cursor moves, want 0", rows)
	}
}

// #52: the cache survives Update, which returns Model by value.
func TestLayoutNodeWidthsCachedAcrossUpdate(t *testing.T) {
	m := bigModel(10000)
	draw(m)
	rows, _ := countScans(func() {
		m2, _ := m.Update(tui.Key{Type: tui.KeyDown})
		draw(m2)
	})
	if rows != 0 {
		t.Fatalf("scanned %d rows, want 0", rows)
	}
}

func TestWidthCacheInvalidatedBySetRowsAndRowsChange(t *testing.T) {
	m := bigModel(100)
	draw(m)
	rows := make([][]string, 50)
	for i := range rows {
		rows[i] = []string{"a-very-long-name-here", "y"}
	}
	m.SetRows(rows)
	if n, p := countScans(func() { draw(m) }); p == 0 || n != 50 {
		t.Fatalf("after SetRows: scanned %d rows in %d passes, want 50 rows", n, p)
	}
	if w := (tableNode{m}).naturalWidths(); w[0] != len("a-very-long-name-here") {
		t.Fatalf("width not refreshed: %v", w)
	}
	// Plain assignment of a new Rows slice is also detected.
	m.Rows = append([][]string(nil), rows[:10]...)
	if n, _ := countScans(func() { draw(m) }); n != 10 {
		t.Fatalf("after Rows change: scanned %d, want 10", n)
	}
	// Appending changes the length.
	m.Rows = append(m.Rows, []string{"zz", "z"})
	if n, _ := countScans(func() { draw(m) }); n != 11 {
		t.Fatalf("after append: scanned %d, want 11", n)
	}
}

func TestWidthCacheSurvivesSort(t *testing.T) {
	m := bigModel(1000)
	draw(m)
	m.sortBy(0)
	if n, _ := countScans(func() { draw(m) }); n != 0 {
		t.Fatalf("scanned %d rows after sort, want 0", n)
	}
}

// #52: cost of a post-cursor-move draw is independent of row count.
func BenchmarkLayoutNodeCursorMove(b *testing.B) {
	for _, n := range []int{100, 10000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			m := bigModel(n)
			draw(m)
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				m.SetCursor(i % n)
				draw(m)
			}
		})
	}
}

// #53: fixed widths hold while scrolling and need no row scan.
func TestColumnWidthsConstantWhileScrolling(t *testing.T) {
	m := bigModel(10000)
	m.ColumnWidths = []int{12, 4}
	rows, _ := countScans(func() {
		for _, c := range []int{0, 3000, 9999} {
			m.SetCursor(c)
			if w := (tableNode{m}).naturalWidths(); w[0] != 12 || w[1] != 4 {
				t.Fatalf("cursor %d: widths %v", c, w)
			}
			sz := m.LayoutNode().Measure(layout.Loose(layout.Size{W: 80, H: 20}))
			if sz.W != cursorGutter+12+4+gap {
				t.Fatalf("cursor %d: measured width %d", c, sz.W)
			}
			draw(m)
		}
	})
	if rows != 0 {
		t.Fatalf("scanned %d rows with all widths fixed", rows)
	}
}

func TestColumnWidthsPartialAndClip(t *testing.T) {
	m := bigModel(5)
	m.ColumnWidths = []int{0, 3} // col 0 natural, col 1 fixed
	w := (tableNode{m}).naturalWidths()
	if w[0] != len("row4") || w[1] != 3 {
		t.Fatalf("widths %v", w)
	}
	m.ColumnWidths = []int{3}
	if out := draw(m); !strings.Contains(out, "ro…") {
		t.Fatalf("expected clipped cell in:\n%s", out)
	}
}
