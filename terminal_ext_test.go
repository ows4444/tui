package tui

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// modeApply runs a mode Cmd's effect on p, as the event loop would.
func modeApply(p *Program, c Cmd) { c().(modeMsg).apply(p) }

// Criterion #61: the title is pushed (CSI 22;0t) before the first title is set
// and popped (CSI 23;0t) on the way out.
func TestSetWindowTitlePushesOnceAndPopsOnQuit(t *testing.T) {
	p, out, errc := startSpinning(t)
	p.Send(SetWindowTitle("one")())
	p.Send(SetWindowTitle("two")())
	p.Send(QuitMsg{})
	if err := waitRun(t, errc); err != nil {
		t.Fatalf("Run = %v", err)
	}
	got := string(out.b)
	if n := strings.Count(got, titlePush); n != 1 {
		t.Fatalf("title pushed %d times, want 1", n)
	}
	if push, set := strings.Index(got, titlePush), strings.Index(got, "\x1b]0;one\x07"); push < 0 || set < push {
		t.Fatalf("push at %d must precede the first title at %d", push, set)
	}
	if !strings.HasSuffix(got, titlePop) || strings.Count(got, titlePop) != 1 {
		t.Fatalf("output must end with one pop; tail %q", got[max(0, len(got)-40):])
	}
}

func TestSetWindowTitlePopsOnTheFixedRestorePath(t *testing.T) {
	out := &unsyncBuffer{}
	p := NewProgram(spinModel{}, WithOutput(out))
	modeApply(p, SetWindowTitle("x"))
	p.restoreViaLoop() // no loop is running, so this is the fixed string
	if got := string(out.b); !strings.HasSuffix(got, titlePop) || strings.Count(got, titlePop) != 1 {
		t.Fatalf("fixed restore must pop the title once; got %q", got)
	}
}

func TestNoTitleSetMeansNoPushOrPop(t *testing.T) {
	p, out, errc := startSpinning(t)
	p.Send(QuitMsg{})
	if err := waitRun(t, errc); err != nil {
		t.Fatal(err)
	}
	if got := string(out.b); strings.Contains(got, titlePush) || strings.Contains(got, titlePop) {
		t.Fatalf("a program that never set a title touched the title stack: %q", got)
	}
}

func TestSetCursorShapeIsWrittenAndRestored(t *testing.T) {
	p, out, errc := startSpinning(t)
	p.Send(SetCursorShape(CursorShapeSteadyBar)())
	p.Send(QuitMsg{})
	if err := waitRun(t, errc); err != nil {
		t.Fatal(err)
	}
	got := string(out.b)
	if !strings.Contains(got, "\x1b[6 q") || !strings.HasSuffix(got, "\x1b[0 q") {
		t.Fatalf("want shape 6 set and default restored last; tail %q", got[max(0, len(got)-40):])
	}
}

// clipModel records every ClipboardMsg and ReplyEvent it sees.
type clipModel struct {
	mu   *sync.Mutex
	clip *[]ClipboardMsg
	rep  *[]ReplyEvent
}

func (m clipModel) Init() Cmd { return nil }
func (m clipModel) Update(msg Msg) (Model, Cmd) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch msg := msg.(type) {
	case ClipboardMsg:
		*m.clip = append(*m.clip, msg)
	case ReplyEvent:
		*m.rep = append(*m.rep, msg)
	}
	return m, nil
}
func (m clipModel) View() string { return "" }

func runClipModel(t *testing.T) (p *Program, in io.Writer, m clipModel, stop func()) {
	t.Helper()
	pr, pw := io.Pipe()
	m = clipModel{mu: &sync.Mutex{}, clip: &[]ClipboardMsg{}, rep: &[]ReplyEvent{}}
	p = NewProgram(m, WithInput(pr), WithOutput(&unsyncBuffer{}))
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	return p, pw, m, func() {
		p.Send(QuitMsg{})
		_ = pw.Close()
		select {
		case <-errc:
		case <-time.After(5 * time.Second):
			t.Error("Run did not return")
		}
	}
}

func waitCond(t *testing.T, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatal("condition not reached")
}

// Criterion #62: a terminal that answers OSC 52 gives the model a ClipboardMsg
// with the decoded text.
func TestReadClipboardDeliversDecodedText(t *testing.T) {
	p, in, m, stop := runClipModel(t)
	defer stop()
	p.Send(ReadClipboard()())
	waitCond(t, func() bool { return p.clipReads.Load() == 1 })
	_, _ = io.WriteString(in, "\x1b]52;c;aGVsbG8gd29ybGQ=\x07")
	waitCond(t, func() bool { m.mu.Lock(); defer m.mu.Unlock(); return len(*m.clip) == 1 })
	if got := (*m.clip)[0]; got.Text != "hello world" || got.Err != nil {
		t.Fatalf("ClipboardMsg = %+v", got)
	}
	if p.clipReads.Load() != 0 {
		t.Fatal("the query was not marked answered")
	}
}

func TestReadClipboardWritesTheQuery(t *testing.T) {
	out := &unsyncBuffer{}
	p := NewProgram(spinModel{}, WithOutput(out))
	modeApply(p, ReadClipboard())
	if got := string(out.b); got != "\x1b]52;c;?\x07" {
		t.Fatalf("query = %q", got)
	}
}

func TestReadClipboardBadReplyIsAnError(t *testing.T) {
	p, in, m, stop := runClipModel(t)
	defer stop()
	p.Send(ReadClipboard()())
	waitCond(t, func() bool { return p.clipReads.Load() == 1 })
	_, _ = io.WriteString(in, "\x1b]52;c;!!not base64!!\x07")
	waitCond(t, func() bool { m.mu.Lock(); defer m.mu.Unlock(); return len(*m.clip) == 1 })
	if (*m.clip)[0].Err == nil {
		t.Fatal("want an error for an undecodable reply")
	}
}

// A reply nobody asked for is not turned into a ClipboardMsg.
func TestUnsolicitedClipboardReplyIsNotDelivered(t *testing.T) {
	_, in, m, stop := runClipModel(t)
	defer stop()
	_, _ = io.WriteString(in, "\x1b]52;c;aGk=\x07")
	waitCond(t, func() bool { m.mu.Lock(); defer m.mu.Unlock(); return len(*m.rep) == 1 })
	if len(*m.clip) != 0 {
		t.Fatalf("an unsolicited reply became %v", *m.clip)
	}
}
