package widgets

import (
	"fmt"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// progressWindow is how far back ProgressStats looks to measure the rate, so a
// stall or a burst a minute ago does not keep skewing the estimate.
const progressWindow = 10 * time.Second

type progressSample struct {
	at   time.Time
	done float64
}

// ProgressStats turns a stream of "done of total" readings into the numbers a
// progress bar is usually shown with: time elapsed, rate and time remaining.
// ProgressBar draws a fraction only; the caller feeds a ProgressStats the
// current count with Update and reads the text back. It never reads the clock
// itself: the time is passed in, so a test (or a tui.Clock) controls it.
//
// The rate is measured over the last ten seconds of readings. The zero value
// is ready to use; its start is the time of the first Update.
type ProgressStats struct {
	start   time.Time
	started bool
	now     time.Time
	done    float64
	total   float64
	window  []progressSample
}

// NewProgressStats returns a ProgressStats whose elapsed time counts from
// start.
func NewProgressStats(start time.Time) *ProgressStats {
	return &ProgressStats{start: start, started: true, now: start}
}

// Update records that done of total units were finished at now. Units are the
// caller's: bytes, files, steps. A total of zero or less means the total is
// not known, so there is no ETA. Readings that go back in time are ignored.
func (s *ProgressStats) Update(now time.Time, done, total float64) {
	if !s.started {
		s.start, s.started = now, true
	}
	if now.Before(s.now) {
		return
	}
	s.now, s.done, s.total = now, done, total
	s.window = append(s.window, progressSample{now, done})
	cut := 0
	// Keep the newest reading at or before the window's edge as the
	// baseline, so sparse updates still leave two readings to measure.
	for cut < len(s.window)-1 && now.Sub(s.window[cut+1].at) >= progressWindow {
		cut++
	}
	s.window = s.window[cut:]
}

// Fraction is done over total, clamped to [0, 1]; it is 0 while the total is
// unknown. It is what ProgressBar takes.
func (s *ProgressStats) Fraction() float64 {
	if s.total <= 0 || s.done <= 0 {
		return 0
	}
	if s.done >= s.total {
		return 1
	}
	return s.done / s.total
}

// Elapsed is the time from the start to the latest Update.
func (s *ProgressStats) Elapsed() time.Duration {
	if !s.started {
		return 0
	}
	return s.now.Sub(s.start)
}

// Rate is units finished per second over the last ten seconds of readings. It
// is 0 until two readings, at different times, are in hand.
func (s *ProgressStats) Rate() float64 {
	if len(s.window) < 2 {
		return 0
	}
	first, last := s.window[0], s.window[len(s.window)-1]
	dt := last.at.Sub(first.at).Seconds()
	if dt <= 0 {
		return 0
	}
	if r := (last.done - first.done) / dt; r > 0 {
		return r
	}
	return 0
}

// ETA is the time left at the current rate, and false while it cannot be
// told: the total is unknown, the rate is zero, or nothing has been done.
// Once done reaches total it is zero and true.
func (s *ProgressStats) ETA() (time.Duration, bool) {
	if s.total <= 0 {
		return 0, false
	}
	if s.done >= s.total {
		return 0, true
	}
	r := s.Rate()
	if r <= 0 {
		return 0, false
	}
	return time.Duration((s.total - s.done) / r * float64(time.Second)), true
}

// Text is a short line for under a bar, such as
// "42%, 0:12 elapsed, 3.2/s, ETA 0:41". The percentage is left out while
// the total is unknown, and the ETA reads "ETA --:--" while it cannot be told.
// unit is appended to the rate ("3.2 files/s"); it may be empty.
func (s *ProgressStats) Text(unit string) string {
	out := ""
	if s.total > 0 {
		out = fmt.Sprintf("%d%%, ", int(s.Fraction()*100))
	}
	out += formatClock(s.Elapsed()) + " elapsed"
	if r := s.Rate(); r > 0 {
		if unit != "" {
			unit = " " + unit
		}
		out += fmt.Sprintf(", %s%s/s", formatRate(r), unit)
	}
	if eta, ok := s.ETA(); ok {
		out += ", ETA " + formatClock(eta)
	} else if s.total > 0 {
		out += ", ETA --:--"
	}
	return out
}

// ProgressBarStats is ProgressBar for s.Fraction() with s.Text(unit) on the
// line below it, in the muted colour.
func ProgressBarStats(s *ProgressStats, width int, unit string, t theme.Theme) string {
	bar := ProgressBar(s.Fraction(), width, t)
	if width <= 0 {
		return bar
	}
	return bar + "\n" + ansi.NewStyle().Foreground(t.Muted).Render(s.Text(unit))
}

// formatClock writes d as m:ss, or h:mm:ss from an hour on. A negative d is 0.
func formatClock(d time.Duration) string {
	secs := int(d.Round(time.Second) / time.Second)
	if secs < 0 {
		secs = 0
	}
	if secs >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", secs/3600, secs%3600/60, secs%60)
	}
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

// formatRate keeps one decimal below 10 and none from 10 up: 3.2, 42, 1500.
func formatRate(r float64) string {
	if r < 10 {
		return fmt.Sprintf("%.1f", r)
	}
	return fmt.Sprintf("%.0f", r)
}
