package tui

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

type recModel struct {
	mu   *sync.Mutex
	got  *[]Msg
	init Cmd
}

func (m recModel) Init() Cmd { return m.init }
func (m recModel) Update(msg Msg) (Model, Cmd) {
	m.mu.Lock()
	*m.got = append(*m.got, msg)
	m.mu.Unlock()
	return m, nil
}
func (m recModel) View() string { return "v" }

func (m recModel) snapshot() []Msg {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Msg(nil), *m.got...)
}

func startRec(t *testing.T, init Cmd, opts ...ProgramOption) (*Program, recModel, *bytes.Buffer, func()) {
	t.Helper()
	pr, pw := mustPipe(t)
	var out bytes.Buffer
	m := recModel{mu: &sync.Mutex{}, got: &[]Msg{}, init: init}
	p := NewProgram(m, append([]ProgramOption{WithInput(pr), WithOutput(&out)}, opts...)...)
	done := make(chan struct{})
	go func() { p.runLoop(); close(done) }()
	stop := func() {
		p.msgs <- QuitMsg{}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("runLoop did not return")
		}
		pr.Close()
		pw.Close()
	}
	return p, m, &out, stop
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

type strMsg string

// Criterion #72
func TestSequenceDeliversEachMessageBeforeStartingNext(t *testing.T) {
	var mu sync.Mutex
	var events []string
	rec := func(s string) { mu.Lock(); events = append(events, s); mu.Unlock() }
	var m recModel
	mk := func(name string, d time.Duration) Cmd {
		return func() Msg {
			rec("start " + name)
			time.Sleep(d)
			return strMsg(name)
		}
	}
	_, m, _, stop := startRec(t, Sequence(mk("a", 30*time.Millisecond), nil, mk("b", 0), mk("c", 0)))
	defer stop()
	waitFor(t, func() bool { return len(m.snapshot()) >= 4 }) // resize + a b c
	var names []string
	for _, x := range m.snapshot() {
		if s, ok := x.(strMsg); ok {
			names = append(names, string(s))
		}
	}
	if strings.Join(names, ",") != "a,b,c" {
		t.Fatalf("delivery order = %v", names)
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(events, ",") != "start a,start b,start c" {
		t.Fatalf("start order = %v", events)
	}
}

// Criterion #73
func TestEveryCancelStopsFurtherTicks(t *testing.T) {
	var n int
	var mu sync.Mutex
	cmd, cancel := Every(5*time.Millisecond, func(time.Time) Msg {
		mu.Lock()
		n++
		mu.Unlock()
		return strMsg("tick")
	})
	_, m, _, stop := startRec(t, cmd)
	defer stop()
	count := func() int {
		c := 0
		for _, x := range m.snapshot() {
			if _, ok := x.(strMsg); ok {
				c++
			}
		}
		return c
	}
	waitFor(t, func() bool { return count() >= 3 })
	cancel()
	cancel() // idempotent
	after := count()
	time.Sleep(60 * time.Millisecond)
	if got := count(); got != after {
		t.Fatalf("ticks after cancel: %d -> %d", after, got)
	}
}

// Criterion #74
func TestGoPassesProgramContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	var out bytes.Buffer
	got := make(chan error, 1)
	m := recModel{mu: &sync.Mutex{}, got: &[]Msg{}}
	started := make(chan struct{})
	m.init = Go(func(c context.Context) Msg {
		close(started)
		<-c.Done()
		got <- c.Err()
		return strMsg("done")
	})
	p := NewProgram(m, WithInput(pr), WithOutput(&out), WithContext(ctx))
	go p.runLoop()
	// Cancelling the parent ends the loop, so wait until fn is running: a Cmd
	// the loop has not started yet is dropped, as it is on Quit.
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("fn never started")
	}
	cancel()
	select {
	case err := <-got:
		if err != context.Canceled {
			t.Fatalf("ctx err = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fn never saw cancelled context")
	}
	p.msgs <- QuitMsg{}
}

func TestRuntimeModeCmds(t *testing.T) {
	p, _, out, stop := startRec(t, nil, WithAltScreen(false))
	for _, c := range []Cmd{EnterAltScreen(), SetWindowTitle("hi\x1b]x"), EnableMouse(MouseAllMotion), ClearScreen(), ExitAltScreen()} {
		p.msgs <- c()
	}
	stop()
	s := out.String()
	for _, want := range []string{"\x1b[?1049h", "\x1b]0;hi]x\x07", "\x1b[?1003h", "\x1b[2J", "\x1b[?1049l"} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q in %q", want, s)
		}
	}
}
