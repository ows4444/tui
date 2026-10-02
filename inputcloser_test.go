package tui

import (
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type countCloser struct {
	io.Reader
	n atomic.Int32
}

func (c *countCloser) Close() error { c.n.Add(1); return nil }

func TestInputPipeReturnsPromptlyOnQuit(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	p := NewProgram(inspModel{keys: new([]string)}, WithInput(pr), WithOutput(&strings.Builder{}))
	quitAt := make(chan time.Time, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		quitAt <- time.Now()
		p.Send(QuitMsg{})
	}()
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	// Generous for -race scheduling; the 250ms stop timeout is the failure mode.
	if d := time.Since(<-quitAt); d > 100*time.Millisecond {
		t.Fatalf("Run took %v after quit", d)
	}
}

func TestInputCloserClosedOnce(t *testing.T) {
	for _, explicit := range []bool{true, false} {
		pr, pw := io.Pipe()
		c := &countCloser{Reader: pr}
		var opts []ProgramOption
		if explicit {
			opts = []ProgramOption{WithInput(pr), WithInputCloser(c)}
		} else {
			opts = []ProgramOption{WithInput(c)}
		}
		opts = append(opts, WithOutput(&strings.Builder{}))
		p := NewProgram(inspModel{keys: new([]string)}, opts...)
		go func() { time.Sleep(20 * time.Millisecond); p.Send(QuitMsg{}) }()
		if _, err := p.Run(); err != nil {
			t.Fatal(err)
		}
		pw.Close()
		if n := c.n.Load(); n != 1 {
			t.Fatalf("explicit=%v: closed %d times, want 1", explicit, n)
		}
	}
}
