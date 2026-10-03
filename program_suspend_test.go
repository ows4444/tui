package tui

import (
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// suspendRec is a model that runs fn as a Suspend when 'e' is typed, records
// every rune key it sees, and quits on 'q'.
type suspendRec struct {
	mu   *sync.Mutex
	keys *[]rune
	fn   func() error
}

func (suspendRec) Init() Cmd { return nil }
func (m suspendRec) Update(msg Msg) (Model, Cmd) {
	k, ok := msg.(Key)
	if !ok || k.Type != KeyRunes || len([]rune(k.Text)) != 1 {
		return m, nil
	}
	m.mu.Lock()
	*m.keys = append(*m.keys, k.Code)
	m.mu.Unlock()
	switch k.Code {
	case 'e':
		return m, Suspend(m.fn)
	case 'q':
		return m, Quit()
	}
	return m, nil
}
func (suspendRec) View() string { return "x" }

func (m suspendRec) got() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return string(*m.keys)
}

func waitKeys(t *testing.T, m suspendRec, want string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for m.got() != want {
		if time.Now().After(deadline) {
			t.Fatalf("keys = %q, want %q", m.got(), want)
		}
		time.Sleep(time.Millisecond)
	}
}

// #28: by the time fn runs the reader has stopped, so fn owns stdin: it reads
// every byte written during it, including bytes the app would otherwise take.
func TestSuspendStopsReaderBeforeFn(t *testing.T) {
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	dn := mustDevNull(t)
	t.Cleanup(func() { dn.Close() })

	var p *Program
	var stoppedAtFn bool
	var fnGot string
	fnDone := make(chan struct{})
	fn := func() error {
		defer close(fnDone)
		select {
		case <-p.rdDone:
			stoppedAtFn = true
		default:
		}
		if _, err := pw.WriteString("xyz"); err != nil {
			return err
		}
		time.Sleep(50 * time.Millisecond) // a live reader would have taken them by now
		buf := make([]byte, 3)
		_ = pr.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, _ := pr.Read(buf)
		_ = pr.SetReadDeadline(time.Time{})
		fnGot = string(buf[:n])
		return nil
	}
	m := suspendRec{mu: &sync.Mutex{}, keys: new([]rune), fn: fn}
	p = NewProgram(m, WithInput(pr), WithOutput(dn))
	ch := startSendLoop(t, p)

	pw.WriteString("e")
	waitKeys(t, m, "e")
	// Update has seen "e" but fn may not have run yet: a "q" written now could
	// land in front of fn's own bytes and be read by fn instead of the app.
	select {
	case <-fnDone:
	case <-time.After(5 * time.Second):
		t.Fatal("fn never finished")
	}
	pw.WriteString("q")
	waitKeys(t, m, "eq")
	if res := <-ch; res.err != nil {
		t.Fatal(res.err)
	}
	if !stoppedAtFn {
		t.Error("reader goroutine still running when fn started")
	}
	if fnGot != "xyz" {
		t.Errorf("fn read %q, want %q", fnGot, "xyz")
	}
}

// #29: after fn returns a new reader is running, and bytes typed during fn
// that fn did not consume are delivered afterwards, not lost.
func TestSuspendRestartsReaderWithoutLosingBytes(t *testing.T) {
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	dn := mustDevNull(t)
	t.Cleanup(func() { dn.Close() })

	var fnGot string
	fn := func() error {
		pw.WriteString("ab")
		buf := make([]byte, 1)
		_ = pr.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, _ := pr.Read(buf)
		_ = pr.SetReadDeadline(time.Time{})
		fnGot = string(buf[:n])
		return nil
	}
	m := suspendRec{mu: &sync.Mutex{}, keys: new([]rune), fn: fn}
	p := NewProgram(m, WithInput(pr), WithOutput(dn))
	ch := startSendLoop(t, p)

	pw.WriteString("e")
	waitKeys(t, m, "eb") // 'a' went to fn, 'b' to the new reader
	pw.WriteString("cq")
	waitKeys(t, m, "ebcq")
	if res := <-ch; res.err != nil {
		t.Fatal(res.err)
	}
	if fnGot != "a" {
		t.Errorf("fn read %q, want %q", fnGot, "a")
	}
}

// A Suspend with no key typed after it, repeated, must not leak readers or
// hang: each cycle stops one reader and starts one.
func TestSuspendRepeatedly(t *testing.T) {
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	dn := mustDevNull(t)
	t.Cleanup(func() { dn.Close() })
	m := suspendRec{mu: &sync.Mutex{}, keys: new([]rune), fn: func() error { return nil }}
	p := NewProgram(m, WithInput(pr), WithOutput(dn))
	ch := startSendLoop(t, p)
	for i := 0; i < 5; i++ {
		pw.WriteString("e")
	}
	waitKeys(t, m, "eeeee")
	pw.WriteString("q")
	if res := <-ch; res.err != nil {
		t.Fatal(res.err)
	}
}

// inflightReader counts the Read calls in progress on r and remembers the most
// it has seen at once.
type inflightReader struct {
	r        io.Reader
	now, max atomic.Int32
}

func (c *inflightReader) Read(b []byte) (int, error) {
	n := c.now.Add(1)
	for {
		m := c.max.Load()
		if n <= m || c.max.CompareAndSwap(m, n) {
			break
		}
	}
	defer c.now.Add(-1)
	return c.r.Read(b)
}

// A WithInput source that is not a file cannot be cancelled, so its reader
// keeps running across a Suspend. Resuming must not start a second one on the
// same decoder: one reader, every key once and in order (run with -race).
func TestSuspendWithPipeInputKeepsOneReader(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })
	in := &inflightReader{r: pr}
	var mu sync.Mutex
	var keys []rune
	rec := suspendRec{mu: &mu, keys: &keys, fn: func() error { return nil }}
	p := NewProgram(rec, WithInput(in), WithOutput(io.Discard))
	errc := make(chan error, 1)
	go func() {
		_, err := p.Run()
		errc <- err
	}()

	want := ""
	for _, k := range "eabecdeq" {
		if _, err := pw.Write([]byte(string(k))); err != nil {
			t.Fatal(err)
		}
		want += string(k)
		waitKeys(t, rec, want)
		if k == 'e' {
			// Give a second reader, if one was started, time to reach Read.
			time.Sleep(20 * time.Millisecond)
		}
	}
	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("Run = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return")
	}
	if got := in.max.Load(); got != 1 {
		t.Fatalf("%d goroutines were reading the input at once, want 1", got)
	}
}
