package loadingbar

import (
	"context"
	"testing"
	"time"

	"github.com/ows4444/tui/motion"
)

func TestSharedClockMovesBarInSync(t *testing.T) {
	c := motion.NewClock(time.Millisecond)
	a, b := New(9), New(9)
	a.Clock, b.Clock = c, c
	if a.Start() != nil {
		t.Fatal("clocked bar scheduled its own tick")
	}
	first := a.View()
	c.Update(c.Start()(context.Background()))
	if a.View() != b.View() || a.View() == first {
		t.Error("bar must advance with the clock, equally for both")
	}
}

func TestBouncePosMatchesUpdateWalk(t *testing.T) {
	m := New(9)
	m.Start()
	for n := 1; n < 30; n++ {
		m, _ = m.Update(tickFor(m))
		if want := bouncePos(n, m.maxPos()); m.pos != want {
			t.Fatalf("step %d: pos %d, bouncePos %d", n, m.pos, want)
		}
	}
}
