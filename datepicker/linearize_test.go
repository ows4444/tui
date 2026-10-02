package datepicker

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	day := time.Date(2026, time.March, 4, 0, 0, 0, 0, time.UTC)
	m := New(day)
	want := "Date picker, cursor on Wednesday, March 4, 2026\nSelected date: Wednesday, March 4, 2026"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	m.MinDate = time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)
	m.MaxDate = time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC)
	got := m.Linearize()
	for _, w := range []string{", unavailable", "Earliest date: Tuesday, March 10, 2026", "Latest date: Friday, March 20, 2026"} {
		if !strings.Contains(got, w) {
			t.Errorf("Linearize lacks %q:\n%s", w, got)
		}
	}
}
