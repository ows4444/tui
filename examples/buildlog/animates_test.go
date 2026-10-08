package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tuitest"
)

// The program is run as main runs it, from Init on, and the spinner must
// move. A spinner that Init starts on a copy of the model is never marked
// running in the model the Program keeps, and it ignores its own ticks.
func TestTheSpinnerAnimatesInTheRunningProgram(t *testing.T) {
	m := initialModel()
	m.spinner.Interval = 5 * time.Millisecond
	s := tuitest.New(m, 60, 12, tui.WithAltScreen(true))
	defer s.Close()
	first := spinnerFrame(s)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if f := spinnerFrame(s); f != "" && f != first && first != "" {
			return
		} else if first == "" {
			first = f
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("the spinner stayed on %q for 2s", first)
}

// spinnerFrame returns the braille spinner glyph on the screen, or "".
func spinnerFrame(s *tuitest.Session) string {
	for _, row := range s.Screen() {
		if i := strings.IndexAny(row, "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"); i >= 0 {
			return row[i : i+3]
		}
	}
	return ""
}
