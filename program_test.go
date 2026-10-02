package tui

import (
	"context"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
)

func mustPipe(t *testing.T) (r, w *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	return r, w
}

func mustDevNull(t *testing.T) *os.File {
	t.Helper()
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	return f
}

// runResult carries runLoop's return values across a goroutine boundary.
type runResult struct {
	model Model
	err   error
}

// runLoopWithTimeout starts p.runLoop on its own goroutine and fails the
// test if it doesn't return within d — a stuck event loop is exactly the
// class of bug these tests exist to catch.
func runLoopWithTimeout(t *testing.T, p *Program, d time.Duration) runResult {
	t.Helper()
	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()

	select {
	case res := <-resultCh:
		return res
	case <-time.After(d):
		t.Fatal("runLoop did not return in time")
		return runResult{}
	}
}

// --- fixture models ---

type quitOnInitModel struct{}

func (quitOnInitModel) Init() Cmd                 { return Quit() }
func (m quitOnInitModel) Update(Msg) (Model, Cmd) { return m, nil }
func (quitOnInitModel) View() string              { return "" }

type quitOnKeyModel struct{}

func (quitOnKeyModel) Init() Cmd { return nil }
func (m quitOnKeyModel) Update(msg Msg) (Model, Cmd) {
	if _, ok := msg.(Key); ok {
		return m, Quit()
	}
	return m, nil
}
func (quitOnKeyModel) View() string { return "" }

// recorderModel appends every Msg it sees, in order, and quits once it has
// recorded quitAfter of them.
type recorderModel struct {
	received  []Msg
	quitAfter int
}

func (m recorderModel) Init() Cmd { return nil }
func (m recorderModel) Update(msg Msg) (Model, Cmd) {
	m.received = append(m.received, msg)
	if len(m.received) >= m.quitAfter {
		return m, Quit()
	}
	return m, nil
}
func (recorderModel) View() string { return "" }

// fpsCounterModel counts View() calls via a shared pointer (Model is
// value-copied by Update, so a pointer is what lets the count survive
// those copies), and records the received-Msg count each View() call saw
// so a test can tell whether an eventual render painted stale or current
// state. It quits once it has received quitAfter non-Resize Msgs,
// mirroring recorderModel's shape.
type fpsCounterModel struct {
	viewCount     *int32
	lastViewValue *int32
	received      int
	quitAfter     int
}

func (m fpsCounterModel) Init() Cmd { return nil }
func (m fpsCounterModel) Update(msg Msg) (Model, Cmd) {
	if _, ok := msg.(ResizeMsg); ok {
		return m, nil
	}
	m.received++
	if m.received >= m.quitAfter {
		return m, Quit()
	}
	return m, nil
}
func (m fpsCounterModel) View() string {
	atomic.AddInt32(m.viewCount, 1)
	atomic.StoreInt32(m.lastViewValue, int32(m.received))
	return strconv.Itoa(m.received)
}

type customMsgA struct{}
type customMsgB struct{}

type batchInitModel struct {
	recorderModel
}

func (m batchInitModel) Init() Cmd {
	return Batch(
		func() Msg { return customMsgA{} },
		func() Msg { return customMsgB{} },
	)
}

// Update is defined explicitly rather than relying on recorderModel's
// promoted method: a promoted method returns the embedded type
// (recorderModel), not the outer composite (batchInitModel), which would
// silently swap the model's dynamic type out from under the loop after the
// first non-Init message.
func (m batchInitModel) Update(msg Msg) (Model, Cmd) {
	next, cmd := m.recorderModel.Update(msg)
	m.recorderModel = next.(recorderModel)
	return m, cmd
}

// --- tests ---

func TestRunLoopQuitFromInit(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	devnull := mustDevNull(t)
	defer devnull.Close()

	p := NewProgram(quitOnInitModel{}, WithInput(pr), WithOutput(devnull))
	res := runLoopWithTimeout(t, p, 2*time.Second)
	if res.err != nil {
		t.Fatalf("runLoop returned error: %v", res.err)
	}
}

func TestRunLoopQuitFromKeypress(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	devnull := mustDevNull(t)
	defer devnull.Close()

	p := NewProgram(quitOnKeyModel{}, WithInput(pr), WithOutput(devnull))

	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()

	if _, err := pw.Write([]byte("q")); err != nil {
		t.Fatalf("write to pipe: %v", err)
	}

	select {
	case res := <-resultCh:
		if res.err != nil {
			t.Fatalf("runLoop returned error: %v", res.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not quit after a keypress")
	}
}

func TestRunLoopSeedsInitialResize(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	devnull := mustDevNull(t)
	defer devnull.Close()

	p := NewProgram(recorderModel{quitAfter: 1}, WithInput(pr), WithOutput(devnull))
	p.width, p.height = 80, 24 // normally set by Run() via term.GetSize before runLoop

	res := runLoopWithTimeout(t, p, 2*time.Second)
	if res.err != nil {
		t.Fatalf("runLoop returned error: %v", res.err)
	}
	rec := res.model.(recorderModel)
	if len(rec.received) != 1 {
		t.Fatalf("got %d messages, want 1", len(rec.received))
	}
	rm, ok := rec.received[0].(ResizeMsg)
	if !ok {
		t.Fatalf("first message = %#v, want ResizeMsg", rec.received[0])
	}
	if rm.Width != 80 || rm.Height != 24 {
		t.Errorf("ResizeMsg = %+v, want {80 24}", rm)
	}
}

func TestRunLoopDeliversKeypressesInOrder(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	devnull := mustDevNull(t)
	defer devnull.Close()

	// quitAfter 3: the seeded ResizeMsg, then two keys.
	p := NewProgram(recorderModel{quitAfter: 3}, WithInput(pr), WithOutput(devnull))

	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()

	if _, err := pw.Write([]byte("ab")); err != nil {
		t.Fatalf("write to pipe: %v", err)
	}

	var res runResult
	select {
	case res = <-resultCh:
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return in time")
	}
	if res.err != nil {
		t.Fatalf("runLoop returned error: %v", res.err)
	}

	rec := res.model.(recorderModel)
	if len(rec.received) != 3 {
		t.Fatalf("got %d messages, want 3: %#v", len(rec.received), rec.received)
	}
	if _, ok := rec.received[0].(ResizeMsg); !ok {
		t.Errorf("message 0 = %#v, want ResizeMsg", rec.received[0])
	}
	wantRunes := []rune{'a', 'b'}
	for i, want := range wantRunes {
		k, ok := rec.received[i+1].(Key)
		if !ok || k.Type != KeyRunes || k.Text != string(want) {
			t.Errorf("message %d = %#v, want rune %q", i+1, rec.received[i+1], want)
		}
	}
}

func TestRunLoopDispatchesAllBatchedCmds(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	devnull := mustDevNull(t)
	defer devnull.Close()

	// quitAfter 3: the seeded ResizeMsg, then both batched custom messages.
	p := NewProgram(batchInitModel{recorderModel{quitAfter: 3}}, WithInput(pr), WithOutput(devnull))
	res := runLoopWithTimeout(t, p, 2*time.Second)
	if res.err != nil {
		t.Fatalf("runLoop returned error: %v", res.err)
	}

	rec := res.model.(batchInitModel).recorderModel
	var gotA, gotB bool
	for _, msg := range rec.received {
		switch msg.(type) {
		case customMsgA:
			gotA = true
		case customMsgB:
			gotB = true
		}
	}
	if !gotA || !gotB {
		t.Errorf("Batch didn't deliver both messages: %#v", rec.received)
	}
}

// TestDispatchUnblocksOnDone is a regression test for the goroutine-leak
// fix: once done is closed (as it is right after runLoop returns), any
// dispatch trying to feed a result back into msgs must return immediately
// rather than block forever on an unbuffered/full channel that nothing
// drains anymore. This fires far more sends than msgs' buffer capacity, so
// most of them are structurally forced through the done branch to avoid
// hanging.
func TestDispatchUnblocksOnDone(t *testing.T) {
	p := &Program{msgs: make(chan Msg, 4)}
	done := make(chan struct{})
	close(done) // simulate Run/runLoop having already returned

	const n = 200
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.dispatch(func() Msg { return QuitMsg{} }, done)
		}()
	}

	finished := make(chan struct{})
	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("dispatch calls did not return after done was closed — goroutine leak")
	}
}

// TestRunLoopStopsBackgroundGoroutinesOnQuit is the direct regression test
// for last round's fix: the reader and resize-watcher goroutines spawned by
// runLoop must actually exit once runLoop returns, not just stop mattering.
func TestRunLoopStopsBackgroundGoroutinesOnQuit(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	devnull := mustDevNull(t)
	defer devnull.Close()

	p := NewProgram(quitOnInitModel{}, WithInput(pr), WithOutput(devnull))
	res := runLoopWithTimeout(t, p, 2*time.Second)
	if res.err != nil {
		t.Fatalf("runLoop returned error: %v", res.err)
	}

	waited := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(waited)
	}()

	select {
	case <-waited:
	case <-time.After(2 * time.Second):
		t.Fatal("background goroutines (reader/sigwinch) did not exit after runLoop returned — leak")
	}
}

func TestRunLoopDeliversInputErrorOnEOF(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	devnull := mustDevNull(t)
	defer devnull.Close()

	p := NewProgram(recorderModel{quitAfter: 2}, WithInput(pr), WithOutput(devnull))
	pw.Close() // reader sees io.EOF

	res := runLoopWithTimeout(t, p, 2*time.Second)
	got := res.model.(recorderModel).received
	if len(got) != 2 {
		t.Fatalf("received %d msgs (%v), want ResizeMsg + InputErrorMsg", len(got), got)
	}
	ie, ok := got[1].(InputErrorMsg)
	if !ok {
		t.Fatalf("second msg = %T, want InputErrorMsg", got[1])
	}
	if !errors.Is(ie.Err, io.EOF) {
		t.Errorf("InputErrorMsg.Err = %v, want io.EOF", ie.Err)
	}
}

func TestRunLoopDeliversInputErrorOnReadFailure(t *testing.T) {
	// A write-only file is "readable" to select/poll but every read on it
	// fails (EBADF, or access denied on windows): a deterministic read error
	// that is not EOF, without closing a descriptor out from under the reader.
	wo, err := os.OpenFile(t.TempDir()+"/writeonly", os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer wo.Close()
	devnull := mustDevNull(t)
	defer devnull.Close()

	p := NewProgram(recorderModel{quitAfter: 2}, WithInput(wo), WithOutput(devnull))
	res := runLoopWithTimeout(t, p, 2*time.Second)
	got := res.model.(recorderModel).received
	ie, ok := got[len(got)-1].(InputErrorMsg)
	if !ok {
		t.Fatalf("last msg = %T, want InputErrorMsg", got[len(got)-1])
	}
	if ie.Err == nil || errors.Is(ie.Err, io.EOF) {
		t.Errorf("InputErrorMsg.Err = %v, want a read failure other than EOF", ie.Err)
	}
}

func TestRunLoopKeepsRunningAfterInputErrorAndReportsOnce(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	devnull := mustDevNull(t)
	defer devnull.Close()

	p := NewProgram(recorderModel{quitAfter: 3}, WithInput(pr), WithOutput(devnull))
	pw.Close()

	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()

	select {
	case <-resultCh:
		t.Fatal("runLoop ended after InputErrorMsg; the model should decide")
	case <-time.After(200 * time.Millisecond):
	}

	p.msgs <- customMsgA{}
	select {
	case res := <-resultCh:
		got := res.model.(recorderModel).received
		if len(got) != 3 {
			t.Fatalf("received %v, want [ResizeMsg InputErrorMsg customMsgA]", got)
		}
		if _, ok := got[1].(InputErrorMsg); !ok {
			t.Errorf("msg 1 = %T, want InputErrorMsg", got[1])
		}
		if _, ok := got[2].(customMsgA); !ok {
			t.Errorf("msg 2 = %T, want customMsgA (a second InputErrorMsg means the reader retried)", got[2])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return after the follow-up msg")
	}
}

// Shutting down unblocks the reader with a read deadline; that must not
// surface as an InputErrorMsg.
func TestRunLoopShutdownDoesNotReportInputError(t *testing.T) {
	for i := 0; i < 20; i++ {
		pr, pw := mustPipe(t)
		devnull := mustDevNull(t)

		p := NewProgram(quitOnInitModel{}, WithInput(pr), WithOutput(devnull))
		runLoopWithTimeout(t, p, 2*time.Second)
		p.wg.Wait()

		select {
		case m := <-p.msgs:
			t.Errorf("iteration %d: leftover msg %T after shutdown, want none", i, m)
		default:
		}
		pr.Close()
		pw.Close()
		devnull.Close()
	}
}

func TestRunRejectsNonTerminalInput(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	out, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	p := NewProgram(quitOnInitModel{}, WithInput(pr), WithOutput(out))
	_, runErr := p.Run()
	if runErr == nil || !strings.Contains(runErr.Error(), "not a terminal") {
		t.Errorf("Run error = %v, want \"not a terminal\"", runErr)
	}
	if fi, _ := out.Stat(); fi.Size() != 0 {
		t.Errorf("Run wrote %d bytes to the output before failing, want none", fi.Size())
	}
}

// TestContextNonNilImmediatelyAfterNewProgram proves Program.Context is
// usable before Run is ever called — the moment described in WithContext's
// doc comment for threading it into a model — not just after Run starts.
func TestContextNonNilImmediatelyAfterNewProgram(t *testing.T) {
	p := NewProgram(staticModel{})
	ctx := p.Context()
	if ctx == nil {
		t.Fatal("Context() returned nil before Run was called")
	}
	if err := ctx.Err(); err != nil {
		t.Errorf("ctx.Err() = %v, want nil (not yet cancelled)", err)
	}
}

// TestReducedMotionDefaultsToMotionDetect proves ReducedMotion falls back
// to motion.Detect() (the NO_ANIMATION environment variable) when
// WithReducedMotion is never used, so existing motion.Preference-aware
// code and a Program's own preference stay in sync automatically.
func TestReducedMotionDefaultsToMotionDetect(t *testing.T) {
	t.Setenv("NO_ANIMATION", "")
	if p := NewProgram(staticModel{}); p.ReducedMotion() {
		t.Error("ReducedMotion() = true with NO_ANIMATION unset, want false")
	}

	t.Setenv("NO_ANIMATION", "1")
	if p := NewProgram(staticModel{}); !p.ReducedMotion() {
		t.Error("ReducedMotion() = false with NO_ANIMATION=1, want true")
	}
}

// TestWithReducedMotionOverridesDefault proves the explicit option wins
// over the environment-variable default in both directions.
func TestWithReducedMotionOverridesDefault(t *testing.T) {
	t.Setenv("NO_ANIMATION", "1")
	if p := NewProgram(staticModel{}, WithReducedMotion(false)); p.ReducedMotion() {
		t.Error("WithReducedMotion(false) did not override NO_ANIMATION=1")
	}

	t.Setenv("NO_ANIMATION", "")
	if p := NewProgram(staticModel{}, WithReducedMotion(true)); !p.ReducedMotion() {
		t.Error("WithReducedMotion(true) did not override an unset NO_ANIMATION")
	}
}

func TestMouseModeCodes(t *testing.T) {
	tests := []struct {
		mode            MouseMode
		enable, disable string
	}{
		{MouseClick, ansi.MouseClickEnable, ansi.MouseClickDisable},
		{MouseCellMotion, ansi.MouseCellMotionEnable, ansi.MouseCellMotionDisable},
		{MouseAllMotion, ansi.MouseAllMotionEnable, ansi.MouseAllMotionDisable},
		{MouseOff, "", ""},
		{MouseMode(99), "", ""},
		{MouseMode(-1), "", ""},
	}
	for _, tt := range tests {
		en, dis := mouseModeCodes(tt.mode)
		if en != tt.enable || dis != tt.disable {
			t.Errorf("mouseModeCodes(%d) = %q, %q; want %q, %q", tt.mode, en, dis, tt.enable, tt.disable)
		}
	}
}

func TestTickWaitsThenBuildsTheMessage(t *testing.T) {
	type tickMsg struct{ at time.Time }
	const d = 40 * time.Millisecond

	called := false
	cmd := Tick(d, func(now time.Time) Msg {
		called = true
		return tickMsg{at: now}
	})
	if called {
		t.Fatal("Tick ran fn before the Cmd was executed")
	}

	start := time.Now()
	msg := RunCmd(context.Background(), cmd)
	elapsed := time.Since(start)
	if elapsed < d {
		t.Errorf("Cmd returned after %v, want at least %v", elapsed, d)
	}
	got, ok := msg.(tickMsg)
	if !ok {
		t.Fatalf("Cmd produced %T, want the message built by fn", msg)
	}
	if got.at.Before(start.Add(d - 5*time.Millisecond)) {
		t.Errorf("fn received time %v, want the firing time (>= %v)", got.at, start.Add(d))
	}
}

// TestWithMaxFPSUnsetRendersEveryMsg proves criterion #624: with
// WithMaxFPS never called, render() runs on every Msg exactly as before
// the option existed (the seed render, plus one per key).
func TestWithMaxFPSUnsetRendersEveryMsg(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	out, _ := captureOutput(t)

	var viewCount, lastView int32
	const nKeys = 5
	p := NewProgram(fpsCounterModel{viewCount: &viewCount, lastViewValue: &lastView, quitAfter: nKeys}, WithInput(pr), WithOutput(out))

	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()
	if _, err := pw.Write([]byte("abcde")); err != nil {
		t.Fatalf("write to pipe: %v", err)
	}
	select {
	case res := <-resultCh:
		if res.err != nil {
			t.Fatalf("runLoop returned error: %v", res.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return in time")
	}

	if got := atomic.LoadInt32(&viewCount); got != nKeys+1 {
		t.Errorf("viewCount = %d, want %d (seed render + one per key, no throttling)", got, nKeys+1)
	}
}

// TestWithMaxFPSThrottlesBurstThenPaintsLatestStateOnNextMsg proves
// criteria #625 and #626: a rapid burst under the cap mostly skips
// repainting (Update still runs for every Msg), and once enough time has
// passed by the time a later Msg arrives, that Msg's render paints the
// model's current (not stale) state.
func TestWithMaxFPSThrottlesBurstThenPaintsLatestStateOnNextMsg(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	out, _ := captureOutput(t)

	var viewCount, lastView int32
	// quitAfter is well above what this test sends before its final
	// cleanup burst, so the burst under test doesn't quit early.
	p := NewProgram(fpsCounterModel{viewCount: &viewCount, lastViewValue: &lastView, quitAfter: 100}, WithInput(pr), WithOutput(out), WithMaxFPS(5)) // 200ms interval

	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()

	time.Sleep(20 * time.Millisecond) // let the seed render happen first
	afterSeed := atomic.LoadInt32(&viewCount)

	if _, err := pw.Write([]byte("abcde")); err != nil { // a rapid burst, all well within 200ms
		t.Fatalf("write to pipe: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	duringBurst := atomic.LoadInt32(&viewCount)
	if duringBurst-afterSeed > 2 {
		t.Errorf("viewCount grew by %d during a rapid burst under the fps cap, want at most 1-2 (mostly throttled)", duringBurst-afterSeed)
	}

	time.Sleep(250 * time.Millisecond) // past the 200ms interval
	if _, err := pw.Write([]byte("f")); err != nil {
		t.Fatalf("write to pipe: %v", err)
	}
	// The trailing flush of the burst may have just drawn a frame, so "f" can
	// itself be throttled; it is drawn within one more interval.
	time.Sleep(250 * time.Millisecond)

	if got := atomic.LoadInt32(&viewCount); got <= duringBurst {
		t.Fatal("viewCount did not grow after waiting past the interval and sending another Msg (no trailing-edge flush)")
	}
	if got := atomic.LoadInt32(&lastView); got != 6 {
		t.Errorf("last rendered received-count = %d, want 6 (the latest state, not a stale mid-burst snapshot)", got)
	}

	if _, err := pw.Write([]byte(strings.Repeat("x", 94))); err != nil { // reach quitAfter to end the loop cleanly
		t.Fatalf("write to pipe: %v", err)
	}
	select {
	case res := <-resultCh:
		if res.err != nil {
			t.Fatalf("runLoop returned error: %v", res.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return in time")
	}
}

// TestWithMaxFPSNonPositiveIsUnthrottled proves criterion #627: a
// non-positive fps behaves exactly like WithMaxFPS never being called.
func TestWithMaxFPSNonPositiveIsUnthrottled(t *testing.T) {
	for _, fps := range []int{0, -1, -100} {
		t.Run(strconv.Itoa(fps), func(t *testing.T) {
			pr, pw := mustPipe(t)
			defer pr.Close()
			defer pw.Close()
			out, _ := captureOutput(t)

			var viewCount, lastView int32
			const nKeys = 3
			p := NewProgram(fpsCounterModel{viewCount: &viewCount, lastViewValue: &lastView, quitAfter: nKeys}, WithInput(pr), WithOutput(out), WithMaxFPS(fps))

			resultCh := make(chan runResult, 1)
			go func() {
				m, err := p.runLoop()
				resultCh <- runResult{m, err}
			}()
			if _, err := pw.Write([]byte("abc")); err != nil {
				t.Fatalf("write to pipe: %v", err)
			}
			select {
			case res := <-resultCh:
				if res.err != nil {
					t.Fatalf("runLoop returned error: %v", res.err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("runLoop did not return in time")
			}

			if got := atomic.LoadInt32(&viewCount); got != nKeys+1 {
				t.Errorf("fps=%d: viewCount = %d, want %d (unthrottled)", fps, got, nKeys+1)
			}
		})
	}
}

// TestWithMaxFPSQuitAlwaysRendersLatestFrame proves criterion #628: even
// under a very slow cap, processing QuitMsg forces a render reflecting
// the model's final state, rather than leaving a throttled-away stale
// frame as the last thing painted.
func TestWithMaxFPSQuitAlwaysRendersLatestFrame(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	out, _ := captureOutput(t)

	var viewCount, lastView int32
	const nKeys = 3
	// 1fps (1s interval): without the QuitMsg bypass, the 2nd and 3rd
	// keys in this near-instant burst would both be throttled away and
	// the final frame would be stale at "1" instead of "3".
	p := NewProgram(fpsCounterModel{viewCount: &viewCount, lastViewValue: &lastView, quitAfter: nKeys}, WithInput(pr), WithOutput(out), WithMaxFPS(1))

	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()
	if _, err := pw.Write([]byte("abc")); err != nil { // the 3rd key reaches quitAfter
		t.Fatalf("write to pipe: %v", err)
	}
	select {
	case res := <-resultCh:
		if res.err != nil {
			t.Fatalf("runLoop returned error: %v", res.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return in time (QuitMsg's forced render may be missing)")
	}

	if got := atomic.LoadInt32(&lastView); got != nKeys {
		t.Errorf("last rendered received-count = %d, want %d (quit must render the final state, bypassing the 1fps cap)", got, nKeys)
	}
}

// TestWithMaxFPSDrawsThrottledFinalFrameWhenIdle proves the trailing-edge
// flush: a burst whose last renders are throttled, followed by silence, still
// ends with the latest state on screen within one frame interval, drawn once.
func TestWithMaxFPSDrawsThrottledFinalFrameWhenIdle(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	out, _ := captureOutput(t)

	var viewCount, lastView int32
	p := NewProgram(fpsCounterModel{viewCount: &viewCount, lastViewValue: &lastView, quitAfter: 100}, WithInput(pr), WithOutput(out), WithMaxFPS(10)) // 100ms interval

	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()

	time.Sleep(20 * time.Millisecond) // seed render
	if _, err := pw.Write([]byte("abcde")); err != nil {
		t.Fatalf("write to pipe: %v", err)
	}

	// Well past one interval, with no further message: the last state is on screen.
	time.Sleep(300 * time.Millisecond)
	if got := atomic.LoadInt32(&lastView); got != 5 {
		t.Fatalf("last drawn state = %d, want 5 (the throttled final frame was never drawn)", got)
	}
	settled := atomic.LoadInt32(&viewCount)
	if settled > 4 {
		t.Errorf("viewCount = %d for a 5-key burst in one interval, want at most seed + first + one flush", settled)
	}

	// Idle: nothing pending, so nothing more is drawn.
	time.Sleep(300 * time.Millisecond)
	if got := atomic.LoadInt32(&viewCount); got != settled {
		t.Errorf("viewCount grew from %d to %d while idle", settled, got)
	}

	if _, err := pw.Write([]byte(strings.Repeat("x", 95))); err != nil {
		t.Fatalf("write to pipe: %v", err)
	}
	select {
	case res := <-resultCh:
		if res.err != nil {
			t.Fatalf("runLoop returned error: %v", res.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return in time")
	}
	if p.flushTimer != nil || p.renderPending {
		t.Error("quit left a throttled render pending")
	}
}

// TestWithMaxFPSArmsNoTimerWhenNothingIsThrottled shows the flush machinery
// stays idle when no render is denied.
func TestWithMaxFPSArmsNoTimerWhenNothingIsThrottled(t *testing.T) {
	p := NewProgram(fpsCounterModel{viewCount: new(int32), lastViewValue: new(int32)}, WithMaxFPS(1000))
	done := make(chan struct{})
	defer close(done)
	if !p.allowRender(done) {
		t.Fatal("first render was denied")
	}
	if p.flushTimer != nil || p.renderPending {
		t.Error("a timer was armed although nothing was throttled")
	}
	unthrottled := NewProgram(fpsCounterModel{viewCount: new(int32), lastViewValue: new(int32)}, WithMaxFPS(0))
	for i := 0; i < 3; i++ {
		if !unthrottled.allowRender(done) || unthrottled.flushTimer != nil {
			t.Fatal("unthrottled program must always render and never arm a timer")
		}
	}
}

func TestRunLoopDropsRepeatedFocusState(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	devnull := mustDevNull(t)
	defer devnull.Close()

	// Resize, then in, out, in: the repeats (in, in, out, out) are dropped.
	p := NewProgram(recorderModel{quitAfter: 4}, WithInput(pr), WithOutput(devnull))
	resultCh := make(chan runResult, 1)
	go func() {
		m, err := p.runLoop()
		resultCh <- runResult{m, err}
	}()
	if _, err := pw.Write([]byte("\x1b[I\x1b[I\x1b[O\x1b[O\x1b[I")); err != nil {
		t.Fatalf("write to pipe: %v", err)
	}
	var res runResult
	select {
	case res = <-resultCh:
	case <-time.After(2 * time.Second):
		t.Fatal("runLoop did not return in time")
	}
	rec := res.model.(recorderModel)
	var got []bool
	for _, m := range rec.received {
		if f, ok := m.(FocusEvent); ok {
			got = append(got, f.Focused)
		}
	}
	if len(got) != 3 || !got[0] || got[1] || !got[2] {
		t.Errorf("focus states = %v, want [true false true]", got)
	}
}

func TestFocusDedupeFirstAlwaysDelivered(t *testing.T) {
	for _, first := range []bool{true, false} {
		var d focusDedupe
		if !d.deliver(FocusEvent{Focused: first}) {
			t.Errorf("first event Focused=%v dropped", first)
		}
		if d.deliver(FocusEvent{Focused: first}) {
			t.Errorf("repeat Focused=%v delivered", first)
		}
		if !d.deliver(FocusEvent{Focused: !first}) {
			t.Errorf("change from %v dropped", first)
		}
	}
}
