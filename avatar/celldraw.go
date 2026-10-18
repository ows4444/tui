package avatar

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/internal/celldraw"
)

// DrawCells draws the avatar into the part of buf in r, the same picture View
// draws, without building the string for the renderer to parse. It makes
// Model a tui.CellDrawer. The picture is Width by Height cells from the corner
// of r and is clipped to r. If the cell grid cannot hold a colour it draws View.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	sub := buf.Sub(r)
	if m.Width <= 0 || m.Height <= 0 || sub.Width() == 0 || sub.Height() == 0 {
		return
	}
	if drawHook != nil {
		drawHook()
	}
	g, p := m.grid()
	colour := [layers]ansi.Color{layerPlate: p.bg, layerBody: p.head, layerEye: p.eye}
	glyphs := m.Theme.GlyphSet()
	ramp := []rune(glyphs.Shades)
	ascii := glyphs.ASCII() && len(ramp) > 0
	// One style per pair of layers, found when first needed; a stretch of cells
	// in the same pair is written as one string.
	var ids [layers][layers]cellbuf.StyleID
	var known [layers][layers]bool
	var run strings.Builder
	for cy := 0; cy < g.h && cy < sub.Height(); cy++ {
		start, ink, paper := 0, layerNone, layerNone
		flush := func(end int) {
			if run.Len() > 0 {
				sub.SetString(start, cy, run.String(), ids[ink][paper])
				run.Reset()
			}
			start = end
		}
		for cx := 0; cx < g.w && cx < sub.Width(); cx++ {
			c := g.cellAt(cx, cy)
			if c.mask == 0 {
				c.ink = layerNone
			}
			if !known[c.ink][c.paper] {
				st := ansi.NewStyle()
				if c.ink != layerNone {
					st = st.Foreground(colour[c.ink])
				}
				if c.paper != layerNone {
					st = st.Background(colour[c.paper])
				}
				id, ok := celldraw.StyleID(sub, st)
				if !ok {
					sub.Clear()
					tui.DrawView(buf, r, m.View())
					return
				}
				ids[c.ink][c.paper], known[c.ink][c.paper] = id, true
			}
			if c.ink != ink || c.paper != paper {
				flush(cx)
				ink, paper = c.ink, c.paper
			}
			run.WriteRune(c.glyph(ramp, ascii))
		}
		flush(0)
	}
}
