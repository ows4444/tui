package tui

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/ows4444/tui/layout"
)

// BenchmarkRenderDiff measures render()'s real per-frame cost: a 24-row
// view where a fifth of the rows (a typical "some state changed" update,
// not a full repaint) differ from the previous frame, so both the
// line-diff comparison and the actual ClearLine/content writes for the
// changed rows are exercised. Output goes to os.DevNull rather than a temp
// file, so the benchmark measures render()'s own cost, not filesystem I/O.
func BenchmarkRenderDiff(b *testing.B) {
	const rows = 24
	frameA := make([]string, rows)
	frameB := make([]string, rows)
	for i := 0; i < rows; i++ {
		frameA[i] = "row " + strconv.Itoa(i) + " steady state content"
		frameB[i] = frameA[i]
	}
	for i := 0; i < rows; i += 5 { // ~20% of rows change each frame
		frameB[i] = "row " + strconv.Itoa(i) + " CHANGED this frame"
	}
	viewA := strings.Join(frameA, "\n")
	viewB := strings.Join(frameB, "\n")

	out, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer out.Close()

	p := NewProgram(staticModel{view: viewA}, WithOutput(out))
	p.width, p.height = 80, rows
	p.render() // seed lastFrame so the loop below measures steady-state diffing

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			p.model = staticModel{view: viewB}
		} else {
			p.model = staticModel{view: viewA}
		}
		p.render()
	}
}

// frameModel draws a layout tree, so a frame includes the View's own cost.
type frameModel struct{ n int }

func (m frameModel) Init() Cmd               { return nil }
func (m frameModel) Update(Msg) (Model, Cmd) { return m, nil }
func (m frameModel) View() string {
	rows := make([]layout.FlexChild, 0, 22)
	for i := 0; i < 22; i++ {
		rows = append(rows, layout.FlexChild{Node: layout.Row(1,
			layout.FlexChild{Node: layout.Block("row " + strconv.Itoa(i)), Basis: 8},
			layout.FlexChild{Node: layout.Block("value " + strconv.Itoa(i+m.n)), Grow: 1},
		)})
	}
	ui := layout.BoxNode(layout.NewBox().Border(layout.RoundedBorder()), layout.Column(0, rows...))
	return layout.Draw(ui, layout.Constraints{MinW: 80, MaxW: 80, MinH: 24, MaxH: 24})
}

func benchProgram(b *testing.B) *Program {
	b.Helper()
	out, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("open %s: %v", os.DevNull, err)
	}
	b.Cleanup(func() { out.Close() })
	p := NewProgram(frameModel{}, WithOutput(out))
	p.width, p.height = 80, 24
	p.render()
	return p
}

// BenchmarkFrame measures one full frame at 80x24: View builds and draws a
// layout tree whose values change, then render() diffs and writes it.
func BenchmarkFrame(b *testing.B) {
	p := benchProgram(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.model = frameModel{n: i}
		p.render()
	}
}

// BenchmarkRepaint measures a full repaint at 80x24, the path a resize
// takes: the previous frame is discarded, so every row is written.
func BenchmarkRepaint(b *testing.B) {
	p := benchProgram(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.repaintFresh()
	}
}
