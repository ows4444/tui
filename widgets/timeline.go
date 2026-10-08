package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TimelineStatus is how far along one item of a Timeline is.
type TimelineStatus int

const (
	// TimelineDone marks an item that is finished, with the theme's check.
	TimelineDone TimelineStatus = iota
	// TimelineCurrent marks the item in progress, with a filled dot.
	TimelineCurrent
	// TimelinePending marks an item still to come, with an empty dot.
	TimelinePending
	// TimelineFailed marks an item that went wrong, with the theme's cross.
	TimelineFailed
)

// TimelineItem is one entry of a Timeline.
type TimelineItem struct {
	// Title is the entry's one line of text.
	Title string
	// Time is shown before the title, in the muted colour: a clock time, a
	// date, "2h ago". Empty leaves the column out for every item when all
	// are empty.
	Time string
	// Detail is shown under the title, in the muted colour. It may have
	// several lines and is wrapped to the width.
	Detail string
	Status TimelineStatus
}

// Timeline renders items one under another, oldest or newest first as the
// caller orders them: a marker for each item's Status, its Time and Title,
// and its Detail beneath, with a rail joining each marker to the next. The
// four statuses use four different glyphs, so they read without colour.
// Like Stepper, which lays steps out on one row, it is stateless.
//
// width is the number of cells to draw in: a title too long for it is cut
// and details are wrapped. width <= 0 draws every line at its natural
// length. The text of every item is sanitised.
func Timeline(items []TimelineItem, t theme.Theme, width int) string {
	g := t.GlyphSet()
	muted := ansi.NewStyle().Foreground(t.Muted)
	timeW := 0
	for _, it := range items {
		timeW = max(timeW, ansi.Width(ansi.Sanitize(it.Time)))
	}
	indent := 2 // the marker and a space
	if timeW > 0 {
		indent += timeW + 2
	}
	room := 0
	if width > 0 {
		room = max(width-indent, 1)
	}

	var lines []string
	for i, it := range items {
		mark, style := g.DotEmpty, muted
		switch it.Status {
		case TimelineDone:
			mark, style = g.Check, ansi.NewStyle().Foreground(t.Success)
		case TimelineCurrent:
			mark, style = g.Dot, ansi.NewStyle().Bold().Foreground(t.Primary)
		case TimelineFailed:
			mark, style = g.Cross, ansi.NewStyle().Foreground(t.Error)
		}
		title := ansi.Sanitize(it.Title)
		if room > 0 {
			title = ansi.Truncate(title, room)
		}
		line := style.Render(mark) + " "
		if timeW > 0 {
			when := ansi.Sanitize(it.Time)
			line += muted.Render(when) + strings.Repeat(" ", timeW-ansi.Width(when)+2)
		}
		titleStyle := ansi.NewStyle().Foreground(t.Text)
		if it.Status == TimelineCurrent {
			titleStyle = titleStyle.Bold()
		}
		lines = append(lines, line+titleStyle.Render(title))

		// The rail runs beside the detail of every item but the last, so the
		// markers read as joined; the last item's detail is indented only.
		rail := muted.Render(g.RuleV)
		if i == len(items)-1 {
			rail = " "
		}
		detail := ansi.Sanitize(it.Detail)
		if detail == "" {
			continue
		}
		if room > 0 {
			detail = ansi.Wrap(detail, room)
		}
		for _, d := range strings.Split(detail, "\n") {
			lines = append(lines, rail+strings.Repeat(" ", indent-1)+muted.Render(d))
		}
	}
	return strings.Join(lines, "\n")
}
