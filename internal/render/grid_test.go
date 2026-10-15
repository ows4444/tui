package render

import "testing"

type testGrid [][]string

func (g testGrid) Cell(x, y int) (string, uint8, Style) {
	if y < len(g) && x < len(g[y]) {
		return g[y][x], 1, Style{}
	}
	return " ", 1, Style{}
}

// A frame with a Grid draws its cells and never parses Lines: a line the parser
// would reject does not make the frame fall back.
func TestFrameGridIsNotParsed(t *testing.T) {
	c := New()
	g := testGrid{{"a", "b"}, {"c"}}
	out, st, ok := c.Frame(Frame{
		Lines: []string{"\x1b[999 unterminated \x1b"}, Grid: g, GridRows: 2, Max: 2, Width: 4,
		RegionTop: func() string { return "" }, FitLine: func(s string) string { return s },
	})
	if !ok || st.Rows != 2 || len(st.Fallbacks) != 0 {
		t.Fatalf("ok=%v stats=%+v", ok, st)
	}
	if want := "ab"; len(out) == 0 || !containsStr(out, want) {
		t.Fatalf("output %q lacks %q", out, want)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// styledGrid gives each cell the style its row string names: 'a' is bold, 'b'
// italic, 'l' a link, anything else the default.
type styledGrid []string

func (g styledGrid) Cell(x, y int) (string, uint8, Style) {
	if y >= len(g) || x >= len(g[y]) {
		return " ", 1, Style{}
	}
	ch := g[y][x : x+1]
	switch ch {
	case "a":
		return ch, 1, Style{Attrs: 1}
	case "b":
		return ch, 1, Style{Attrs: 2}
	case "l":
		return ch, 1, Style{Link: "https://example.com"}
	}
	return ch, 1, Style{}
}

// The last style is remembered between cells, but a change of style, a link and
// a wide or non-ASCII cell all still resolve to the right style: each row's
// styles read back the same as when every cell is looked up on its own.
func TestGridRowStyleMemoMatchesPerCellLookup(t *testing.T) {
	g := styledGrid{"aaabbabl l", "xxxx aab", "llll"}
	c := New()
	var rows [][]cell
	for y := range g {
		row, ok := c.gridRow(g, y, 10, nil)
		if !ok {
			t.Fatalf("row %d not drawn", y)
		}
		rows = append(rows, row)
	}
	// A second Cells with the memo defeated by resetting it before every cell.
	d := New()
	for y := range g {
		var want []cell
		for x := 0; x < 10; x++ {
			d.gridHave = false
			one, ok := d.gridRow(oneCell{g, x, y}, 0, 1, nil)
			if !ok {
				t.Fatal("per-cell row not drawn")
			}
			if len(one) == 1 {
				want = append(want, one[0])
			} else {
				want = append(want, blankCell)
			}
		}
		for len(want) > 0 && want[len(want)-1] == blankCell {
			want = want[:len(want)-1]
		}
		// Style ids differ between the two tables, so compare through the style.
		if len(rows[y]) != len(want) {
			t.Fatalf("row %d: %d cells, want %d", y, len(rows[y]), len(want))
		}
		for x := range want {
			if rows[y][x].ch != want[x].ch || c.styles[rows[y][x].st] != d.styles[want[x].st] {
				t.Errorf("row %d cell %d: %+v (style %+v), want %+v (style %+v)", y, x,
					rows[y][x], c.styles[rows[y][x].st], want[x], d.styles[want[x].st])
			}
		}
	}
	// After the tables are reset the remembered style is not reused.
	c.resetStyles()
	if c.gridHave {
		t.Error("resetStyles left a remembered style")
	}
}

// oneCell presents cell (x, y) of a styledGrid as the only cell of a grid.
type oneCell struct {
	g    styledGrid
	x, y int
}

func (o oneCell) Cell(int, int) (string, uint8, Style) { return o.g.Cell(o.x, o.y) }
