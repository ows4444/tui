package skeleton

import (
	"context"
	"testing"
	"time"

	"github.com/ows4444/tui/motion"
)

func TestSharedClockMovesShimmerInSync(t *testing.T) {
	c := motion.NewClock(time.Millisecond)
	a, b := New(), New()
	a.Width, a.Lines, b.Width, b.Lines = 10, 1, 10, 1
	a.Clock, b.Clock = c, c
	if a.Start() != nil {
		t.Fatal("clocked skeleton scheduled its own tick")
	}
	first := a.View()
	c.Update(c.Start()(context.Background()))
	if a.View() != b.View() || a.View() == first {
		t.Error("shimmer must advance with the clock, equally for both")
	}
}
