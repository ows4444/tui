package skeleton

import (
	"context"
	"testing"
	"time"

	"github.com/ows4444/tui"
)

// A program that holds two Models passes every message to both. Each must
// act on its own tick only: one that took the other's would advance twice
// per interval and schedule a second tick, and the number of pending ticks
// would double every interval.
func TestATickIsIgnoredByAnotherModel(t *testing.T) {
	a, b := New(), New()
	a.Interval, b.Interval = time.Millisecond, time.Millisecond
	tick := tui.RunCmd(context.Background(), a.Start())
	b.Start()

	a2, cmdA := a.Update(tick)
	if a2.frame == a.frame || cmdA == nil {
		t.Fatalf("its own tick: advanced = %v, rescheduled = %v", a2.frame != a.frame, cmdA != nil)
	}
	b2, cmdB := b.Update(tick)
	if b2.frame != b.frame || cmdB != nil {
		t.Fatalf("another Model's tick: advanced = %v, rescheduled = %v; want it ignored", b2.frame != b.frame, cmdB != nil)
	}
	// A copy is the same widget: the tick still reaches it.
	c := a
	if c2, _ := c.Update(tick); c2.frame == c.frame {
		t.Fatal("a copy ignored its own tick")
	}
}

// A program whose Init has a value receiver calls Start twice: once where
// the model is built, so the kept model is running, and once in Init for
// the Cmd. The tick from the second call must still reach the kept model.
func TestATickFromStartOnACopyReachesTheModel(t *testing.T) {
	m := New()
	m.Interval = time.Millisecond
	m.Start()
	kept := m
	tick := tui.RunCmd(context.Background(), m.Start())
	if next, cmd := kept.Update(tick); next.frame == kept.frame || cmd == nil {
		t.Fatalf("advanced = %v, rescheduled = %v", next.frame != kept.frame, cmd != nil)
	}
}

// tickFor is the tick m's own chain delivers next.
func tickFor(m Model) tickMsg { return tickMsg{id: m.id} }
