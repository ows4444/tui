package tui

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/internal/render"
)

// CellDrawer is an optional interface for a Model (or a child composed into
// one) that draws straight into a cell grid instead of building an ANSI string.
// It is the fast path of the cell renderer: when the root Model of a Program
// implements CellDrawer, each frame is drawn by DrawCells into the frame's
// grid and diffed from there, and the View string is neither built nor parsed.
//
// DrawCells draws the part of the frame in r (in buf's coordinates) into buf.
// For the root, buf is the whole frame, as wide and as tall as the terminal, and
// r is buf.Bounds(); trailing rows left blank in the default style are not
// part of an inline frame, as trailing empty lines are not in a View string.
// An implementation should stay inside r (draw through buf.Sub(r) to have it
// clipped). It must not keep buf, which is reused for the next frame, and starts
// each frame blank.
//
// View remains the primary contract (Model requires it). A CellDrawer root
// should still return a View that shows the same screen: the Program uses it
// when the direct path is unavailable, namely in accessible mode, with the
// inspector or an announcement region on, when the colour profile is below
// TrueColor or the terminal lacks styled underlines (those conversions are done
// on strings), when the terminal height is unknown, when an inline frame has
// already scrolled rows into history, and when a cell cannot be drawn by the
// cell renderer (it then falls back as a string frame does).
//
// A CellDrawer root is redrawn on every render: the Program has no View string
// to compare with the last frame, so an unchanged frame is still diffed (and
// writes nothing but the frame's framing bytes).
type CellDrawer interface {
	DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect)
}

// DrawView draws the styled string view, as a View result would be shown, into
// the region r of buf: line i of view is row r.Y+i, clipped to r. It is how a
// CellDrawer composes a child that only has a View string. A line the cell grid
// cannot represent (see cellbuf.ErrUnsupported) is drawn as plain text with its
// escape sequences removed.
func DrawView(buf *cellbuf.Buffer, r cellbuf.Rect, view string) {
	sub := buf.Sub(r)
	for y, line := range strings.Split(view, "\n") {
		if y >= sub.Height() {
			break
		}
		line = strings.TrimSuffix(line, "\r")
		if _, err := sub.SetStyled(0, y, line); err != nil {
			sub.SetString(0, y, line, 0)
		}
	}
}

// DrawChild draws child into the region r of buf: through its DrawCells when it
// implements CellDrawer, else by drawing its View string with DrawView. The
// child's DrawCells sees a buffer clipped to r whose origin is r's corner, and
// the rectangle 0,0,r.W,r.H. It is how a CellDrawer mixes CellDrawer and
// string children, producing the same screen as if all were strings.
func DrawChild(buf *cellbuf.Buffer, r cellbuf.Rect, child Model) {
	if d, ok := child.(CellDrawer); ok {
		sub := buf.Sub(r)
		d.DrawCells(sub, sub.Bounds())
		return
	}
	DrawView(buf, r, child.View())
}

// bufGrid adapts a cellbuf.Buffer to the renderer's GridSource.
type bufGrid struct{ b *cellbuf.Buffer }

func (g bufGrid) Cell(x, y int) (string, uint8, render.Style) {
	c := g.b.At(x, y)
	s := g.b.Style(c.Style)
	return c.Cluster, c.Width, render.Style{
		Attrs: uint16(s.Attrs), FG: uint32(s.FG), BG: uint32(s.BG), UL: s.UnderlineStyle, Link: s.Link,
	}
}

// directDraw reports whether this frame can be drawn from the root's
// DrawCells, and returns it.
func (p *Program) directDraw() (CellDrawer, bool) {
	d, ok := p.model.(CellDrawer)
	if !ok || !p.cellRender || p.accessible || p.inspectorOn || p.outlineOn || p.msgOn || p.modelOn || p.announce.Rows > 0 ||
		p.width <= 0 || p.height <= 0 || p.committed > 0 || p.colorProfile < ansi.TrueColor ||
		p.probeSaysNo(func(c Capabilities) bool { return c.StyledUnderline }) {
		return nil, false
	}
	return d, true
}

// renderDirect builds the diffed frame from the root's DrawCells. ok is false
// when the cell renderer cannot draw it; nothing has been recorded then and the
// caller draws the View string instead.
func (p *Program) renderDirect(d CellDrawer) (string, bool) {
	if p.drawBuf == nil || p.drawBuf.Width() != p.width || p.drawBuf.Height() != p.height {
		p.drawBuf = cellbuf.New(p.width, p.height)
	}
	buf := p.drawBuf
	buf.SetMeasurer(p.Measurer())
	buf.Clear()
	d.DrawCells(buf, buf.Bounds())
	rows := 0
	for y := buf.Height() - 1; y >= 0 && rows == 0; y-- {
		for x := 0; x < buf.Width(); x++ {
			if c := buf.At(x, y); c.Cluster != " " || c.Style != 0 || c.Width != 1 {
				rows = y + 1
				break
			}
		}
	}
	rows = max(rows, 1)
	total := max(rows, len(p.lastFrame))
	if p.cells == nil {
		p.cells = render.New()
	}
	if p.regionTopFn == nil {
		p.regionTopFn, p.fitLineFn = p.toRegionTop, p.fitLine
	}
	out, st, ok := p.cells.Frame(render.Frame{
		Grid: bufGrid{buf}, GridRows: rows, Max: total,
		Width: p.width, Height: p.height, PrevRows: len(p.lastFrame),
		RegionTop: p.regionTopFn, FitLine: p.fitLineFn, Measurer: p.Measurer(),
	})
	if !ok {
		return "", false
	}
	if cap(p.lastFrame) >= total {
		// Only the length matters (every row is blank): reuse the storage.
		p.lastFrame = p.lastFrame[:total]
		clear(p.lastFrame)
	} else {
		p.lastFrame = make([]string, total)
	}
	p.liveLines = total
	p.lastView, p.haveView = "", false
	fs := frameStats{rows: st.Rows, changed: st.Changed, full: st.Full}
	if len(st.Fallbacks) > 0 {
		fs.rowFallbacks = append([]render.RowFallback(nil), st.Fallbacks...)
	}
	p.frameStats = fs
	return out, true
}
