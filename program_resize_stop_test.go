package tui

import (
	"io"
	"sync/atomic"
	"testing"
	"time"
)

// slowTerm is a Terminal whose Size takes a while once slow is set, and which
// notes a Size call that is still running, or starts, after the test says
// Run has returned.
type slowTerm struct {
	resizes  chan struct{}
	slow     atomic.Bool
	inSize   atomic.Int32
	returned atomic.Bool
	late     atomic.Int32
}

func (s *slowTerm) IsTerminal() bool { return true }
func (s *slowTerm) Size() (int, int, bool) {
	if s.slow.Load() {
		s.inSize.Add(1)
		time.Sleep(60 * time.Millisecond)
	}
	if s.returned.Load() {
		s.late.Add(1)
	}
	return 80, 24, true
}
func (s *slowTerm) MakeRaw(bool) (func() error, error)    { return func() error { return nil }, nil }
func (s *slowTerm) EnableOutputVT() (func() error, error) { return nil, nil }
func (s *slowTerm) Resizes() <-chan struct{}              { return s.resizes }

// Run waits for its resize watcher: once Run has returned, nothing it started
// is still asking the Terminal for its size, so the caller may close the
// output.
func TestRunWaitsForTheResizeWatcher(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	term := &slowTerm{resizes: make(chan struct{}, 1)}
	p := NewProgram(staticModel{view: "x"}, WithInput(pr), WithOutput(io.Discard), WithTerminal(term))
	done := make(chan error, 1)
	go func() {
		_, err := p.Run()
		term.returned.Store(true)
		done <- err
	}()
	for end := time.Now().Add(3 * time.Second); !p.running(); {
		if time.Now().After(end) {
			t.Fatal("Run never started its loop")
		}
		time.Sleep(time.Millisecond)
	}

	term.slow.Store(true)
	term.resizes <- struct{}{} // the watcher asks for the size, slowly
	for end := time.Now().Add(3 * time.Second); term.inSize.Load() == 0; {
		if time.Now().After(end) {
			t.Fatal("the resize watcher never asked for the size")
		}
		time.Sleep(time.Millisecond)
	}
	p.Quit() // Run ends while that Size call is still in progress

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return")
	}
	time.Sleep(100 * time.Millisecond) // let a leftover Size call finish
	if n := term.late.Load(); n != 0 {
		t.Fatalf("%d Size call(s) were still running or started after Run returned", n)
	}
}
