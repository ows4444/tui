package main

import "testing"

func TestWeeksStartOnMonday(t *testing.T) {
	ws := weeks(date(2026, 10, 14)) // 1 October 2026 is a Thursday
	if len(ws) != 5 {
		t.Fatalf("%d weeks, want 5", len(ws))
	}
	if !ws[0][0].IsZero() || !ws[0][2].IsZero() || ws[0][3].Day() != 1 {
		t.Fatalf("first row = %v", ws[0])
	}
	last := ws[len(ws)-1]
	if last[5].Day() != 31 || !last[6].IsZero() {
		t.Fatalf("last row = %v", last)
	}
}

func TestEveryDayOfAMonthAppearsOnce(t *testing.T) {
	for _, m := range []day{date(2026, 2, 1), date(2028, 2, 1), date(2026, 12, 1), date(2026, 11, 1)} {
		seen := 0
		for _, w := range weeks(m) {
			for _, d := range w {
				if !d.IsZero() {
					seen++
				}
			}
		}
		want := int(addMonths(m, 1).Sub(m).Hours() / 24)
		if seen != want {
			t.Errorf("%s: %d days drawn, want %d", m.Format("2006-01"), seen, want)
		}
	}
}

func TestMonthMath(t *testing.T) {
	if got := addMonths(date(2026, 12, 20), 1); got != date(2027, 1, 1) {
		t.Errorf("december + 1 = %v", got)
	}
	if got := addMonths(date(2026, 1, 20), -1); got != date(2025, 12, 1) {
		t.Errorf("january - 1 = %v", got)
	}
	if got := nights(date(2026, 3, 28), date(2026, 3, 30)); got != 2 {
		t.Errorf("nights across the clock change = %d", got)
	}
	if column(date(2026, 10, 18)) != 6 || column(date(2026, 10, 19)) != 0 {
		t.Error("Sunday is not column 6 and Monday column 0")
	}
}
