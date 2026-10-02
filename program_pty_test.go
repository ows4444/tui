//go:build darwin || linux

package tui

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/ptytest"
	"github.com/ows4444/tui/internal/vtscreen"
)

// These tests run Program.Run against a real pseudo-terminal, so raw mode,
// the escape codes written around the run, terminal size and SIGWINCH
// handling are exercised for real. They skip where a pty can't be opened.

// ptyModel records every Msg and draws "frame:<count>", so each message
// repaints the screen; typing q quits.
type ptyModel struct{ msgs []Msg }

func (m ptyModel) Init() Cmd { return nil }
func (m ptyModel) Update(msg Msg) (Model, Cmd) {
	m.msgs = append(append([]Msg(nil), m.msgs...), msg)
	if k, ok := msg.(Key); ok && k.Type == KeyRunes && k.Text == "q" {
		return m, Quit()
	}
	return m, nil
}
func (m ptyModel) View() string { return fmt.Sprintf("frame:%d", len(m.msgs)) }

// ptySession is a Program running on one end of a pty with the test on the
// other, reading the program's output without ever blocking.
type ptySession struct {
	t       *testing.T
	master  *os.File
	slave   *os.File
	mfd     int
	out     bytes.Buffer
	done    chan runResult
	before  syscall.Termios // the terminal's settings before Run started
	program *Program
}

// openPTY returns a fresh pty of the given size, skipping the test where
// none can be opened, and closes it when the test ends.
func openPTY(t *testing.T, cols, rows uint16) (master, slave *os.File) {
	t.Helper()
	master, slave, err := ptytest.Open()
	if err != nil {
		t.Skipf("cannot open a pty here: %v", err)
	}
	t.Cleanup(func() {
		slave.Close()
		master.Close()
	})
	if err := ptytest.SetSize(master, cols, rows); err != nil {
		t.Fatalf("set size: %v", err)
	}
	return master, slave
}

func startPTY(t *testing.T, m Model, cols, rows uint16, opts ...ProgramOption) *ptySession {
	t.Helper()
	master, slave := openPTY(t, cols, rows)
	return attachPTY(t, master, slave, m, opts...)
}

// attachPTY runs a Program on an already-open pty (so a test can run
// several programs on the same terminal in turn).
func attachPTY(t *testing.T, master, slave *os.File, m Model, opts ...ProgramOption) *ptySession {
	t.Helper()
	s := &ptySession{t: t, master: master, slave: slave, done: make(chan runResult, 1)}
	var err error
	if s.before, err = ptytest.Termios(slave); err != nil {
		t.Fatalf("termios: %v", err)
	}
	s.mfd = int(master.Fd())
	if err := syscall.SetNonblock(s.mfd, true); err != nil {
		t.Fatalf("SetNonblock: %v", err)
	}
	p := NewProgram(m, append([]ProgramOption{WithInput(slave), WithOutput(slave), WithCellRenderer(false)}, opts...)...)
	s.program = p
	go func() {
		model, err := p.Run()
		s.done <- runResult{model, err}
	}()
	return s
}

// pump reads program output for up to d, or until stop reports true.
func (s *ptySession) pump(d time.Duration, stop func() bool) {
	buf := make([]byte, 4096)
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		n, err := syscall.Read(s.mfd, buf)
		if n > 0 {
			s.out.Write(buf[:n])
		}
		if stop != nil && stop() {
			return
		}
		if n > 0 {
			continue
		}
		if err != nil && err != syscall.EAGAIN {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func (s *ptySession) waitFor(sub string) {
	s.t.Helper()
	s.pump(3*time.Second, func() bool { return strings.Contains(s.out.String(), sub) })
	if !strings.Contains(s.out.String(), sub) {
		s.t.Fatalf("timed out waiting for %q in output %q", sub, s.out.String())
	}
}

func (s *ptySession) send(str string) {
	s.t.Helper()
	if n, err := syscall.Write(s.mfd, []byte(str)); err != nil || n != len(str) {
		s.t.Fatalf("write %q: n=%d err=%v", str, n, err)
	}
}

// finish waits for Run to return, then collects any trailing output.
func (s *ptySession) finish() runResult {
	s.t.Helper()
	start := time.Now()
	for {
		s.pump(50*time.Millisecond, nil)
		select {
		case r := <-s.done:
			s.pump(200*time.Millisecond, nil)
			return r
		default:
		}
		if time.Since(start) > 5*time.Second {
			s.t.Fatalf("Run did not return; output so far %q", s.out.String())
		}
	}
}

func TestRunEntersRawModeAndRestoresTheTerminal(t *testing.T) {
	s := startPTY(t, ptyModel{}, 80, 24)
	if uint64(s.before.Lflag)&uint64(syscall.ICANON) == 0 || uint64(s.before.Lflag)&uint64(syscall.ECHO) == 0 {
		t.Fatalf("expected a cooked starting state, got Lflag %#x", s.before.Lflag)
	}

	s.waitFor("frame:1")
	during, err := ptytest.Termios(s.slave)
	if err != nil {
		t.Fatal(err)
	}
	for name, flag := range map[string]uint64{"ICANON": uint64(syscall.ICANON), "ECHO": uint64(syscall.ECHO), "ISIG": uint64(syscall.ISIG)} {
		if uint64(during.Lflag)&flag != 0 {
			t.Errorf("%s still set while Run is running: not in raw mode", name)
		}
	}

	// A key reaches the model with no newline, and is not echoed.
	s.send("a")
	s.waitFor("frame:2")
	s.send("q")
	if res := s.finish(); res.err != nil {
		t.Fatalf("Run error: %v", res.err)
	}

	after, err := ptytest.Termios(s.slave)
	if err != nil {
		t.Fatal(err)
	}
	if after != s.before {
		t.Errorf("terminal settings after Run differ from before:\n got  %+v\n want %+v", after, s.before)
	}
}

// TestContextCancelledOnQuit proves first acceptance criterion:
// the context.Context returned by Program.Context is cancelled once Run
// returns via the ordinary Quit path, so a Cmd holding that context (e.g.
// one wrapping an HTTP request via http.NewRequestWithContext) observes
// shutdown instead of leaking past it.
func TestContextCancelledOnQuit(t *testing.T) {
	s := startPTY(t, ptyModel{}, 80, 24)
	ctx := s.program.Context()

	select {
	case <-ctx.Done():
		t.Fatal("Context() already cancelled before Run finished")
	default:
	}

	s.waitFor("frame:1")
	s.send("q")
	if res := s.finish(); res.err != nil {
		t.Fatalf("Run error: %v", res.err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Context() was not cancelled after Run returned")
	}
	if err := ctx.Err(); err != context.Canceled {
		t.Errorf("ctx.Err() = %v, want context.Canceled", err)
	}
}

// TestContextDerivesFromWithContext proves the second acceptance criterion:
// isn't just "a context exists" but that it's genuinely derived from
// WithContext's parent: a value set on the parent is visible through
// Program.Context(), and cancelling the parent also cancels Program's
// derived context (the standard context.WithCancel propagation contract) —
// without requiring Run to have started, since NewProgram derives ctx
// immediately.
func TestContextDerivesFromWithContext(t *testing.T) {
	type ctxKey struct{}
	parent, cancelParent := context.WithCancel(context.WithValue(context.Background(), ctxKey{}, "hello"))
	defer cancelParent()

	p := NewProgram(ptyModel{}, WithContext(parent))
	ctx := p.Context()

	if v := ctx.Value(ctxKey{}); v != "hello" {
		t.Errorf("Context().Value() = %v, want %q (not derived from WithContext's parent)", v, "hello")
	}

	select {
	case <-ctx.Done():
		t.Fatal("Context() cancelled before parent was cancelled or Run started")
	default:
	}

	cancelParent()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("Context() was not cancelled when its WithContext parent was cancelled")
	}
}

// sigtermHelperEnv, when set to "1", tells
// TestSigtermRestoresTerminal_HelperProcess to actually run instead of
// skipping — it's re-executed as a subprocess of TestSigtermRestoresTerminal
// via os.Args[0], since sending SIGTERM to the test binary's own process
// would kill the whole `go test` run (the fix under test calls os.Exit).
const sigtermHelperEnv = "TUI_SIGTERM_HELPER"

// TestSigtermRestoresTerminal_HelperProcess is not a real test: it's the
// child-process body for TestSigtermRestoresTerminal, run only when
// sigtermHelperEnv is set. It runs a Program on the pty passed as fd 3 and
// blocks until killed.
func TestSigtermRestoresTerminal_HelperProcess(t *testing.T) {
	if os.Getenv(sigtermHelperEnv) != "1" {
		t.Skip("helper process for TestSigtermRestoresTerminal")
	}
	tty := os.NewFile(3, "tty")
	NewProgram(ptyModel{}, WithInput(tty), WithOutput(tty)).Run()
}

// TestSigtermRestoresTerminal proves acceptance criterion: a
// SIGTERM delivered while Run has the terminal in raw mode must restore it
// before the process exits, since Go's default SIGTERM disposition skips
// deferred functions entirely.
func TestSigtermRestoresTerminal(t *testing.T) {
	master, slave := openPTY(t, 80, 24)
	_ = master // kept open (via openPTY's t.Cleanup) so the pty stays valid; unused directly here
	before, err := ptytest.Termios(slave)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestSigtermRestoresTerminal_HelperProcess$")
	cmd.Env = append(os.Environ(), sigtermHelperEnv+"=1")
	cmd.ExtraFiles = []*os.File{slave}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })

	// Wait for the helper to actually enter raw mode before signaling it,
	// so the test doesn't race a SIGTERM against Run's own setup.
	deadline := time.Now().Add(3 * time.Second)
	for {
		during, terr := ptytest.Termios(slave)
		if terr == nil && during != before {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("helper process never entered raw mode")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal: %v", err)
	}

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	select {
	case <-waitErr:
		// The fix calls os.Exit(1) after restoring, so a non-zero exit is
		// expected — only the terminal state below is being verified.
	case <-time.After(3 * time.Second):
		t.Fatal("helper process did not exit after SIGTERM")
	}

	after, err := ptytest.Termios(slave)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Errorf("terminal settings after SIGTERM differ from before:\n got  %+v\n want %+v", after, before)
	}
}

// Run's comment promises that a second sequential Run on the same input
// file works (it clears the read deadline the first one left behind); on a
// real terminal that also means the first Run's input reader must not be
// left behind to steal the second Run's keys.
func TestRunTwiceOnTheSameTerminal(t *testing.T) {
	master, slave := openPTY(t, 80, 24)

	first := attachPTY(t, master, slave, ptyModel{})
	first.waitFor("frame:1")
	first.send("q")
	if res := first.finish(); res.err != nil {
		t.Fatalf("first Run error: %v", res.err)
	}

	second := attachPTY(t, master, slave, ptyModel{})
	second.waitFor("frame:1")
	second.send("a")
	second.waitFor("frame:2") // fails if a leftover reader from the first Run took the key
	second.send("q")
	res := second.finish()
	if res.err != nil {
		t.Fatalf("second Run error: %v", res.err)
	}
	if n := len(res.model.(ptyModel).msgs); n != 3 {
		t.Errorf("second Run's model saw %d msgs, want 3 (resize, a, q)", n)
	}
	if after, _ := ptytest.Termios(slave); after != first.before {
		t.Error("terminal not restored after the second Run")
	}
}

func TestRunWritesTerminalModeCodesInOrder(t *testing.T) {
	tests := []struct {
		name string
		opts []ProgramOption
		// startCodes is exactly what Run writes before the first frame;
		// endCodes exactly what it writes after the last.
		startCodes, endCodes string
		absent               []string
	}{
		{
			name:       "defaults: alt screen, hidden cursor and bracketed paste",
			startCodes: ansi.AltScreenEnable + ansi.CursorHide + ansi.BracketedPasteEnable,
			endCodes:   ansi.BracketedPasteDisable + ansi.CursorShow + ansi.AltScreenDisable,
			absent:     []string{ansi.MouseSGREnable},
		},
		{
			name:       "alt screen off",
			opts:       []ProgramOption{WithAltScreen(false)},
			startCodes: ansi.CursorHide,
			endCodes:   ansi.CursorShow + "\r\n", // inline exit ends on a fresh row
			absent:     []string{ansi.AltScreenEnable, ansi.AltScreenDisable},
		},
		{
			name:       "mouse click",
			opts:       []ProgramOption{WithMouse(MouseClick)},
			startCodes: ansi.AltScreenEnable + ansi.CursorHide + ansi.MouseClickEnable + ansi.MouseSGREnable,
			endCodes:   ansi.MouseSGRDisable + ansi.MouseClickDisable + ansi.CursorShow + ansi.AltScreenDisable,
		},
		{
			name:       "mouse cell motion",
			opts:       []ProgramOption{WithMouse(MouseCellMotion)},
			startCodes: ansi.AltScreenEnable + ansi.CursorHide + ansi.MouseCellMotionEnable + ansi.MouseSGREnable,
			endCodes:   ansi.MouseSGRDisable + ansi.MouseCellMotionDisable + ansi.CursorShow + ansi.AltScreenDisable,
		},
		{
			name:       "mouse all motion",
			opts:       []ProgramOption{WithMouse(MouseAllMotion)},
			startCodes: ansi.AltScreenEnable + ansi.CursorHide + ansi.MouseAllMotionEnable + ansi.MouseSGREnable,
			endCodes:   ansi.MouseSGRDisable + ansi.MouseAllMotionDisable + ansi.CursorShow + ansi.AltScreenDisable,
		},
		{
			name:       "bracketed paste",
			opts:       []ProgramOption{WithBracketedPaste(true)},
			startCodes: ansi.AltScreenEnable + ansi.CursorHide + ansi.BracketedPasteEnable,
			endCodes:   ansi.BracketedPasteDisable + ansi.CursorShow + ansi.AltScreenDisable,
			absent:     []string{ansi.MouseSGREnable},
		},
		{
			name:       "kitty keyboard",
			opts:       []ProgramOption{WithKittyKeyboard(true)},
			startCodes: ansi.AltScreenEnable + ansi.CursorHide + ansi.BracketedPasteEnable + ansi.KittyKeyboardEnable,
			endCodes:   ansi.KittyKeyboardDisable + ansi.BracketedPasteDisable + ansi.CursorShow + ansi.AltScreenDisable,
			absent:     []string{ansi.MouseSGREnable},
		},
		{
			name:       "everything on",
			opts:       []ProgramOption{WithMouse(MouseAllMotion), WithBracketedPaste(true), WithKittyKeyboard(true)},
			startCodes: ansi.AltScreenEnable + ansi.CursorHide + ansi.MouseAllMotionEnable + ansi.MouseSGREnable + ansi.BracketedPasteEnable + ansi.KittyKeyboardEnable,
			endCodes:   ansi.KittyKeyboardDisable + ansi.BracketedPasteDisable + ansi.MouseSGRDisable + ansi.MouseAllMotionDisable + ansi.CursorShow + ansi.AltScreenDisable,
		},
		{
			name:       "mouse off and paste off explicit", // #17
			opts:       []ProgramOption{WithMouse(MouseOff), WithBracketedPaste(false)},
			startCodes: ansi.AltScreenEnable + ansi.CursorHide,
			endCodes:   ansi.CursorShow + ansi.AltScreenDisable,
			absent:     []string{ansi.MouseSGREnable, ansi.MouseSGRDisable, ansi.BracketedPasteEnable, ansi.BracketedPasteDisable},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := startPTY(t, ptyModel{}, 80, 24, tt.opts...)
			s.waitFor("frame:1")
			s.send("q")
			if res := s.finish(); res.err != nil {
				t.Fatalf("Run error: %v", res.err)
			}
			out := s.out.String()
			if !strings.HasPrefix(out, tt.startCodes) {
				t.Errorf("output does not start with the start-up codes\n got  %q\n want prefix %q", head(out, len(tt.startCodes)+20), tt.startCodes)
			}
			if !strings.HasSuffix(out, tt.endCodes) {
				t.Errorf("output does not end with the shutdown codes in reverse order\n got  ...%q\n want suffix %q", tail(out, len(tt.endCodes)+20), tt.endCodes)
			}
			if strings.Index(out, "frame:1") < len(tt.startCodes) {
				t.Error("first frame written before the start-up codes")
			}
			for _, a := range tt.absent {
				if strings.Contains(out, a) {
					t.Errorf("output contains %q, want it absent", a)
				}
			}
		})
	}
}

func head(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func tail(s string, n int) string {
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}

func TestRunSeedsSizeTracksResizeAndDeliversKeys(t *testing.T) {
	s := startPTY(t, ptyModel{}, 90, 25)
	s.waitFor("frame:1") // the seeded ResizeMsg has been handled and drawn

	// A resize arrives as SIGWINCH: change the pty's size, then signal.
	if err := ptytest.SetSize(s.master, 70, 20); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGWINCH); err != nil {
		t.Fatal(err)
	}
	s.waitFor("frame:2")

	s.send("hi")
	s.waitFor("frame:4")
	s.send("q")
	res := s.finish()
	if res.err != nil {
		t.Fatalf("Run error: %v", res.err)
	}

	got := res.model.(ptyModel).msgs
	if len(got) != 5 {
		t.Fatalf("model saw %d msgs (%v), want [Resize Resize h i q]", len(got), got)
	}
	if got[0] != (ResizeMsg{Width: 90, Height: 25}) {
		t.Errorf("first msg = %#v, want the seeded size 90x25", got[0])
	}
	if got[1] != (ResizeMsg{Width: 70, Height: 20}) {
		t.Errorf("after resize msg = %#v, want 70x20", got[1])
	}
	for i, want := range []rune{'h', 'i', 'q'} {
		k, ok := got[2+i].(Key)
		if !ok || k.Type != KeyRunes || k.Text != string(want) {
			t.Errorf("msg %d = %#v, want Key %q", 2+i, got[2+i], want)
		}
	}
	// Each message repainted the frame.
	for _, f := range []string{"frame:1", "frame:2", "frame:3", "frame:4"} {
		if !strings.Contains(s.out.String(), f) {
			t.Errorf("output missing %q", f)
		}
	}
}

// --- resize repaint ---

// tallModel draws a fixed grid of rows, each wider than any test terminal
// but the biggest, under a "WxH" header showing the size the model was
// last told. The view is deliberately taller/wider than a small terminal.
type tallModel struct {
	rows int
	w, h int
	fill int
}

func (m tallModel) Init() Cmd { return nil }
func (m tallModel) Update(msg Msg) (Model, Cmd) {
	if r, ok := msg.(ResizeMsg); ok {
		m.w, m.h = r.Width, r.Height
	}
	if k, ok := msg.(Key); ok && k.Type == KeyRunes && k.Text == "q" {
		return m, Quit()
	}
	return m, nil
}
func (m tallModel) View() string {
	lines := make([]string, m.rows)
	lines[0] = fmt.Sprintf("%dx%d", m.w, m.h)
	for i := 1; i < m.rows; i++ {
		lines[i] = fmt.Sprintf("row%03d", i) + strings.Repeat("x", m.fill)
	}
	return strings.Join(lines, "\n")
}

// screenSession pairs a ptySession with an emulated screen fed from the
// program's output.
type screenSession struct {
	*ptySession
	scr *vtscreen.Screen
	fed int
}

func (s *screenSession) sync() {
	b := s.out.Bytes()
	s.scr.Write(b[s.fed:])
	s.fed = len(b)
}

// resizeTo resizes the pty and the emulated screen together and signals the
// program, then waits until the first screen row reads want (the model's
// "WxH" header) or a timeout passes. It returns the screen rows either way,
// so the caller's comparison reports what was actually left on screen.
func (s *screenSession) resizeTo(cols, rows int, want string) []string {
	s.t.Helper()
	s.sync()
	s.scr.Resize(cols, rows)
	if err := ptytest.SetSize(s.master, uint16(cols), uint16(rows)); err != nil {
		s.t.Fatal(err)
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGWINCH); err != nil {
		s.t.Fatal(err)
	}
	// Generous: this only costs time when the repaint is late (-race on a
	// loaded runner), never when it arrives promptly.
	s.pump(10*time.Second, func() bool {
		s.sync()
		return strings.HasPrefix(s.scr.Lines()[0], want)
	})
	// Let any trailing repaint bytes land before judging the screen.
	s.pump(80*time.Millisecond, nil)
	s.sync()
	return s.scr.Lines()
}

func startScreen(t *testing.T, m Model, cols, rows int, opts ...ProgramOption) *screenSession {
	t.Helper()
	s := &screenSession{ptySession: startPTY(t, m, uint16(cols), uint16(rows), opts...), scr: vtscreen.NewScreen(cols, rows)}
	s.waitFor("row")
	s.sync()
	return s
}

// wantScreen is what a cols x rows terminal should show for m's view when
// nothing wraps or scrolls: the top `rows` lines, each cut to `cols`.
func wantScreen(m tallModel, cols, rows int) []string {
	out := make([]string, rows)
	for i, l := range strings.Split(m.View(), "\n") {
		if i >= rows {
			break
		}
		if len(l) > cols {
			l = l[:cols]
		}
		out[i] = l
	}
	return out
}

func diffScreen(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("screen has %d rows, want %d", len(got), len(want))
	}
	bad := 0
	for i := range want {
		if got[i] != want[i] {
			if bad++; bad <= 5 {
				t.Errorf("row %d = %q, want %q", i, got[i], want[i])
			}
		}
	}
	if bad > 5 {
		t.Errorf("... and %d more differing rows", bad-5)
	}
}

func TestResizeShrinkAndRegrowRepaints(t *testing.T) {
	m := tallModel{rows: 100, fill: 120}
	s := startScreen(t, m, 100, 100)
	defer s.finishQuit()

	m.w, m.h = 5, 5
	diffScreen(t, s.resizeTo(5, 5, "5x5"), wantScreen(m, 5, 5))

	m.w, m.h = 100, 100
	diffScreen(t, s.resizeTo(100, 100, "100x100"), wantScreen(m, 100, 100))
}

func TestResizeRepeatedlyEndsConsistent(t *testing.T) {
	m := tallModel{rows: 100, fill: 120}
	s := startScreen(t, m, 100, 100)
	defer s.finishQuit()

	for _, sz := range [][2]int{{5, 5}, {60, 30}, {3, 3}, {80, 100}, {1, 1}, {100, 100}} {
		m.w, m.h = sz[0], sz[1]
		got := s.resizeTo(sz[0], sz[1], fmt.Sprintf("%dx%d", sz[0], sz[1])[:min(sz[0], len(fmt.Sprintf("%dx%d", sz[0], sz[1])))])
		diffScreen(t, got, wantScreen(m, sz[0], sz[1]))
	}
}

func TestResizeIsNotThrottled(t *testing.T) {
	m := tallModel{rows: 10, fill: 120}
	s := startScreen(t, m, 40, 20, WithMaxFPS(1))
	defer s.finishQuit()

	m.w, m.h = 30, 12
	diffScreen(t, s.resizeTo(30, 12, "30x12"), wantScreen(m, 30, 12))
}

func TestResizeInlineLeavesOneCopyOfTheFrame(t *testing.T) {
	m := tallModel{rows: 8, fill: 120}
	s := startScreen(t, m, 40, 20, WithAltScreen(false))
	defer s.finishQuit()

	m.w, m.h = 30, 10
	diffScreen(t, s.resizeTo(30, 10, "30x10"), wantScreen(m, 30, 10))
	m.w, m.h = 40, 20
	diffScreen(t, s.resizeTo(40, 20, "40x20"), wantScreen(m, 40, 20))
}

// A terminal that reflows on resize shows a 100-wide row over two rows at
// width 50; the repaint must climb past all of them, not just liveLines.
func TestResizeInlineReflowLeavesNoGhostRows(t *testing.T) {
	m := tallModel{rows: 10, fill: 94}
	s := startScreen(t, m, 100, 30, WithAltScreen(false))
	defer s.finishQuit()

	m.w, m.h = 50, 30
	s.sync()
	s.scr.ResizeReflow(50, 30)
	if err := ptytest.SetSize(s.master, 50, 30); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGWINCH); err != nil {
		t.Fatal(err)
	}
	s.pump(10*time.Second, func() bool {
		s.sync()
		return strings.HasPrefix(s.scr.Lines()[0], "50x30")
	})
	s.pump(80*time.Millisecond, nil)
	s.sync()
	got := s.scr.Lines()
	want := make([]string, 30)
	for i, l := range strings.Split(m.View(), "\n") {
		want[i] = l[:min(len(l), 50)]
	}
	diffScreen(t, got, want)
}

// finishQuit ends the program; used with defer so a failed assertion still
// shuts Run down.
func (s *screenSession) finishQuit() {
	s.send("q")
	s.finish()
}
