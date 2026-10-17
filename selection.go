package tui

import (
	"strings"
	"unicode"

	"github.com/ows4444/tui/ansi"
)

// SelectionMsg is delivered to Update when the user finishes selecting text
// with the mouse (see WithTextSelection). Text is what was selected, rows joined
// by "\n" with their trailing spaces cut; the Program has already copied it to
// the clipboard with OSC 52.
type SelectionMsg struct{ Text string }

// WithTextSelection lets the user select text anywhere on the screen by
// dragging with Alt held, whatever widget drew it. The selected cells are drawn
// in style (reverse video when style is the zero Style), a release copies the
// text with OSC 52 and delivers a SelectionMsg, and a double click with Alt
// selects a word and a triple click a row. The selection goes away on the next
// key, click or resize. Mouse events that start a selection, and the drag and
// release after it, are consumed by the Program and never reach Update; every
// other mouse event is delivered as usual.
//
// It needs the alternate screen and WithMouse at MouseCellMotion or above.
func WithTextSelection(style ansi.Style) ProgramOption {
	return func(p *Program) {
		p.selOn = true
		p.selStyle = style
		if style == (ansi.Style{}) {
			p.selStyle = ansi.NewStyle().Reverse()
		}
	}
}

// selection is the text selected on screen: from the cell where the button went
// down to the cell where it is now, in reading order like a terminal's.
type selection struct {
	on     bool // a selection is shown
	drag   bool // the button is down
	moved  bool // the pointer left the cell the button went down on
	ar, ac int  // where it began: row and column
	cr, cc int  // where it is now
}

// span returns the ends in reading order.
func (s selection) span() (r0, c0, r1, c1 int) {
	r0, c0, r1, c1 = s.ar, s.ac, s.cr, s.cc
	if r1 < r0 || r1 == r0 && c1 < c0 {
		r0, c0, r1, c1 = r1, c1, r0, c0
	}
	return
}

// clearSelection drops the selection and asks for a frame without it.
func (p *Program) clearSelection() {
	if p.sel.on {
		p.sel = selection{}
		p.selDirty = true
	}
}

// selectMouse handles ev for text selection. handled says the Program took it;
// then msg, when not nil, is what Update gets in its place.
func (p *Program) selectMouse(ev MouseEvent) (msg Msg, handled bool) {
	if !p.altScreen || ev.Button != MouseButtonLeft {
		return nil, false
	}
	switch ev.Action {
	case MouseActionPress:
		if !ev.Mod.Alt() {
			p.clearSelection()
			return nil, false
		}
		p.sel = selection{on: true, drag: true, ar: ev.Y, ac: ev.X, cr: ev.Y, cc: ev.X}
		switch row := p.screenRow(ev.Y); {
		case ev.Clicks >= 3:
			p.sel.ac, p.sel.cc = 0, max(p.Measurer().Width(row)-1, 0)
		case ev.Clicks == 2:
			from, to := wordSpan(ansi.StripANSI(row), ev.X)
			p.sel.ac, p.sel.cc = from, max(to-1, from)
		}
		p.render()
		return nil, true
	case MouseActionMotion:
		if !p.sel.drag {
			return nil, false
		}
		p.sel.cr, p.sel.cc = ev.Y, ev.X
		p.sel.moved = p.sel.moved || ev.Y != p.sel.ar || ev.X != p.sel.ac
		p.render()
		return nil, true
	case MouseActionRelease:
		if !p.sel.drag {
			return nil, false
		}
		p.sel.drag = false
		text := p.selectedText()
		if text == "" || !p.sel.moved && ev.Clicks < 2 { // a single click with Alt held selects nothing
			p.clearSelection()
			p.render()
			return nil, true
		}
		p.copied = text
		p.write(ansi.OSC52Copy(text))
		p.render()
		return SelectionMsg{Text: text}, true
	}
	return nil, false
}

// screenRow is the row y of the frame last drawn, with its styling.
func (p *Program) screenRow(y int) string {
	if y < 0 || y >= len(p.lastFrame) {
		return ""
	}
	return p.lastFrame[y]
}

// selectedText is the text under the selection, read from the last frame.
func (p *Program) selectedText() string {
	r0, c0, r1, c1 := p.sel.span()
	m := p.Measurer()
	var rows []string
	for y := r0; y <= r1; y++ {
		row := p.screenRow(y)
		a, b := 0, m.Width(row)
		if y == r0 {
			a = c0
		}
		if y == r1 {
			b = min(b, c1+1)
		}
		if a >= b {
			rows = append(rows, "")
			continue
		}
		cut := ansi.StripANSI(m.TrimLeftWidth(m.Truncate(row, b), a))
		rows = append(rows, strings.TrimRight(cut, " "))
	}
	return strings.TrimRight(strings.Join(rows, "\n"), "\n")
}

// paintSelection draws the selection over view.
func (p *Program) paintSelection(view string) string {
	if !p.sel.on {
		return view
	}
	r0, c0, r1, c1 := p.sel.span()
	lines := strings.Split(view, "\n")
	m := p.Measurer()
	for y := max(r0, 0); y <= r1 && y < len(lines); y++ {
		a, b := 0, m.Width(lines[y])
		if y == r0 {
			a = c0
		}
		if y == r1 {
			b = min(b, c1+1)
		}
		if a >= b {
			continue
		}
		left := m.Truncate(lines[y], a)
		left += strings.Repeat(" ", max(a-m.Width(left), 0))
		mid := ansi.StripANSI(m.TrimLeftWidth(m.Truncate(lines[y], b), a))
		mid += strings.Repeat(" ", max(b-a-m.Width(mid), 0))
		lines[y] = left + p.selStyle.Render(mid) + m.TrimLeftWidth(lines[y], b)
	}
	return strings.Join(lines, "\n")
}

// wordSpan is the columns [from, to) of the run of the same kind of character
// (word characters, blanks, or punctuation) that covers column col of text.
func wordSpan(text string, col int) (from, to int) {
	kind := func(r rune) int {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
			return 1
		case unicode.IsSpace(r):
			return 0
		}
		return 2
	}
	type cell struct{ from, to, kind int }
	var cells []cell
	x := 0
	for _, r := range text {
		w := ansi.Width(string(r))
		cells = append(cells, cell{x, x + w, kind(r)})
		x += w
	}
	for i, c := range cells {
		if col < c.from || col >= c.to {
			continue
		}
		from, to = c.from, c.to
		for j := i - 1; j >= 0 && cells[j].kind == c.kind; j-- {
			from = cells[j].from
		}
		for j := i + 1; j < len(cells) && cells[j].kind == c.kind; j++ {
			to = cells[j].to
		}
		return from, to
	}
	return col, col + 1
}
