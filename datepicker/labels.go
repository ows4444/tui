package datepicker

import (
	"fmt"
	"strings"
	"time"

	"github.com/ows4444/tui/ansi"
)

// Labels are the words the picker shows and speaks, so it can be shown in
// another language or start its week on another day. The zero value is English
// with the week starting on Sunday, which is what the picker did before Labels
// existed; any field left empty falls back to English.
type Labels struct {
	// Weekdays are the header cells, Sunday first, each fitted to two columns
	// (cut or padded on the right).
	Weekdays [7]string
	// WeekdayNames and MonthNames are the full names, Sunday and January first,
	// used when the date is read aloud and by Title.
	WeekdayNames [7]string
	MonthNames   [12]string
	// WeekStart is the first column of the grid; the zero value is Sunday.
	WeekStart time.Weekday
	// Spoken writes a date for speech from the weekday and month names; nil is
	// "Monday, January 2, 2006".
	Spoken func(weekday, month string, day, year int) string
	// Phrases of Linearize: "Date picker, cursor on", "unavailable", "Selected
	// date", "Earliest date" and "Latest date".
	Picker, Unavailable, Selected, Earliest, Latest string
}

var englishDays = [7]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}

// labels returns m.Labels with every empty field filled in with English.
func (m Model) labels() Labels {
	l := m.Labels
	for i := 0; i < 7; i++ {
		if l.Weekdays[i] == "" {
			l.Weekdays[i] = englishDays[i]
		}
		if l.WeekdayNames[i] == "" {
			l.WeekdayNames[i] = time.Weekday(i).String()
		}
	}
	for i := 0; i < 12; i++ {
		if l.MonthNames[i] == "" {
			l.MonthNames[i] = time.Month(i + 1).String()
		}
	}
	if l.Spoken == nil {
		l.Spoken = func(wd, mo string, d, y int) string { return fmt.Sprintf("%s, %s %d, %d", wd, mo, d, y) }
	}
	for _, p := range []struct {
		dst *string
		def string
	}{
		{&l.Picker, "Date picker, cursor on"}, {&l.Unavailable, "unavailable"}, {&l.Selected, "Selected date"},
		{&l.Earliest, "Earliest date"}, {&l.Latest, "Latest date"},
	} {
		if *p.dst == "" {
			*p.dst = p.def
		}
	}
	return l
}

// speak writes t for speech.
func (m Model) speak(t time.Time) string {
	l := m.labels()
	return l.Spoken(l.WeekdayNames[t.Weekday()], l.MonthNames[t.Month()-1], t.Day(), t.Year())
}

// Title is the month the grid shows, such as "October 2026", in the month names
// of Labels.
func (m Model) Title() string {
	l := m.labels()
	return fmt.Sprintf("%s %d", l.MonthNames[m.cursor.Month()-1], m.cursor.Year())
}

// lead is how many blank cells come before the 1st of first's month in the
// grid, given the week's first day.
func (m Model) lead(first time.Time) int {
	return (int(first.Weekday()) - int(m.Labels.WeekStart) + 7) % 7
}

// header is the weekday row, starting at WeekStart, each cell two columns wide.
func (m Model) header() string {
	l := m.labels()
	cells := make([]string, 7)
	for i := range cells {
		w := ansi.Truncate(ansi.Sanitize(l.Weekdays[(int(l.WeekStart)+i)%7]), 2)
		cells[i] = w + strings.Repeat(" ", 2-ansi.Width(w))
	}
	return strings.Join(cells, " ")
}
