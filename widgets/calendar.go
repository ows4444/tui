package widgets

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// CalendarOptions shapes Calendar. Every field is optional.
type CalendarOptions struct {
	// FirstWeekday is the first column; the zero value is Sunday.
	FirstWeekday time.Weekday
	// Today, when set, is drawn in the theme's selected style if it falls in the
	// month shown.
	Today time.Time
	// Marks styles days by day of the month (1 to 31): an event, a deadline, a
	// day off. A style alone does not show without colour, so pair it with
	// LinearizeCalendar for accessible output.
	Marks map[int]ansi.Style
	// NoTitle leaves out the "October 2026" line above the weekday names.
	NoTitle bool
}

var weekdayNames = [7]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}

// Calendar renders the month of month as a 20-column grid: a title, the weekday
// names and the days, each two columns wide, for printed output where
// datepicker's interactive calendar is not wanted. It is stateless: the marks
// and today come from options.
func Calendar(month time.Time, o CalendarOptions, t theme.Theme) string {
	y, mo, _ := month.Date()
	loc := month.Location()
	first := time.Date(y, mo, 1, 0, 0, 0, 0, loc)
	last := time.Date(y, mo+1, 0, 0, 0, 0, 0, loc).Day()
	start := int(o.FirstWeekday) % 7

	var b strings.Builder
	if !o.NoTitle {
		title := fmt.Sprintf("%s %d", mo, y)
		b.WriteString(strings.Repeat(" ", max((20-ansi.Width(title))/2, 0)) + title + "\n")
	}
	heads := make([]string, 7)
	for i := range heads {
		heads[i] = weekdayNames[(start+i)%7]
	}
	b.WriteString(strings.Join(heads, " "))

	today := ansi.Style{}
	if !o.Today.IsZero() {
		today = t.ResolvedStates().Selected.Bold()
	}
	col := (int(first.Weekday()) - start + 7) % 7
	b.WriteString("\n" + strings.Repeat("   ", col))
	for day := 1; day <= last; day++ {
		cell := fmt.Sprintf("%2d", day)
		if st, ok := o.Marks[day]; ok {
			cell = st.Render(cell)
		}
		if ty, tm, td := o.Today.Date(); !o.Today.IsZero() && ty == y && tm == mo && td == day {
			cell = today.Render(ansi.StripANSI(cell))
		}
		b.WriteString(cell)
		if col == 6 && day < last {
			b.WriteString("\n")
		} else if day < last {
			b.WriteString(" ")
		}
		col = (col + 1) % 7
	}
	return b.String()
}

// LinearizeCalendar gives a Calendar a plain-text summary for accessible
// output: the month, and the marked days in order.
func LinearizeCalendar(month time.Time, o CalendarOptions) string {
	y, mo, _ := month.Date()
	last := time.Date(y, mo+1, 0, 0, 0, 0, 0, month.Location()).Day()
	var days []int
	for d := range o.Marks {
		if d >= 1 && d <= last {
			days = append(days, d)
		}
	}
	sort.Ints(days)
	out := fmt.Sprintf("%s %d", mo, y)
	if len(days) == 0 {
		return out + ", no marked days"
	}
	parts := make([]string, len(days))
	for i, d := range days {
		parts[i] = strconv.Itoa(d)
	}
	return out + ", marked days: " + strings.Join(parts, ", ")
}
