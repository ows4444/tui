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
