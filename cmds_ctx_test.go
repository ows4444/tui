package tui

import (
	"bytes"
	"context"
	"io"
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
