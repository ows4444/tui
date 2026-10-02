// Command mixedscreen composes a string child and a cell child into one screen
// with DrawChild and DrawView. See docs/rendering.md.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

// banner is a plain View-only child: it knows nothing about cells.
type banner struct{ text string }

func (b banner) Init() tui.Cmd                       { return nil }
func (b banner) Update(tui.Msg) (tui.Model, tui.Cmd) { return b, nil }
func (b banner) View() string {
	return ansi.NewStyle().Bold().Foreground(ansi.BrightCyan).Render(b.text)
}

// bars is a CellDrawer child: a bar chart drawn cell by cell.
type bars struct{ values []int }

func (b bars) Init() tui.Cmd                       { return nil }
func (b bars) Update(tui.Msg) (tui.Model, tui.Cmd) { return b, nil }
func (b bars) View() string {
	buf := cellbuf.New(len(b.values)*2, 4)
	b.DrawCells(buf, buf.Bounds())
	return buf.String()
}

func (b bars) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	buf = buf.Sub(r)
	id := buf.StyleID(cellbuf.Style{FG: cellbuf.Bright(2)})
	for i, v := range b.values {
		h := min(v, buf.Height())
		for k := 0; k < h; k++ {
			buf.SetString(i*2, buf.Height()-1-k, "█", id)
		}
	}
}

// screen is the root: a header (string), a chart (cells) and a footer (string).
type screen struct {
	header, footer banner
	chart          bars
	width, height  int
}

func (s screen) Init() tui.Cmd { return nil }

func (s screen) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		s.width, s.height = msg.Width, msg.Height
	case tui.Key:
		if msg.Type == tui.KeyCtrlC || (msg.Type == tui.KeyRunes && msg.Text == "q") {
			return s, tui.Quit()
		}
	}
	return s, nil
}

// DrawCells lays the three children out in rows and draws each with DrawChild,
// which uses a child's DrawCells when it has one and its View otherwise.
func (s screen) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	chartH := max(r.H-2, 0)
	tui.DrawChild(buf, cellbuf.Rect{X: r.X, Y: r.Y, W: r.W, H: 1}, s.header)
	tui.DrawChild(buf, cellbuf.Rect{X: r.X, Y: r.Y + 1, W: r.W, H: chartH}, s.chart)
	tui.DrawChild(buf, cellbuf.Rect{X: r.X, Y: r.Y + 1 + chartH, W: r.W, H: 1}, s.footer)
}

// View is the same screen as a string, for when the direct path is unavailable.
func (s screen) View() string {
	chartH := max(s.height-2, 4)
	chart := strings.Split(s.chart.View(), "\n")
	for len(chart) < chartH {
		chart = append([]string{""}, chart...)
	}
	return s.header.View() + "\n" + strings.Join(chart, "\n") + "\n" + s.footer.View()
}

func newScreen() screen {
	return screen{
		header: banner{"Mixed string and cell screen"},
		footer: banner{"q quits"},
		chart:  bars{values: []int{1, 3, 2, 5, 4, 6, 2}},
	}
}

func main() {
	if _, err := tui.NewProgram(newScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
