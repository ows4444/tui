package datatable

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/layout"
)

func shown(m Model) []string {
	var out []string
	for _, l := range strings.Split(ansi.StripANSI(m.View()), "\n") {
		out = append(out, strings.TrimRight(l, " "))
	}
	return out
}

func sizes() Model {
	m := New([]string{"Name", "Size"}, [][]string{{"a.txt", "5"}, {"big.bin", "12345"}, {"c", "100"}})
	m.Height = 3
	return m
}

func TestColumnAlignEndPadsNumbersRight(t *testing.T) {
	m := sizes()
	m.Columns = []Column{{}, {Align: layout.AlignEnd}}
	got := shown(m)
	// Header row, divider, then rows: the Size column is 5 wide (12345).
	want := []string{"  Name     Size", "  ----     ----"}
	_ = want
	if !strings.HasSuffix(got[0], " Size") || !strings.HasSuffix(got[2], "    5") || !strings.HasSuffix(got[3], "12345") || !strings.HasSuffix(got[4], "  100") {
		t.Fatalf("not right-aligned:\n%s", strings.Join(got, "\n"))
	}
}

func TestColumnCenterAndMin(t *testing.T) {
	m := sizes()
	m.Columns = []Column{{Min: 12, Align: layout.AlignCenter}, {}}
	got := shown(m)
	if !strings.Contains(got[0], "    Name    ") {
		t.Fatalf("header not centred in 12: %q", got[0])
	}
	if !strings.Contains(got[2], "   a.txt    ") {
		t.Fatalf("cell not centred in 12: %q", got[2])
	}
}

func TestColumnMaxCutsWithAnEllipsis(t *testing.T) {
	m := sizes()
	m.Columns = []Column{{Max: 5}, {}}
	got := strings.Join(shown(m), "\n")
	if strings.Contains(got, "big.bin") || !strings.Contains(got, "big.") {
		t.Fatalf("Max 5 not applied:\n%s", got)
	}
	gl := m.themed().GlyphSet().Ellipsis
	if !strings.Contains(got, "big."+gl) {
		t.Fatalf("no ellipsis %q:\n%s", gl, got)
	}
}

func human(cell string) string {
	n, err := strconv.Atoi(cell)
	if err != nil {
		return cell
	}
	return fmt.Sprintf("%.1fK", float64(n)/1000)
}

func TestColumnFormatShowsButDoesNotChangeTheCell(t *testing.T) {
	m := sizes()
	m.Columns = []Column{{}, {Format: human, Align: layout.AlignEnd}}
	got := strings.Join(shown(m), "\n")
	if !strings.Contains(got, "12.3K") || strings.Contains(got, "12345") {
		t.Fatalf("Format not applied:\n%s", got)
	}
	m.SetCursor(1)
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if sel := cmd().(SelectedMsg); sel.Cells[1] != "12345" {
		t.Fatalf("SelectedMsg carries %q; want the cell as given", sel.Cells[1])
	}
	if lin := m.Linearize(); !strings.Contains(lin, "Size: 12.3K") {
		t.Fatalf("Linearize should speak what is shown:\n%s", lin)
	}
}

func TestColumnLessSortsByItsOwnOrder(t *testing.T) {
	m := New([]string{"Level"}, [][]string{{"warn"}, {"error"}, {"info"}, {"debug"}})
	rank := map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}
	m.Columns = []Column{{Less: func(a, b string) bool { return rank[a] < rank[b] }}}
	m.SortBy(0, false)
	var got []string
	for i := 0; i < 4; i++ {
		got = append(got, m.rowAt(i)[0])
	}
	if strings.Join(got, ",") != "debug,info,warn,error" {
		t.Fatalf("sorted %v", got)
	}
	m.SortBy(0, true)
	if m.rowAt(0)[0] != "error" {
		t.Fatalf("descending starts with %q", m.rowAt(0)[0])
	}
}

func TestColumnsDrawCellsAndLayoutNode(t *testing.T) {
	m := sizes()
	m.Columns = []Column{{Max: 5}, {Format: human, Align: layout.AlignEnd, Min: 7}}
	want, err := cellbuf.Parse(m.View())
	if err != nil {
		t.Fatal(err)
	}
	got := cellbuf.New(40, 6)
	m.DrawCells(got, got.Bounds())
	for y := 0; y < 5; y++ {
		for x := 0; x < 30; x++ {
			if a, b := want.At(x, y), got.At(x, y); a.Cluster != b.Cluster {
				t.Fatalf("DrawCells differs from View at (%d,%d): %q vs %q", x, y, a.Cluster, b.Cluster)
			}
		}
	}
	node := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 30, H: 5}))
	lines := strings.Split(node, "\n")
	if !strings.Contains(lines[3], "12.3K") || strings.Contains(node, "12345") {
		t.Fatalf("layout node ignores Format:\n%s", node)
	}
	col := func(l string) int { return ansi.Width(l[:strings.Index(l, "K")]) } // the ellipsis is 3 bytes
	if !strings.Contains(lines[2], "K") || col(lines[2]) != col(lines[3]) {
		t.Fatalf("layout node:\n%s", node)
	}
	if w := m.LayoutNode().Measure(layout.Unconstrained()).W; w < cursorGutter+5+gap+7 {
		t.Fatalf("Measure W = %d ignores Min and Max", w)
	}
}

func TestNoColumnsIsUnchanged(t *testing.T) {
	a, b := sizes(), sizes()
	b.Columns = []Column{{}, {}}
	if a.View() != b.View() || a.hasColumns() || b.hasColumns() {
		t.Fatal("zero Columns changed the table")
	}
}
