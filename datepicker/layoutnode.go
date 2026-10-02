package datepicker

import (
	"strings"
	"time"

	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the calendar to a layout.Node. Measure reports its
// natural size (the weekday row plus one row per week of the cursor's
// month). Render fits the allotted Size: the weekday row stays, and when there
// are more weeks than rows it shows a window of weeks containing the cursor's
// week; columns are cut to the width. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return calendarNode{m} }

type calendarNode struct{ m Model }

func (n calendarNode) Measure(c layout.Constraints) layout.Size {
	return layout.Block(n.m.View()).Measure(c)
}

// cursorWeek is the index of the cursor's week among the month's week rows.
func (n calendarNode) cursorWeek() int {
	y, mo, _ := n.m.cursor.Date()
	first := time.Date(y, mo, 1, 0, 0, 0, 0, n.m.cursor.Location())
	return (int(first.Weekday()) + n.m.cursor.Day() - 1) / 7
}

func (n calendarNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	lines := strings.Split(n.m.View(), "\n")
	if len(lines) > s.H {
		if s.H == 1 {
			lines = lines[:1]
		} else {
			weeks := layout.Window(lines[1:], n.cursorWeek(), s.H-1)
			lines = append([]string{lines[0]}, weeks...)
		}
	}
	return layout.Block(strings.Join(lines, "\n")).Render(s)
}
