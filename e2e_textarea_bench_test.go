package tui_test

import (
	"os"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/textarea"
)

// taRoot is a Program root that draws a textarea straight into the cell grid.
type taRoot struct{ ta textarea.Model }

func (m taRoot) Init() tui.Cmd { return nil }
func (m taRoot) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}
func (m taRoot) View() string { return m.ta.View() }
func (m taRoot) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	sub := buf.Sub(r)
	m.ta.DrawCells(sub, sub.Bounds())
}

// BenchmarkE2EKeystrokeCellTextarea is one keystroke through a Program whose
// root is a focused textarea drawn by DrawCells: Update, the direct cell draw
// and the diffed write. Criterion #48 budgets it at 10 allocs/op.
func BenchmarkE2EKeystrokeCellTextarea(b *testing.B) {
	out, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	defer out.Close()
	ta := textarea.New()
	ta.SetValue(strings.Repeat("the quick brown fox jumps over the lazy dog\n", 8))
	ta.Focus()
	// Pinned like benchKeystroke's, so CI measures what a desktop does.
	p := tui.NewBenchProgram(taRoot{ta}, 80, 24, tui.WithOutput(out), tui.WithCellRenderer(true), tui.WithColorProfile(ansi.TrueColor))
	// Typing and Backspace alternate so the buffer stays the same size however
	// long the benchmark runs.
	keys := [2]tui.Key{{Type: tui.KeyRunes, Text: "x", Code: 'x'}, {Type: tui.KeyBackspace}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.BenchStep(keys[i&1])
	}
}
