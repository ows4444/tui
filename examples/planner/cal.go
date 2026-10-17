package main

import "time"

// day is a calendar date at midnight UTC: dates are compared and counted
// without a time zone or daylight saving getting in the way.
type day = time.Time

func date(y int, m time.Month, d int) day { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func startOfDay(t time.Time) day { return date(t.Year(), t.Month(), t.Day()) }

func firstOfMonth(d day) day { return date(d.Year(), d.Month(), 1) }

func addDays(d day, n int) day { return d.AddDate(0, 0, n) }

// addMonths moves to the first of the month n months away.
func addMonths(d day, n int) day { return date(d.Year(), d.Month()+time.Month(n), 1) }

// column is the weekday as a column of a week that starts on Monday.
func column(d day) int { return (int(d.Weekday()) + 6) % 7 }

// nights is how many nights lie between two dates.
func nights(a, b day) int { return int(b.Sub(a).Hours() / 24) }

// weeks lays the month of d out as rows of seven days, Monday first. Days
// that belong to the month before or after are the zero time.
func weeks(d day) [][7]day {
	first := firstOfMonth(d)
	var out [][7]day
	var row [7]day
	col := column(first)
	for cur := first; cur.Month() == first.Month(); cur = addDays(cur, 1) {
		row[col] = cur
		col++
		if col == 7 {
			out = append(out, row)
			row, col = [7]day{}, 0
		}
	}
	if col > 0 {
		out = append(out, row)
	}
	return out
}

func dayKey(d day) string { return d.Format("2006-01-02") }
