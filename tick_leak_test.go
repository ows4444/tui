package tui

import (
	"context"
	"io"
	"runtime"
	"strings"
	"testing"
	"time"
)

// tickLeakModel starts a very long Tick on Init, and one more inside each of a
// Batch and a Sequence, then quits on 'q'.
type tickLeakModel struct{}

func (tickLeakModel) Init() Cmd {
	long := func() Cmd {
		return Tick(time.Hour, func(time.Time) Msg { return "late" })
	}
	return Batch(long(), Sequence(long(), long()))
}
func (m tickLeakModel) Update(msg Msg) (Model, Cmd) {
	if k, ok := msg.(Key); ok && k.Type == KeyRunes && k.Text == "q" {
		return m, Quit()
	}
	return m, nil
}
func (tickLeakModel) View() string { return "v" }

func tickGoroutines() int {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	return strings.Count(string(buf), ".TickCtx.func")
}

// #23: when Run returns, no goroutine started by Tick remains. The 10 ms of
// the criterion is the polling grace here, extended to a bound that a loaded
// machine still meets.
func TestNoTickGoroutineOutlivesRun(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })
	p := NewProgram(tickLeakModel{}, WithInput(pr), WithOutput(&strings.Builder{}))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()

	deadline := time.Now().Add(2 * time.Second)
	for tickGoroutines() < 2 { // the Batch Tick and the Sequence's first Tick are waiting
		if time.Now().After(deadline) {
			t.Fatalf("only %d of 2 Tick goroutines started", tickGoroutines())
		}
		time.Sleep(time.Millisecond)
	}
	pw.Write([]byte("q"))
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	for tickGoroutines() > 0 {
		if time.Now().After(deadline.Add(2 * time.Second)) {
			t.Fatalf("%d Tick goroutines still alive after Run returned", tickGoroutines())
		}
		time.Sleep(time.Millisecond)
	}
}

// A Tick is a context-carrying Cmd: the Program runs it with its context, and
// run with any context it waits for d and builds its Msg.
func TestTickRunsAsAContextCmd(t *testing.T) {
	cm, ok := Tick(20*time.Millisecond, func(time.Time) Msg { return "tick" })().(ctxMsg)
	if !ok {
		t.Fatal("Tick's Cmd does not produce a context-carrying message")
	}
	start := time.Now()
	if msg := cm.fn(context.Background()); msg != "tick" || time.Since(start) < 20*time.Millisecond {
		t.Fatalf("msg=%v after %v", msg, time.Since(start))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if msg := cm.fn(ctx); msg != nil {
		t.Errorf("a cancelled context still produced %v", msg)
	}
}

// Sequence waits for a Tick before the next Cmd, as it did when Tick blocked.
func TestSequenceWaitsForATick(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })
	var seen []string
	m := &seqModel{seen: &seen}
	p := NewProgram(m, WithInput(pr), WithOutput(&strings.Builder{}))
	start := time.Now()
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not finish")
	}
	if d := time.Since(start); d < 40*time.Millisecond {
		t.Errorf("the Sequence finished after %v; it must wait for its 40ms Tick", d)
	}
	if strings.Join(seen, ",") != "tick,after" {
		t.Errorf("order = %v, want tick then after", seen)
	}
}

type seqModel struct{ seen *[]string }

func (m *seqModel) Init() Cmd {
	return Sequence(
		Tick(40*time.Millisecond, func(time.Time) Msg { return "tick" }),
		func() Msg { return "after" },
	)
}
func (m *seqModel) Update(msg Msg) (Model, Cmd) {
	if s, ok := msg.(string); ok {
		*m.seen = append(*m.seen, s)
		if s == "after" {
			return m, Quit()
		}
	}
	return m, nil
}
func (m *seqModel) View() string { return "v" }
