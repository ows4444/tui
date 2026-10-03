package tuitest_test

import (
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tuitest"
)

type tickModel struct{ n *int }

type tickMsg struct{}

func (m tickModel) Init() tui.Cmd {
	c, _ := tui.Every(100*time.Millisecond, func(time.Time) tui.Msg { return tickMsg{} })
	return c
}
func (m tickModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if _, ok := msg.(tickMsg); ok {
		*m.n++
	}
	return m, nil
}
func (m tickModel) View() string { return "ticks" }

// Criterion #56: Advance(1s) on a 100 ms Every delivers exactly 10 ticks,
// without waiting on the wall clock.
func TestAdvanceFiresEveryDeterministically(t *testing.T) {
	n := 0
	s := tuitest.New(tickModel{&n}, 20, 3, tuitest.WithClock(tuitest.NewFakeClock()))
	defer s.Close()
	start := time.Now()
	s.Advance(time.Second)
	// Far less than the simulated second, with room for a slow machine.
	if wall := time.Since(start); wall > 500*time.Millisecond {
		t.Errorf("Advance(1s) took %v of wall time; it must not wait on the wall clock", wall)
	}
	if n != 10 {
		t.Errorf("ticks = %d, want 10", n)
	}
}
