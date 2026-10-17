package datepicker

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
)

// October 2026 starts on a Thursday.
var oct = time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)

func german() Labels {
	return Labels{
		Weekdays:     [7]string{"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"},
		WeekdayNames: [7]string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"},
		MonthNames: [12]string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August",
			"September", "Oktober", "November", "Dezember"},
		WeekStart: time.Monday,
		Spoken: func(wd, mo string, d, y int) string {
			return fmt.Sprintf("%s, %d. %s %d", wd, d, mo, y)
		},
		Picker: "Datumsauswahl, Cursor auf", Unavailable: "nicht verfügbar", Selected: "Gewählt",
	}
}

func rows(m Model) []string { return strings.Split(ansi.StripANSI(m.View()), "\n") }

func TestDefaultLabelsAreEnglishSundayFirst(t *testing.T) {
	m := New(oct)
	r := rows(m)
	if r[0] != "Su Mo Tu We Th Fr Sa" || !strings.HasPrefix(r[1], "             1  2  3") {
		t.Fatalf("grid:\n%s", strings.Join(r, "\n"))
	}
	if m.Title() != "October 2026" {
		t.Fatalf("Title %q", m.Title())
	}
	if got := m.Linearize(); got != "Date picker, cursor on Saturday, October 17, 2026\nSelected date: Saturday, October 17, 2026" {
		t.Fatalf("%q", got)
	}
}

func TestMondayFirstShiftsTheGrid(t *testing.T) {
	m := New(oct)
	m.Labels = german()
	r := rows(m)
	if r[0] != "Mo Di Mi Do Fr Sa So" {
		t.Fatalf("header %q", r[0])
	}
	// Thursday is the fourth column when the week starts on Monday.
	if !strings.HasPrefix(r[1], "          1  2  3  4") {
		t.Fatalf("first week %q", r[1])
	}
	// Sunday the 4th ends the first week.
	if !strings.HasSuffix(r[1], " 4") || !strings.HasPrefix(strings.TrimLeft(r[2], " "), "5") {
		t.Fatalf("week break:\n%s", strings.Join(r, "\n"))
	}
	if m.Title() != "Oktober 2026" {
		t.Fatalf("Title %q", m.Title())
	}
}

func TestLocalisedSpeech(t *testing.T) {
	m := New(oct)
	m.Labels = german()
	m.Value = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	m.MaxDate = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	got := m.Linearize()
	for _, want := range []string{"Datumsauswahl, Cursor auf Samstag, 17. Oktober 2026", "nicht verfügbar", "Gewählt: Sonntag, 1. März 2026", "Latest date: Donnerstag, 1. Oktober 2026"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestMouseFollowsTheWeekStart(t *testing.T) {
	m := New(oct)
	m.Labels = german()
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 30, H: 8}
	// Row 1 (first week), the Thursday column is index 3: x = 9 -> the 1st.
	m2, _ := m.Update(tui.MouseEvent{X: 9, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m2.cursor.Day() != 1 {
		t.Fatalf("clicked the 1st, cursor on %d", m2.cursor.Day())
	}
	m2, _ = m.Update(tui.MouseEvent{X: 18, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}) // the Sunday column
	if m2.cursor.Day() != 4 {
		t.Fatalf("clicked Sunday, cursor on %d", m2.cursor.Day())
	}
}

func TestLongWeekdayLabelsAreFittedToTwoColumns(t *testing.T) {
	m := New(oct)
	m.Labels.Weekdays = [7]string{"Sunday", "M", "Tue", "W", "Thu", "\x1b[31mF", "世界"}
	h := ansi.StripANSI(m.header())
	if ansi.Width(h) != 20 || strings.Contains(h, "\x1b") || !strings.HasPrefix(h, "Su M ") {
		t.Fatalf("header %q", h)
	}
}

func TestLayoutNodeWindowsOnTheCursorWeekFromTheWeekStart(t *testing.T) {
	m := New(time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC))
	m.Labels.WeekStart = time.Monday
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 20, H: 2}))
	lines := strings.Split(out, "\n")
	if len(lines) != 2 || !strings.Contains(lines[1], "31") {
		t.Fatalf("layout node:\n%s", out)
	}
}
