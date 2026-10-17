package virtuallist

import (
	"strconv"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
)

// tall renders item i as h(i) lines "i.0", "i.1", ... where h(i) = 1 + i%3.
func tallHeight(i int) int { return 1 + i%3 }

func tall(n, rows int) Model {
	m := New(n, rows, func(i int) string {
		var ls []string
		for k := 0; k < tallHeight(i); k++ {
			ls = append(ls, strconv.Itoa(i)+"."+strconv.Itoa(k))
		}
		return strings.Join(ls, "\n")
	})
	m.HeightOf = tallHeight
	return m
}

func TestViewIsExactlyHeightRowsAndCutsTheLastItem(t *testing.T) {
	m := tall(100, 5)
	got := strings.Split(m.View(), "\n")
	want := []string{"0.0", "1.0", "1.1", "2.0", "2.1"} // item 2 is 3 rows tall: cut after 2
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("rows %v; want %v", got, want)
	}
}

func TestShortRenderingIsPaddedAndTallOneCut(t *testing.T) {
	m := New(3, 6, func(i int) string {
		if i == 0 {
			return "a" // asked 2 rows, gives 1
		}
		return "b\nc\nd\ne" // asked 2, gives 4
	})
	m.HeightOf = func(int) int { return 2 }
	got := strings.Split(m.View(), "\n")
	if strings.Join(got, ",") != "a,,b,c,b,c" {
		t.Fatalf("%v", got)
	}
}

func TestScrollingStopsWithTheLastItemWhole(t *testing.T) {
	m := tall(10, 6)
	m.SetCursor(9)
	rows := strings.Split(m.View(), "\n")
	if len(rows) != 6 || rows[len(rows)-1] != "9.0" {
		t.Fatalf("last item not at the bottom: %v", rows)
	}
	m.LineDown(50)
	if m.Offset() != m.maxOffsetRows() {
		t.Fatalf("offset %d; max %d", m.Offset(), m.maxOffsetRows())
	}
	// Items 9 (1 row), 8 (3 rows), 7 (2 rows) = 6 rows, so offset 7.
	if m.Offset() != 7 {
		t.Fatalf("offset %d; want 7", m.Offset())
	}
}

func TestCursorFollowShowsTheItemWhole(t *testing.T) {
	m := tall(50, 5)
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	}
	rows := strings.Join(strings.Split(m.View(), "\n"), ",")
	if m.Cursor() != 4 || !strings.Contains(rows, "4.0") || !strings.Contains(rows, "4.1") {
		t.Fatalf("cursor %d rows %s", m.Cursor(), rows)
	}
	m.SetCursor(0)
	if m.Offset() != 0 {
		t.Fatalf("offset %d", m.Offset())
	}
	m.SetCursor(30)
	used := 0
	for i := m.Offset(); i <= 30; i++ {
		used += tallHeight(i)
	}
	if used > 5 {
		t.Fatalf("cursor item not fully visible: offset %d uses %d rows", m.Offset(), used)
	}
}

func TestMousePressMapsRowsToItems(t *testing.T) {
	m := tall(50, 8)
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 20, H: 8}
	// Rows: 0.0 | 1.0 1.1 | 2.0 2.1 2.2 | 3.0 | 4.0
	for row, want := range map[int]int{0: 0, 1: 1, 2: 1, 3: 2, 5: 2, 6: 3, 7: 4} {
		m2, _ := m.Update(tui.MouseEvent{X: 1, Y: row, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
		if m2.Cursor() != want {
			t.Errorf("row %d: cursor %d; want %d", row, m2.Cursor(), want)
		}
	}
	if _, ok := m.itemAtRow(99); ok {
		t.Fatal("a row past the content mapped to an item")
	}
}

func TestOnlyNearbyItemsAreAsked(t *testing.T) {
	calls := 0
	m := New(1_000_000, 10, func(i int) string { return "x" })
	m.HeightOf = func(i int) int { calls++; return 1 + i%2 }
	_ = m.View()
	m.SetCursor(999_999)
	_ = m.View()
	if calls > 200 {
		t.Fatalf("HeightOf was called %d times for a million items", calls)
	}
}

func TestVariableHeightsDrawLikeViewAndFillLayout(t *testing.T) {
	m := tall(40, 7)
	m.SetCursor(12)
	want, err := cellbuf.Parse(m.View())
	if err != nil {
		t.Fatal(err)
	}
	got := cellbuf.New(10, 7)
	m.DrawCells(got, got.Bounds())
	for y := 0; y < 7; y++ {
		for x := 0; x < 5; x++ {
			if a, b := want.At(x, y), got.At(x, y); a.Cluster != b.Cluster {
				t.Fatalf("(%d,%d): View %q DrawCells %q", x, y, a.Cluster, b.Cluster)
			}
		}
	}
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 10, H: 4}))
	if len(strings.Split(out, "\n")) != 4 {
		t.Fatalf("layout node:\n%s", out)
	}
	lin := m.Linearize()
	if !strings.Contains(lin, "Item ") {
		t.Fatalf("linearize %q", lin)
	}
}

func TestAnItemTallerThanTheWindow(t *testing.T) {
	m := New(3, 2, func(i int) string { return "a\nb\nc\nd" })
	m.HeightOf = func(int) int { return 4 }
	if m.maxOffsetRows() != 2 {
		t.Fatalf("max offset %d", m.maxOffsetRows())
	}
	m.SetCursor(1)
	if rows := strings.Split(m.View(), "\n"); len(rows) != 2 || rows[0] != "a" {
		t.Fatalf("%v", rows)
	}
	empty := New(0, 3, func(int) string { return "" })
	empty.HeightOf = func(int) int { return 2 }
	if empty.View() != "" || empty.maxOffsetRows() != 0 {
		t.Fatal("empty list")
	}
}
