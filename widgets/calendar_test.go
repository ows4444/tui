package widgets

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

var oct2026 = time.Date(2026, time.October, 16, 0, 0, 0, 0, time.UTC)

func TestCalendarLaysOutAMonth(t *testing.T) {
	got := ansi.StripANSI(Calendar(oct2026, CalendarOptions{}, theme.DarkTheme()))
	want := "    October 2026\n" +
		"Su Mo Tu We Th Fr Sa\n" +
		"             1  2  3\n" +
		" 4  5  6  7  8  9 10\n" +
		"11 12 13 14 15 16 17\n" +
		"18 19 20 21 22 23 24\n" +
		"25 26 27 28 29 30 31"
	if got != want {
		t.Errorf("Calendar =\n%s\nwant\n%s", got, want)
	}
}

func TestCalendarFirstWeekdayAndNoTitle(t *testing.T) {
	got := ansi.StripANSI(Calendar(oct2026, CalendarOptions{FirstWeekday: time.Monday, NoTitle: true}, theme.DarkTheme()))
	want := "Mo Tu We Th Fr Sa Su\n" +
		"          1  2  3  4\n" +
		" 5  6  7  8  9 10 11\n" +
		"12 13 14 15 16 17 18\n" +
		"19 20 21 22 23 24 25\n" +
		"26 27 28 29 30 31"
	if got != want {
		t.Errorf("Monday-first =\n%s\nwant\n%s", got, want)
	}
}

func TestCalendarMarksAndTodayAreStyledButKeepTheText(t *testing.T) {
	marks := map[int]ansi.Style{3: ansi.NewStyle().Foreground(ansi.Red), 99: ansi.NewStyle().Bold()}
	opts := CalendarOptions{Marks: marks, Today: oct2026}
	got := Calendar(oct2026, opts, theme.DarkTheme())
	plain := ansi.StripANSI(got)
	if plain != ansi.StripANSI(Calendar(oct2026, CalendarOptions{}, theme.DarkTheme())) {
		t.Error("marks or today changed the text")
	}
	if !strings.Contains(got, "\x1b[") {
		t.Error("nothing is styled")
	}
	lines := strings.Split(got, "\n")
	if !strings.Contains(lines[2], "\x1b[31m") { // the 3rd, on the first week row
		t.Errorf("day 3 not styled: %q", lines[2])
	}
	// Today in another month is not highlighted.
	other := time.Date(2026, time.November, 16, 0, 0, 0, 0, time.UTC)
	if strings.Contains(Calendar(oct2026, CalendarOptions{Today: other}, theme.DarkTheme()), "\x1b[") {
		t.Error("a today outside the month was highlighted")
	}
}

func TestCalendarFebruaryAndSundayStart(t *testing.T) {
	feb := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC) // starts on a Sunday, 28 days
	got := ansi.StripANSI(Calendar(feb, CalendarOptions{NoTitle: true}, theme.DarkTheme()))
	lines := strings.Split(got, "\n")
	if lines[1] != " 1  2  3  4  5  6  7" || lines[4] != "22 23 24 25 26 27 28" || len(lines) != 5 {
		t.Errorf("February 2026 =\n%s", got)
	}
}

func TestLinearizeCalendar(t *testing.T) {
	if got := LinearizeCalendar(oct2026, CalendarOptions{}); got != "October 2026, no marked days" {
		t.Errorf("got %q", got)
	}
	got := LinearizeCalendar(oct2026, CalendarOptions{Marks: map[int]ansi.Style{20: {}, 3: {}, 40: {}}})
	if got != "October 2026, marked days: 3, 20" {
		t.Errorf("got %q", got)
	}
}
