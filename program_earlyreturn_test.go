package tui

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ows4444/tui/internal/termio"
)

// runFailsEarly builds a Program on a fake terminal, parks a producer on a
// full queue before Run, lets Run fail before the loop starts, and checks that
// the context is done and the producer is released.
func runFailsEarly(t *testing.T, ft *termio.Fake, wantErr error) {
	t.Helper()
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	p := NewProgram(keyLogger{}, WithInput(pr), WithOutput(&strings.Builder{}), WithTerminal(ft))

	sent := make(chan struct{})
	go func() {
		defer close(sent)
		for i := 0; i < 200; i++ { // more than the queue holds
			p.Send(i)
		}
	}()

	_, err := p.Run()
	if err == nil {
		t.Fatal("Run succeeded; the test would prove nothing")
	}
	if wantErr != nil && !errors.Is(err, wantErr) {
		t.Fatalf("Run = %v, want %v", err, wantErr)
	}
	select {
	case <-p.Context().Done():
	default:
		t.Error("Context is not done after Run returned")
	}
	select {
	case <-sent:
	case <-time.After(3 * time.Second):
		t.Fatal("a Send that started before Run is still blocked after Run returned")
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			p.Send(i)
		}
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Send blocks after Run returned")
	}
}

// When the input is not a terminal, Run returns before the loop starts; the
// context is still cancelled and nobody stays blocked in Send.
func TestRunNotATerminalCancelsContextAndReleasesSend(t *testing.T) {
	runFailsEarly(t, &termio.Fake{Interactive: false}, nil)
}

// The same when raw mode cannot be entered.
func TestRunRawModeErrorCancelsContextAndReleasesSend(t *testing.T) {
	boom := errors.New("no raw")
	runFailsEarly(t, &termio.Fake{Interactive: true, RawErr: boom}, boom)
}

// sendCounter counts the ints it is sent and quits on a QuitMsg from Quit.
type sendCounter struct {
	mu *sync.Mutex
	n  *int
}

func (sendCounter) Init() Cmd { return nil }
func (m sendCounter) Update(msg Msg) (Model, Cmd) {
	if _, ok := msg.(int); ok {
		m.mu.Lock()
		*m.n++
		m.mu.Unlock()
	}
	return m, nil
}
func (sendCounter) View() string { return "x" }

// A second Run while the first is still running reports ErrProgramReused and
// leaves the first alone: its context stays live and it keeps taking messages.
func TestSecondRunLeavesTheRunningProgramAlone(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	var mu sync.Mutex
	n := 0
	p := NewProgram(sendCounter{mu: &mu, n: &n}, WithInput(pr), WithOutput(io.Discard))
	errc := make(chan error, 1)
	go func() {
		_, err := p.Run()
		errc <- err
	}()
	for end := time.Now().Add(3 * time.Second); !p.running(); {
		if time.Now().After(end) {
			t.Fatal("the first Run never started its loop")
		}
		time.Sleep(time.Millisecond)
	}
	p.Send(1)

	if _, err := p.Run(); !errors.Is(err, ErrProgramReused) {
		t.Fatalf("second Run = %v, want ErrProgramReused", err)
	}
	if err := p.Context().Err(); err != nil {
		t.Fatalf("the second Run cancelled the running Program's context: %v", err)
	}
	for i := 0; i < 100; i++ {
		p.Send(i)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		got := n
		mu.Unlock()
		if got == 101 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the running Program handled %d of 101 messages after the second Run", got)
		}
		time.Sleep(time.Millisecond)
	}
	p.Quit()
	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("first Run = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("first Run did not return")
	}
}

// WithContext(nil) is the default parent, context.Background(): NewProgram
// does not panic and the Program's context is live until Run returns.
func TestWithContextNilIsBackground(t *testing.T) {
	p := NewProgram(keyLogger{}, WithContext(nil), WithInput(strings.NewReader("q")), WithOutput(io.Discard)) //nolint:staticcheck // nil is the case under test
	if err := p.Context().Err(); err != nil {
		t.Fatalf("Context before Run: %v", err)
	}
	if _, err := p.Run(); err != nil {
		t.Fatalf("Run = %v", err)
	}
	if p.Context().Err() == nil {
		t.Fatal("Context is not done after Run returned")
	}
}
