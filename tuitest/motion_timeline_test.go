package tuitest_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/tuitest"
)

type tlModel struct {
	clk *motion.Clock
	tl  *motion.Timeline
	at  *time.Time // time of the tick being handled
}

func (m tlModel) Init() tui.Cmd {
	m.tl.Start(m.clk.Now())
	c := m.clk.Start()
	return tui.FromCtx(func(ctx context.Context) tui.Msg { return c(ctx) })
}

func (m tlModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if tick, ok := msg.(motion.TickMsg); ok {
		*m.at = tick.At
		m.tl.Update(tick.At)
	}
	c := m.clk.Update(msg)
	if c == nil {
		return m, nil
	}
	return m, tui.FromCtx(func(ctx context.Context) tui.Msg { return c(ctx) })
}

func (m tlModel) View() string { return "timeline" }

// Criterion #104: with two sequential tweens, the second starts when the
// first ends, measured on the FakeClock.
func TestTimelineSecondTweenStartsWhenFirstEnds(t *testing.T) {
	fc := tuitest.NewFakeClock()
	epoch := fc.Now()
	clk := motion.NewClock(50 * time.Millisecond)
	clk.Source = fc
	var log []string
	var at time.Time
	rec := func(name string) func(float64) {
		return func(v float64) {
			log = append(log, fmt.Sprintf("%s@%v=%v", name, at.Sub(epoch), v))
		}
	}
	tl := new(motion.Timeline).
		Then(motion.Tween{To: 1, Duration: 100 * time.Millisecond}, rec("a")).
		Then(motion.Tween{To: 1, Duration: 100 * time.Millisecond}, rec("b"))
	s := tuitest.New(tlModel{clk, tl, &at}, 20, 3, tuitest.WithClock(fc))
	defer s.Close()
	s.Advance(100 * time.Millisecond)
	want := []string{"a@50ms=0.5", "a@100ms=1", "b@100ms=0"}
	if fmt.Sprint(log) != fmt.Sprint(want) {
		t.Fatalf("log = %v, want %v", log, want)
	}
	s.Advance(100 * time.Millisecond)
	if last := log[len(log)-1]; last != "b@200ms=1" {
		t.Errorf("last = %q, want b@200ms=1", last)
	}
}
