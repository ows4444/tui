package tui

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/ows4444/tui/internal/render"
)

// keystrokeModel is a styled 24-row screen where one row (the "input line")
// changes per keystroke, the case the spec's CPU-parity criterion names.
type keystrokeModel struct{ n int }

func (m keystrokeModel) Init() Cmd               { return nil }
func (m keystrokeModel) Update(Msg) (Model, Cmd) { return m, nil }
func (m keystrokeModel) View() string {
	rows := make([]string, 24)
	for i := range rows {
		rows[i] = "\x1b[1;34mrow " + strconv.Itoa(i) + "\x1b[0m \x1b[38;2;10;200;30msteady state content \x1b[0m\x1b[3mwith style\x1b[0m and some plain text to fill the line"
	}
	rows[20] = "> " + strings.Repeat("k", m.n%40) + "\x1b[7m \x1b[0m"
	return strings.Join(rows, "\n")
}

func benchKeystroke(b *testing.B, opts ...ProgramOption) {
	out, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	defer out.Close()
	p := NewProgram(keystrokeModel{}, append([]ProgramOption{WithOutput(out)}, opts...)...)
	p.width, p.height = 80, 24
	p.render()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.model = keystrokeModel{n: i + 1}
		p.render()
	}
}

func BenchmarkE2EKeystrokeLine(b *testing.B) { benchKeystroke(b) }
func BenchmarkE2EKeystrokeCell(b *testing.B) { benchKeystroke(b, WithCellRenderer(true)) }

// TestE2EKeystrokeCellParity proves criterion #60: with one row changing per
// frame the cell renderer costs at most 1.2x the line renderer's CPU.
func TestE2EKeystrokeCellParity(t *testing.T) {
	if testing.Short() || raceEnabled {
		t.Skip("timing comparison")
	}
	if os.Getenv("TUI_TIMING_TESTS") == "" {
		t.Skip("wall-clock ratio test; set TUI_TIMING_TESTS=1 to run it")
	}
	best := func(f func(*testing.B)) float64 {
		m := 1e18
		for i := 0; i < 5; i++ {
			if v := float64(testing.Benchmark(f).NsPerOp()); v < m {
				m = v
			}
		}
		return m
	}
	line := best(BenchmarkE2EKeystrokeLine)
	cell := best(BenchmarkE2EKeystrokeCell)
	t.Logf("line %.0f ns/op, cell %.0f ns/op, ratio %.2f", line, cell, cell/line)
	if cell > 1.2*line {
		t.Fatalf("cell renderer %.2fx line (limit 1.2x)", cell/line)
	}
}

// TestCellRowCacheEquivalence: reusing parsed rows must not change a single
// output byte, including when a row is unchanged for several frames, then
// changes, reverts, or the view grows and shrinks.
func TestCellRowCacheEquivalence(t *testing.T) {
	views := []string{
		"a\n\x1b[31mred\x1b[0m\nc",
		"a\n\x1b[31mred\x1b[0m\nc",
		"a\n\x1b[31mred\x1b[0m\nc",
		"a\nblue\nc",
		"a\n\x1b[31mred\x1b[0m\nc",
		"a\n\x1b[31mred\x1b[0m\nc\nd 世界",
		"a\nx\nc\nd 世界",
		"a\nx\nc\nd 世界",
		"a\nx",
		"a\nx\nc",
		"a\n\x1b[31mred\x1b[0m\nc",
	}
	mk := func(noCache bool) (*Program, *os.File) {
		f, err := os.CreateTemp(t.TempDir(), "out")
		if err != nil {
			t.Fatal(err)
		}
		p := NewProgram(staticModel{}, WithOutput(f), WithCellRenderer(true))
		p.width, p.height = 80, 24
		p.cells = render.New()
		p.cells.SetNoCache(noCache)
		return p, f
	}
	pa, fa := mk(false)
	pb, fb := mk(true)
	for i, v := range views {
		pa.model, pb.model = staticModel{view: v}, staticModel{view: v}
		pa.render()
		pb.render()
		a, _ := os.ReadFile(fa.Name())
		b, _ := os.ReadFile(fb.Name())
		if string(a) != string(b) {
			t.Fatalf("frame %d: cached output differs\ncached: %q\nfresh:  %q", i, a, b)
		}
	}
}
