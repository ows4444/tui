package tui

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// readerOnly hides that r is a file, so WithInput reads it as a plain reader (no
// terminal, no raw mode) instead of as the terminal.
type readerOnly struct{ io.Reader }

// Criterion #16
func TestFromCtxCancelledWithinMsOfRunReturning(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	var out bytes.Buffer
	started := make(chan struct{})
	cancelledAt := make(chan time.Time, 1)
	m := recModel{mu: &sync.Mutex{}, got: &[]Msg{}, init: FromCtx(func(ctx context.Context) Msg {
		close(started)
		<-ctx.Done()
		cancelledAt <- time.Now()
		return nil
	})}
	p := NewProgram(m, WithInput(readerOnly{pr}), WithOutput(&out))
	runDone := make(chan time.Time, 1)
	go func() {
		_, _ = p.Run()
		runDone <- time.Now()
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("FromCtx Cmd never started")
	}
	p.msgs <- QuitMsg{}
	var returned time.Time
	select {
	case returned = <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
	}
	select {
	case at := <-cancelledAt:
		if d := at.Sub(returned); d > time.Millisecond {
			t.Fatalf("context cancelled %v after Run returned, want <= 1ms", d)
		}
	case <-time.After(time.Second):
		t.Fatal("context not cancelled after Run returned")
	}
}

// goSeqModel runs a slow Go Cmd followed by a fast one in a Sequence and
// quits once it has both results.
type goSeqModel struct{ seen *[]string }

func (m goSeqModel) Init() Cmd {
	return Sequence(
		Go(func(context.Context) Msg {
			time.Sleep(40 * time.Millisecond)
			return "first"
		}),
		func() Msg { return "second" },
	)
}
func (m goSeqModel) Update(msg Msg) (Model, Cmd) {
	if s, ok := msg.(string); ok {
		*m.seen = append(*m.seen, s)
		if len(*m.seen) == 2 {
			return m, Quit()
		}
	}
	return m, nil
}
func (goSeqModel) View() string { return "v" }

// A Go Cmd in a Sequence is waited for like any other: its Msg arrives before
// the next Cmd's.
func TestSequenceWaitsForAGoCmd(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })
	var seen []string
	p := NewProgram(goSeqModel{seen: &seen}, WithInput(pr), WithOutput(&strings.Builder{}))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not finish")
	}
	if got := strings.Join(seen, ","); got != "first,second" {
		t.Errorf("order = %s, want first,second", got)
	}
}

// RunCmd runs a Go Cmd with the given context and returns what it produced.
func TestRunCmdRunsAGoCmd(t *testing.T) {
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "v")
	got := RunCmd(ctx, Go(func(ctx context.Context) Msg { return ctx.Value(ctxKey{}) }))
	if got != "v" {
		t.Fatalf("RunCmd(Go(fn)) = %#v, want the Msg fn returned", got)
	}
	if Go(nil) != nil {
		t.Error("Go(nil) is not a nil Cmd")
	}
}
