package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// tile is the box the layouts arrange. It fills whatever size it is given with
// a border and a label that includes that size, so the effect of every setting
// shows as numbers as well as shapes. A selected tile has a double border.
type tile struct {
	label string
	w, h  int // natural size
	col   ansi.Color
	sel   bool
}

func (t tile) Measure(c layout.Constraints) layout.Size {
	return layout.Size{W: max(c.MinW, min(c.MaxW, t.w)), H: max(c.MinH, min(c.MaxH, t.h))}
}

func (t tile) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	b := layout.NormalBorder()
	if t.sel {
		b = layout.DoubleBorder()
	}
	st := ansi.NewStyle().Foreground(t.col)
	if t.sel {
		st = st.Bold()
	}
	rows := make([]string, s.H)
	for y := range rows {
		var line string
		switch {
		case s.W == 1:
			line = b.Left
		case y == 0 && s.H > 1:
			line = b.TopLeft + strings.Repeat(b.Top, s.W-2) + b.TopRight
		case y == s.H-1 && s.H > 1:
			line = b.BottomLeft + strings.Repeat(b.Bottom, s.W-2) + b.BottomRight
		default:
			line = b.Left + strings.Repeat(" ", s.W-2) + b.Right
		}
		rows[y] = line
	}
	if s.H >= 3 && s.W >= 4 {
		text := ansi.Truncate(fmt.Sprintf("%s %dx%d", t.label, s.W, s.H), s.W-2)
		pad := s.W - 2 - ansi.Width(text)
		mid := s.H / 2
		rows[mid] = b.Left + strings.Repeat(" ", pad/2) + text + strings.Repeat(" ", pad-pad/2) + b.Right
	}
	for i := range rows {
		rows[i] = st.Render(rows[i])
	}
	return strings.Join(rows, "\n")
}
