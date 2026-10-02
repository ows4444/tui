package tui

import (
	"io"
	"runtime"
	"strings"
	"testing"
	"time"
)

// sendGoroutineState returns the scheduler state ("select", "sleep", ...) of
// the goroutine currently inside Program.Send, or "" if there is none.
func sendGoroutineState() string {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	for _, g := range strings.Split(string(buf), "\n\n") {
		if strings.Contains(g, "tui.(*Program).Send(") && !strings.Contains(g, "sendGoroutineState") {
			header, _, _ := strings.Cut(g, "\n")
			return header
		}
	}
	return ""
}

// #25: Send before Run with a full queue parks (a select, not a sleep loop)
// and delivers once Run starts.
func TestSendBeforeRunWithFullQueueBlocksWithoutPollingThenDelivers(t *testing.T) {
	p := NewProgram(recorderModel{quitAfter: 1 << 30})
	for len(p.msgs) < cap(p.msgs) {
		p.msgs <- "fill"
	}
	sent := make(chan struct{})
	go func() { p.Send("late"); close(sent) }()

	deadline := time.Now().Add(2 * time.Second)
	var state string
	for state = sendGoroutineState(); state == "" && time.Now().Before(deadline); state = sendGoroutineState() {
		time.Sleep(time.Millisecond)
	}
	if state == "" {
		t.Fatal("Send returned instead of blocking on the full queue")
	}
	time.Sleep(20 * time.Millisecond) // long enough for a poller to be caught sleeping
	if state = sendGoroutineState(); !strings.Contains(state, "select") || strings.Contains(state, "sleep") {
		t.Fatalf("blocked Send goroutine is %q, want parked in a select", state)
	}
	select {
	case <-sent:
		t.Fatal("Send returned before Run started")
	default:
	}

	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })
	p.input, p.inReader = nil, pr
	done := make(chan Model, 1)
	p.output = &strings.Builder{}
	go func() { m, _ := p.Run(); done <- m }()

	select {
	case <-sent:
	case <-time.After(2 * time.Second):
		t.Fatal("Send did not return after Run started")
	}
	p.Quit()
	select {
	case m := <-done:
		got := m.(recorderModel).received
		if len(got) == 0 || got[len(got)-1] == nil {
			t.Fatalf("received %d msgs", len(got))
		}
		found := false
		for _, g := range got {
			if g == "late" {
				found = true
			}
		}
		if !found {
			t.Fatalf("the blocked message never reached Update: %v", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return")
	}
}

// A Run that returns without the message ever fitting wakes the sender too.
func TestSendBlockedBeforeRunReturnsWhenRunEndsFirst(t *testing.T) {
	p := NewProgram(recorderModel{})
	for len(p.msgs) < cap(p.msgs) {
		p.msgs <- "fill"
	}
	sent := make(chan struct{})
	go func() { p.Send("x"); close(sent) }()
	time.Sleep(10 * time.Millisecond)
	p.setLoopDone(make(chan struct{}))
	p.setLoopDone(nil) // Run started and returned
	select {
	case <-sent:
	case <-time.After(2 * time.Second):
		t.Fatal("Send stayed blocked after Run ended")
	}
}
