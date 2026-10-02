package spinner

import (
	"context"
	"testing"
	"time"

	"github.com/ows4444/tui/motion"
)

func TestSharedClockKeepsSpinnersInPhase(t *testing.T) {
	c := motion.NewClock(time.Millisecond)
	a, b := New(), New()
	a.Clock, b.Clock = c, c
	if a.Start() != nil || b.Start() != nil {
		t.Fatal("clocked spinner scheduled its own tick")
	}
	cmd := c.Start()
	prev := a.View()
	for i := 0; i < 12; i++ {
		if a.View() != b.View() {
			t.Fatalf("tick %d: spinners differ", i)
		}
		cmd = c.Update(cmd(context.Background()))
		if a.View() == prev {
			t.Fatalf("tick %d: frame did not advance", i)
		}
		prev = a.View()
	}
}

func TestReducedClockSchedulesNoTick(t *testing.T) {
	c := &motion.Clock{Interval: time.Millisecond, Motion: motion.Reduced}
	if c.Start() != nil {
		t.Error("reduced clock scheduled a tick")
	}
}
