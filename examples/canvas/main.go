// Command canvas is a custom widget that draws straight into the cell grid
// through DrawCells: a colour gradient with a moving marker. See docs/rendering.md.
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/cellbuf"
)

// canvas is a widget with both contracts: View (the primary one, used when the
// direct path is unavailable) and DrawCells (the fast path).
type canvas struct {
	x, y          int // marker position
	width, height int
}

func (c canvas) Init() tui.Cmd { return nil }

func (c canvas) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		c.width, c.height = msg.Width, msg.Height
	case tui.Key:
		switch msg.Type {
		case tui.KeyCtrlC:
			return c, tui.Quit()
		case tui.KeyLeft:
			c.x--
		case tui.KeyRight:
			c.x++
		case tui.KeyUp:
			c.y--
		case tui.KeyDown:
			c.y++
		case tui.KeyRunes:
			if msg.Text == "q" {
				return c, tui.Quit()
			}
		}
	}
	c.x = clamp(c.x, c.width-1)
	c.y = clamp(c.y, c.height-2)
	return c, nil
}

func clamp(v, hi int) int { return max(0, min(v, hi)) }

// DrawCells paints one cell per position: a gradient background and the marker.
// It draws only inside r and does not keep buf.
func (c canvas) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	buf = buf.Sub(r) // clip to r; coordinates now start at r's corner
	w, h := buf.Width(), buf.Height()
	for y := 0; y < h-1; y++ { // last row is the help line
		for x := 0; x < w; x++ {
			bg := cellbuf.RGB(uint8(x*255/max(w, 1)), uint8(y*255/max(h, 1)), 128) // #nosec G115 -- x < w and y < h, so each is at most 255
			id := buf.StyleID(cellbuf.Style{BG: bg})
			buf.SetString(x, y, " ", id)
		}
	}
	mark := buf.StyleID(cellbuf.Style{Attrs: cellbuf.AttrBold, FG: cellbuf.RGB(255, 255, 255), BG: cellbuf.RGB(0, 0, 0)})
	buf.SetString(c.x, c.y, "@", mark)
	buf.SetString(0, h-1, "arrows move, q quits", buf.StyleID(cellbuf.Style{Attrs: cellbuf.AttrFaint}))
}

// View shows the same screen as plain text; the Program uses it when the
// direct path is unavailable (accessible mode, a low colour profile, ...).
func (c canvas) View() string {
	buf := cellbuf.New(max(c.width, 1), max(c.height, 1))
	c.DrawCells(buf, buf.Bounds())
	return buf.String()
}

func main() {
	if _, err := tui.NewProgram(canvas{}).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
