package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
)

func TestFocusReportingOffEmitsNothing(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{}, WithOutput(out))
	p.enterModes()
	p.leaveModes()
	if got := string(read()); strings.Contains(got, "?1004") {
		t.Errorf("focus sequence emitted without WithFocusReporting: %q", got)
	}
}

func TestFocusReportingEnabledOnEnterDisabledFirstOnLeave(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{}, WithOutput(out), WithFocusReporting(true), WithBracketedPaste(true))
	p.enterModes()
	got := string(read())
	if !strings.Contains(got, ansi.FocusReportingEnable) {
		t.Fatalf("enterModes did not enable focus reporting: %q", got)
	}
	if strings.Index(got, ansi.BracketedPasteEnable) > strings.Index(got, ansi.FocusReportingEnable) {
		t.Errorf("focus reporting should be enabled after bracketed paste: %q", got)
	}

	before := len(read())
	p.leaveModes()
	left := string(read()[before:])
	f, b := strings.Index(left, ansi.FocusReportingDisable), strings.Index(left, ansi.BracketedPasteDisable)
	if f < 0 || b < 0 || f > b {
		t.Errorf("leaveModes should disable focus reporting first (reverse of enable): %q", left)
	}
}

func TestFocusReportingSurvivesSuspendAndRestore(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{}, WithOutput(out), WithFocusReporting(true))
	var during string
	if err := p.suspend(func() error { during = string(read()); return nil }); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(during, ansi.FocusReportingDisable) {
		t.Errorf("focus reporting not disabled before fn ran: %q", during)
	}
	after := string(read())[len(during):]
	if !strings.Contains(after, ansi.FocusReportingEnable) {
		t.Errorf("focus reporting not re-enabled after Suspend: %q", after)
	}

	before := len(read())
	p.restoreTerminal()
	if !strings.Contains(string(read()[before:]), ansi.FocusReportingDisable) {
		t.Error("restoreTerminal did not disable focus reporting")
	}
}

// bgRun starts runLoop against a pipe, returning the pipe's write end, the
// captured output, and a way to collect the result.
func bgRun(t *testing.T, m Model, opts ...ProgramOption) (write func(string), read func() []byte, wait func() runResult) {
	t.Helper()
	pr, pw := mustPipe(t)
	t.Cleanup(func() { pr.Close(); pw.Close() })
	out, readOut := captureOutput(t)
	p := NewProgram(m, append([]ProgramOption{WithInput(pr), WithOutput(out)}, opts...)...)
	ch := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		ch <- runResult{m, err}
	}()
	return func(s string) {
			if _, err := pw.Write([]byte(s)); err != nil {
				t.Fatalf("write: %v", err)
			}
		}, readOut, func() runResult {
			select {
			case r := <-ch:
				return r
			case <-time.After(3 * time.Second):
				t.Fatal("runLoop did not return in time")
				return runResult{}
			}
		}
}

func TestBackgroundDetectionQueriesOnceAndDeliversReply(t *testing.T) {
	write, read, wait := bgRun(t, recorderModel{quitAfter: 2}, WithBackgroundDetection(time.Hour))
	write("\x1b]11;rgb:ffff/ffff/ffff\x07")
	rec := wait().model.(recorderModel)
	ev, ok := rec.received[1].(BackgroundColorEvent)
	if !ok || ev != (BackgroundColorEvent{R: 255, G: 255, B: 255}) {
		t.Fatalf("message 1 = %#v, want BackgroundColorEvent{255 255 255}", rec.received[1])
	}
	if n := strings.Count(string(read()), ansi.QueryBackgroundColor); n != 1 {
		t.Errorf("query sent %d times, want 1", n)
	}
}

func TestBackgroundDetectionTimeoutSendsUnknownOnce(t *testing.T) {
	_, _, wait := bgRun(t, recorderModel{quitAfter: 2}, WithBackgroundDetection(50*time.Millisecond))
	rec := wait().model.(recorderModel)
	if _, ok := rec.received[1].(BackgroundUnknownMsg); !ok {
		t.Fatalf("message 1 = %#v, want BackgroundUnknownMsg", rec.received[1])
	}
}

func TestBackgroundDetectionNoUnknownAfterReply(t *testing.T) {
	write, _, wait := bgRun(t, recorderModel{quitAfter: 3}, WithBackgroundDetection(100*time.Millisecond))
	write("\x1b]11;rgb:0000/0000/0000\x1b\\")
	time.Sleep(300 * time.Millisecond) // well past the timeout
	write("a")
	rec := wait().model.(recorderModel)
	for i, m := range rec.received {
		if _, bad := m.(BackgroundUnknownMsg); bad {
			t.Fatalf("message %d is BackgroundUnknownMsg although a reply arrived: %#v", i, rec.received)
		}
	}
	if _, ok := rec.received[1].(BackgroundColorEvent); !ok {
		t.Errorf("message 1 = %#v, want BackgroundColorEvent", rec.received[1])
	}
}

func TestBackgroundDetectionOffSendsNothing(t *testing.T) {
	write, read, wait := bgRun(t, recorderModel{quitAfter: 2})
	write("a")
	wait()
	if strings.Contains(string(read()), "]11;?") {
		t.Errorf("background query sent without WithBackgroundDetection: %q", read())
	}
}

func TestBackgroundDetectionWatcherExitsWithRunLoop(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	out, _ := captureOutput(t)
	p := NewProgram(quitOnInitModel{}, WithInput(pr), WithOutput(out), WithBackgroundDetection(time.Hour))
	runLoopWithTimeout(t, p, 2*time.Second)

	done := make(chan struct{})
	go func() { p.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("background-detection goroutine outlived runLoop")
	}
}
