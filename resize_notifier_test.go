package tui

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/ows4444/tui/internal/termio"
)

// sizes records every ResizeMsg the model sees, safely readable from the test.
type sizeLog struct {
	mu   sync.Mutex
	list []ResizeMsg
}

func (l *sizeLog) add(m ResizeMsg) { l.mu.Lock(); l.list = append(l.list, m); l.mu.Unlock() }
func (l *sizeLog) last() (ResizeMsg, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.list) == 0 {
		return ResizeMsg{}, 0
	}
	return l.list[len(l.list)-1], len(l.list)
}

type sizeModel struct{ log *sizeLog }

func (sizeModel) Init() Cmd { return nil }
func (m sizeModel) Update(msg Msg) (Model, Cmd) {
	if rm, ok := msg.(ResizeMsg); ok {
		m.log.add(rm)
	}
	return m, nil
}
func (sizeModel) View() string { return "v" }

// A Terminal that implements ResizeNotifier delivers resizes through the port:
// no Send(ResizeMsg) and no signal.
func TestResizeNotifierDeliversResizeMsg(t *testing.T) {
	pr, pw := io.Pipe()
	ft := &termio.Fake{W: 80, H: 24, SizeKnown: true}
	log := &sizeLog{}
	p := NewProgram(sizeModel{log}, WithInput(pr), WithOutput(io.Discard), WithTerminal(ft))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	defer func() {
		p.Quit()
		_ = pw.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("Run did not return")
		}
	}()

	ft.Resize(132, 43)
	deadline := time.After(5 * time.Second)
	for {
		if m, _ := log.last(); m == (ResizeMsg{Width: 132, Height: 43}) {
			return
		}
		select {
		case <-deadline:
			m, n := log.last()
			t.Fatalf("last ResizeMsg = %+v after %d, want 132x43", m, n)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// A burst of notifications ends with the final size: values coalesce, none is lost.
func TestResizeNotifierBurstEndsAtTheLastSize(t *testing.T) {
	pr, pw := io.Pipe()
	ft := &termio.Fake{W: 80, H: 24, SizeKnown: true}
	log := &sizeLog{}
	p := NewProgram(sizeModel{log}, WithInput(pr), WithOutput(io.Discard), WithTerminal(ft))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	defer func() {
		p.Quit()
		_ = pw.Close()
		<-done
	}()
	for w := 90; w <= 150; w += 10 {
		ft.Resize(w, 40)
	}
	deadline := time.After(5 * time.Second)
	for {
		if m, _ := log.last(); m == (ResizeMsg{Width: 150, Height: 40}) {
			return
		}
		select {
		case <-deadline:
			m, n := log.last()
			t.Fatalf("last ResizeMsg = %+v after %d, want 150x40", m, n)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// A source that closes its channel stops being read; the forwarder must not spin.
func TestForwardResizeSignalsStopsReadingAClosedSource(t *testing.T) {
	p := NewProgram(staticModel{view: "v"})
	sig := make(chan struct{})
	close(sig)
	calls := make(chan struct{}, 1000)
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		forwardResizeSignals(p, sig, func() (int, int, bool) { calls <- struct{}{}; return 1, 1, true }, done)
		close(finished)
	}()
	time.Sleep(50 * time.Millisecond)
	close(done)
	<-finished
	if n := len(calls); n != 0 {
		t.Fatalf("size was read %d times for a closed source, want 0", n)
	}
}
