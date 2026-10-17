package form

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/internal/celldraw"
)

// DrawCells draws the form into region r of buf, showing what View shows (it
// implements tui.CellDrawer). The text and secret fields, which are one row,
// draw straight into the grid through their own DrawCells; descriptions and
// error lines are written as styled text; every other kind of field, and the
// group header, is drawn from its View string. A description or an error that
// holds control characters makes the whole form draw through View instead.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	sub := buf.Sub(r)
	if sub.Width() == 0 || sub.Height() == 0 {
		return
	}
	faint, ok1 := celldraw.StyleID(sub, ansi.NewStyle().Faint())
	errID, ok2 := celldraw.StyleID(sub, ansi.NewStyle().Foreground(m.themed().Error))
	if !ok1 || !ok2 || !m.plainText() {
		tui.DrawView(buf, r, m.View())
		return
	}
	y := 0
	for _, h := range m.headerRows() {
		if y >= sub.Height() {
			return
		}
		tui.DrawView(sub, cellbuf.Rect{X: 0, Y: y, W: sub.Width(), H: 1}, h)
		y++
	}
	lo, hi := m.span()
	for i := lo; i < hi && y < sub.Height(); i++ {
		in := m.inputs[i]
		rows := 1
		switch {
		case in.kind == FieldText && in.isPass:
			in.secret.DrawCells(sub, cellbuf.Rect{X: 0, Y: y, W: sub.Width(), H: 1})
		case in.kind == FieldText:
			in.text.DrawCells(sub, cellbuf.Rect{X: 0, Y: y, W: sub.Width(), H: 1})
		default:
			v := in.view(m.themed())
			rows = strings.Count(v, "\n") + 1
			tui.DrawView(sub, cellbuf.Rect{X: 0, Y: y, W: sub.Width(), H: rows}, v)
		}
		y += rows
		if in.desc != "" && y < sub.Height() {
			sub.SetString(2, y, ansi.Sanitize(in.desc), faint)
			y++
		}
		if m.errs[i] != "" && y < sub.Height() {
			sub.SetString(2, y, m.errs[i], errID)
			y++
		}
	}
}

// plainText reports whether every description and error line is free of
// control characters, so SetString draws what View would.
func (m Model) plainText() bool {
	lo, hi := m.span()
	for i := lo; i < hi; i++ {
		if !celldraw.Plain(m.inputs[i].desc, m.errs[i]) {
			return false
		}
	}
	return true
}

var _ tui.CellDrawer = Model{}
