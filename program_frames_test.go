package tui

import (
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// pacedMsg is a non-input message (like a Tick or Send result).
type pacedMsg struct{}

// pacedModel counts View calls and shows how many non-resize Msgs it has
// seen; with constant set, View never changes.
type pacedModel struct {
	views, last *int32
	received    int
	constant    bool
}

func (m pacedModel) Init() Cmd { return nil }
func (m pacedModel) Update(msg Msg) (Model, Cmd) {
	if _, ok := msg.(ResizeMsg); !ok {
		m.received++
	}
	return m, nil
}
func (m pacedModel) View() string {
	atomic.AddInt32(m.views, 1)
	atomic.StoreInt32(m.last, int32(m.received))
	if m.constant {
		return "static"
	}
	return strconv.Itoa(m.received)
}

// startPaced runs the loop for a pacedModel and returns the program, the
// counters, the output file and a stop func that quits and waits.
func startPaced(t *testing.T, constant bool, opts ...ProgramOption) (p *Program, views, last *int32, out *os.File, stop func()) {
	t.Helper()
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	out, _ = captureOutput(t)
	views, last = new(int32), new(int32)
	opts = append([]ProgramOption{WithInput(pr), WithOutput(out)}, opts...)
	p = NewProgram(pacedModel{views: views, last: last, constant: constant}, opts...)
	done := make(chan struct{})
	go func() { _, _ = p.runLoop(); close(done) }()
	time.Sleep(30 * time.Millisecond) // seed render
	return p, views, last, out, func() {
		p.msgs <- QuitMsg{}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("runLoop did not return")
		}
	}
}

func fileSize(t *testing.T, f *os.File) int64 {
	t.Helper()
	fi, err := os.Stat(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return fi.Size()
}

// Criterion 1: an unchanged View string writes nothing.
func TestDirtySkipWritesNothingForUnchangedView(t *testing.T) {
	p, _, _, out, stop := startPaced(t, true)
	before := fileSize(t, out)
	if before == 0 {
		t.Fatal("seed render wrote nothing")
	}
	for i := 0; i < 5; i++ {
		p.msgs <- pacedMsg{}
		time.Sleep(25 * time.Millisecond) // past the default cap interval
	}
	if after := fileSize(t, out); after != before {
		t.Errorf("output grew by %d bytes for an unchanged view, want 0", after-before)
	}
	stop()
}

// A changed view is still written after skipped ones.
func TestDirtySkipStillWritesChangedView(t *testing.T) {
	p, _, _, out, stop := startPaced(t, false)
	before := fileSize(t, out)
	p.msgs <- pacedMsg{}
	time.Sleep(40 * time.Millisecond)
	if fileSize(t, out) == before {
		t.Error("changed view was not written")
	}
	stop()
}

// Criterion 2: a fast non-input burst renders at most cap frames per second
// and ends with a trailing flush of the latest state.
func TestDefaultCapCoalescesNonInputBurstWithTrailingFlush(t *testing.T) {
	p, views, last, _, stop := startPaced(t, false)
	base := atomic.LoadInt32(views)
	for i := 0; i < 200; i++ {
		p.msgs <- pacedMsg{}
	}
	time.Sleep(100 * time.Millisecond)
	if got := atomic.LoadInt32(views) - base; got > 6 {
		t.Errorf("%d renders for a 200-msg burst, want a handful (cap 60/s)", got)
	}
	if got := atomic.LoadInt32(last); got != 200 {
		t.Errorf("last rendered state = %d, want 200 (trailing flush)", got)
	}
	stop()
}

// Criterion 3: a key renders immediately, even right after another render.
func TestDefaultCapDoesNotDelayKeys(t *testing.T) {
	p, views, last, _, stop := startPaced(t, false)
	base := atomic.LoadInt32(views)
	for i := 0; i < 5; i++ {
		p.msgs <- Key{Type: KeyRunes, Text: "a", Code: 'a'}
		// Wait for this key's render before sending the next, so each key
		// lands right after a render without depending on scheduler timing.
		want := base + int32(i+1)
		deadline := time.Now().Add(5 * time.Second)
		for atomic.LoadInt32(views) < want {
			if time.Now().After(deadline) {
				t.Fatalf("key %d was not rendered (views=%d, want %d)", i+1, atomic.LoadInt32(views), want)
			}
			time.Sleep(200 * time.Microsecond)
		}
	}
	if got := atomic.LoadInt32(views) - base; got != 5 {
		t.Errorf("%d renders for 5 keys, want 5", got)
	}
	if got := atomic.LoadInt32(last); got != 5 {
		t.Errorf("last = %d, want 5", got)
	}
	stop()
}

// WithMaxFPS(0) is the documented way to turn the default cap off.
func TestWithMaxFPSZeroDisablesDefaultCap(t *testing.T) {
	p, views, _, _, stop := startPaced(t, false, WithMaxFPS(0))
	base := atomic.LoadInt32(views)
	for i := 0; i < 5; i++ {
		p.msgs <- pacedMsg{}
	}
	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(views) - base; got != 5 {
		t.Errorf("%d renders for 5 msgs with the cap off, want 5", got)
	}
	stop()
}
