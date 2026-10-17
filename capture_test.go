package tui

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
)

// syncBuf is a buffer a Program and a test can share.
type syncBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

type quietModel struct{}

func (quietModel) Init() Cmd               { return nil }
func (quietModel) Update(Msg) (Model, Cmd) { return quietModel{}, nil }
func (quietModel) View() string            { return "live" }

// captureEnv replaces os.Stdout and os.Stderr with temp files for the test, so
// there is a known "terminal" to check, and puts the real ones back.
type captureEnv struct {
	out, err *os.File
}

func newCaptureEnv(t *testing.T) captureEnv {
	t.Helper()
	dir := t.TempDir()
	out, e1 := os.Create(dir + "/stdout")
	errf, e2 := os.Create(dir + "/stderr")
	if e1 != nil || e2 != nil {
		t.Fatal(e1, e2)
	}
	oldOut, oldErr, oldLog := os.Stdout, os.Stderr, log.Writer()
	os.Stdout, os.Stderr = out, errf
	log.SetOutput(errf) // the logger holds the Stderr it was given, as the standard one does
	t.Cleanup(func() {
		os.Stdout, os.Stderr = oldOut, oldErr
		log.SetOutput(oldLog)
		out.Close()
		errf.Close()
	})
	return captureEnv{out: out, err: errf}
}

func (e captureEnv) read(t *testing.T, f *os.File) string {
	t.Helper()
	b, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// startCaptured runs a Program with WithCaptureOutput until the test quits it.
func startCaptured(t *testing.T) (p *Program, screen, errScreen *syncBuf, stop func() error) {
	t.Helper()
	pr, pw := io.Pipe()
	screen, errScreen = &syncBuf{}, &syncBuf{}
	p = NewProgram(quietModel{}, WithInput(pr), WithOutput(screen), WithErrOutput(errScreen),
		WithCaptureOutput(true), WithAltScreen(false), WithReducedMotion(true), WithColorProfile(ansi.TrueColor))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	deadline := time.Now().Add(10 * time.Second)
	for p.cap.Load() == nil {
		if time.Now().After(deadline) {
			t.Fatal("the capture never started")
		}
		time.Sleep(time.Millisecond)
	}
	return p, screen, errScreen, func() error {
		p.Quit()
		pw.Close()
		return <-done
	}
}

func capWait(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestStrayPrintsAreCommittedAboveTheFrame(t *testing.T) {
	env := newCaptureEnv(t)
	_, screen, errScreen, stop := startCaptured(t)
	fmt.Println("hello from stdout")
	fmt.Fprintln(os.Stderr, "oops on stderr")
	log.Printf("logged %d", 1)
	capWait(t, "stdout text", func() bool { return strings.Contains(screen.String(), "hello from stdout") })
	capWait(t, "stderr and log text", func() bool {
		s := errScreen.String()
		return strings.Contains(s, "oops on stderr") && strings.Contains(s, "logged 1")
	})
	if err := stop(); err != nil {
		t.Fatal(err)
	}
	// Nothing reached the real files while the program was running, and they are back.
	if os.Stdout != env.out || os.Stderr != env.err {
		t.Error("os.Stdout or os.Stderr was not put back")
	}
	if got := env.read(t, env.out); strings.Contains(got, "hello from stdout") {
		t.Errorf("the print also reached the real stdout: %q", got)
	}
	log.Printf("after") // the logger is back on the real stderr
	if got := env.read(t, env.err); !strings.Contains(got, "after") || strings.Contains(got, "logged 1") {
		t.Errorf("real stderr = %q", got)
	}
}

func TestCapturedLinesKeepTheirOrderAndStylingButNotTerminalCodes(t *testing.T) {
	newCaptureEnv(t)
	_, screen, _, stop := startCaptured(t)
	for i := 0; i < 300; i++ {
		fmt.Printf("line %03d\n", i)
	}
	fmt.Println("\x1b[2Jcleared\x1b[31m red\x1b[0m")
	capWait(t, "all lines", func() bool {
		return strings.Contains(screen.String(), "cleared") && strings.Contains(screen.String(), "line 299")
	})
	if err := stop(); err != nil {
		t.Fatal(err)
	}
	got := screen.String()
	last := -1
	for i := 0; i < 300; i++ {
		at := strings.Index(got, fmt.Sprintf("line %03d", i))
		if at < 0 || at < last {
			t.Fatalf("line %d is missing or out of order", i)
		}
		last = at
	}
	if strings.Contains(got, "\x1b[2J") {
		t.Error("a clear-screen sequence got through")
	}
	if !strings.Contains(got, "\x1b[31m") {
		t.Error("the styling was removed too")
	}
}

func TestPartialAndLongLinesAreNotLost(t *testing.T) {
	newCaptureEnv(t)
	_, screen, _, stop := startCaptured(t)
	fmt.Print("no newline at the end")
	fmt.Println()
	fmt.Println(strings.Repeat("x", 3*captureChunk/2))
	fmt.Print("tail without newline")
	capWait(t, "the long line", func() bool { return strings.Count(screen.String(), "x") >= 3*captureChunk/2 })
	if err := stop(); err != nil {
		t.Fatal(err)
	}
	if got := screen.String(); strings.Count(got, "x") < 3*captureChunk/2 {
		t.Errorf("the long line lost %d characters", 3*captureChunk/2-strings.Count(got, "x"))
	}
}

func TestLinesNotCommittedAreWrittenToTheRealFilesWhenRunReturns(t *testing.T) {
	env := newCaptureEnv(t)
	_, _, _, stop := startCaptured(t)
	if err := stop(); err != nil {
		t.Fatal(err)
	}
	// The loop is over: a capture started now has nowhere to commit to, so a
	// line goes straight to the file.
	p := NewProgram(quietModel{}, WithOutput(io.Discard))
	p.started.Store(true)
	p.setLoopDone(nil)
	endCapture := p.startCapture()
	fmt.Println("late line")
	endCapture()
	if got := env.read(t, env.out); !strings.Contains(got, "late line") {
		t.Errorf("real stdout = %q, want the late line", got)
	}
}

func TestTooManyPendingLinesDropTheOldestAndSayHowMany(t *testing.T) {
	c := &capture{wake: make(chan struct{}, 1)}
	for i := 0; i < capturePendingMax+25; i++ {
		c.add(fmt.Sprintf("n%d", i), false)
	}
	got := c.take()
	if len(got) != capturePendingMax+1 {
		t.Fatalf("take returned %d lines, want %d plus a notice", len(got), capturePendingMax)
	}
	if !strings.Contains(got[0].text, "25 captured lines were dropped") || !got[0].stderr {
		t.Errorf("notice = %+v", got[0])
	}
	if got[1].text != "n25" || got[len(got)-1].text != fmt.Sprintf("n%d", capturePendingMax+24) {
		t.Errorf("kept lines %q ... %q", got[1].text, got[len(got)-1].text)
	}
	if rest := c.take(); len(rest) != 0 {
		t.Errorf("a second take returned %d lines", len(rest))
	}
}

// stdoutProbe records os.Stdout from inside the event loop: during a Suspend
// function (typing 'e') and on the next key ('f'). It reads os.Stdout only on
// the loop's own goroutine, which is also the one that swaps it.
type stdoutProbe struct {
	during, after *os.File
	ran, seen     chan struct{} // closed once the Suspend function has run, and once 'f' has been handled
}

type probeModel struct{ probe *stdoutProbe }

func (probeModel) Init() Cmd { return nil }
func (m probeModel) Update(msg Msg) (Model, Cmd) {
	if k, ok := msg.(Key); ok {
		switch k.Text {
		case "e":
			return m, Suspend(func() error { m.probe.during = os.Stdout; close(m.probe.ran); return nil })
		case "f":
			m.probe.after = os.Stdout
			close(m.probe.seen)
		}
	}
	return m, nil
}
func (probeModel) View() string { return "live" }

func TestSuspendGivesTheTerminalBackWhileItRuns(t *testing.T) {
	env := newCaptureEnv(t)
	pr, pw := io.Pipe()
	probe := &stdoutProbe{ran: make(chan struct{}), seen: make(chan struct{})}
	p := NewProgram(probeModel{probe: probe}, WithInput(pr), WithOutput(&syncBuf{}), WithCaptureOutput(true), WithAltScreen(false), WithReducedMotion(true))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	capWait(t, "the capture", func() bool { return p.cap.Load() != nil })
	pw.Write([]byte("e"))
	select {
	case <-probe.ran:
	case <-time.After(10 * time.Second):
		t.Fatal("the Suspend function never ran")
	}
	pw.Write([]byte("f"))
	select {
	case <-probe.seen:
	case <-time.After(10 * time.Second):
		t.Fatal("the second key was never handled")
	}
	p.Quit()
	pw.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if probe.during != env.out {
		t.Error("os.Stdout was still the capture pipe inside a Suspend function")
	}
	if probe.after == env.out || probe.after == nil {
		t.Error("the capture was not resumed after the Suspend function")
	}
	if os.Stdout != env.out {
		t.Error("os.Stdout was not put back when Run returned")
	}
}

func TestOptionOffLeavesStdoutAlone(t *testing.T) {
	env := newCaptureEnv(t)
	pr, pw := io.Pipe()
	p := NewProgram(quietModel{}, WithInput(pr), WithOutput(&syncBuf{}), WithAltScreen(false), WithReducedMotion(true))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	capWait(t, "the loop", func() bool { d, _ := p.loopState(); return d != nil })
	if os.Stdout != env.out || p.cap.Load() != nil {
		t.Error("os.Stdout changed without WithCaptureOutput")
	}
	p.Quit()
	pw.Close()
	<-done
}
