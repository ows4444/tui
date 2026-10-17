package widgets

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

var t0 = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

func TestProgressStatsRateAndETA(t *testing.T) {
	s := NewProgressStats(t0)
	s.Update(t0, 0, 100)
	s.Update(t0.Add(10*time.Second), 20, 100)
	if r := s.Rate(); r != 2 {
		t.Fatalf("Rate = %v; want 2", r)
	}
	eta, ok := s.ETA()
	if !ok || eta != 40*time.Second {
		t.Fatalf("ETA = %v, %v; want 40s, true", eta, ok)
	}
	if got, want := s.Text("files"), "20%, 0:10 elapsed, 2.0 files/s, ETA 0:40"; got != want {
		t.Fatalf("Text = %q; want %q", got, want)
	}
}

func TestProgressStatsWindowForgetsOldReadings(t *testing.T) {
	s := NewProgressStats(t0)
	s.Update(t0, 0, 1000)
	s.Update(t0.Add(5*time.Second), 500, 1000) // a burst
	s.Update(t0.Add(30*time.Second), 500, 1000)
	s.Update(t0.Add(40*time.Second), 510, 1000)
	if r := s.Rate(); r != 1 {
		t.Fatalf("Rate = %v; want 1 (only the last ten seconds count)", r)
	}
}

func TestProgressStatsUnknown(t *testing.T) {
	var s ProgressStats // zero value is usable, starts at its first Update
	if s.Elapsed() != 0 || s.Rate() != 0 || s.Fraction() != 0 {
		t.Fatal("zero value is not all zero")
	}
	s.Update(t0, 0, 0)
	s.Update(t0.Add(3*time.Second), 5, 0)
	if _, ok := s.ETA(); ok {
		t.Fatal("ETA told with no total")
	}
	if got, want := s.Text(""), "0:03 elapsed, 1.7/s"; got != want {
		t.Fatalf("Text = %q; want %q", got, want)
	}
}

func TestProgressStatsNoRateYet(t *testing.T) {
	s := NewProgressStats(t0)
	s.Update(t0.Add(2*time.Second), 0, 10)
	if got, want := s.Text(""), "0%, 0:02 elapsed, ETA --:--"; got != want {
		t.Fatalf("Text = %q; want %q", got, want)
	}
}

func TestProgressStatsDoneAndClamp(t *testing.T) {
	s := NewProgressStats(t0)
	s.Update(t0.Add(time.Hour+2*time.Minute+3*time.Second), 12, 10)
	if s.Fraction() != 1 {
		t.Fatalf("Fraction = %v; want 1", s.Fraction())
	}
	if eta, ok := s.ETA(); !ok || eta != 0 {
		t.Fatalf("ETA = %v, %v; want 0, true", eta, ok)
	}
	if !strings.HasPrefix(s.Text(""), "100%, 1:02:03 elapsed") {
		t.Fatalf("Text = %q", s.Text(""))
	}
}

func TestProgressStatsIgnoresPastReadings(t *testing.T) {
	s := NewProgressStats(t0)
	s.Update(t0.Add(10*time.Second), 10, 100)
	s.Update(t0.Add(5*time.Second), 90, 100)
	if s.Fraction() != 0.1 {
		t.Fatalf("Fraction = %v; a reading from the past was applied", s.Fraction())
	}
}

func TestProgressBarStats(t *testing.T) {
	s := NewProgressStats(t0)
	s.Update(t0, 0, 4)
	s.Update(t0.Add(time.Second), 1, 4)
	out := ProgressBarStats(s, 8, "", theme.DarkTheme())
	lines := strings.Split(out, "\n")
	if len(lines) != 2 || ansi.Width(lines[0]) != 8 {
		t.Fatalf("got %q; want an 8 column bar and a line", out)
	}
	if !strings.Contains(ansi.StripANSI(lines[1]), "25%, 0:01 elapsed, 1.0/s, ETA 0:03") {
		t.Fatalf("stats line %q", lines[1])
	}
	if ProgressBarStats(s, 0, "", theme.DarkTheme()) != "" {
		t.Fatal("zero width should draw nothing")
	}
}
