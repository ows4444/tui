// Package tuitest is a headless harness for testing tui models. A Session
// runs a model in a real tui.Program against an in-memory terminal: keys,
// pastes, resizes and arbitrary Msgs go in, and the VT-emulated screen (what
// a user would see after the renderer's escape sequences are applied) comes
// out as lines, optionally compared against golden files.
package tuitest

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/vtscreen"
)

const (
	settleQuiet   = 25 * time.Millisecond
	settleTimeout = 5 * time.Second
)

// updating reports whether Golden should rewrite files: TUITEST_UPDATE=1 is
// set, or the test binary has a boolean "update" flag that is true. tuitest
// registers no flag itself (importing it must not touch flag.CommandLine); a
// test package that wants -update declares it, as any golden test does.
func updating() bool {
	if os.Getenv("TUITEST_UPDATE") == "1" {
		return true
	}
	f := flag.Lookup("update")
	return f != nil && f.Value.String() == "true"
}

// TB is the subset of testing.TB that Golden needs.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// Session is a model running headless. Its methods are not safe for
// concurrent use with each other.
type Session struct {
	w, h    int
	prog    *tui.Program
	inW     *io.PipeWriter
	inR     *io.PipeReader
	inbox   chan tui.Msg
	quit    chan struct{}
	done    chan struct{}
	updates atomic.Int64

	mu     sync.Mutex
	screen *vtscreen.Screen
	writes int

	closeOnce sync.Once
}

// New starts model in a w x h terminal and returns once its first frame is
// drawn. opts are extra tui.ProgramOptions, applied after the harness's own
// (which set the alternate screen, an in-memory input and output). Call Close
// when finished.
func New(model tui.Model, w, h int, opts ...tui.ProgramOption) *Session {
	pr, pw := io.Pipe()
	s := &Session{
		w: w, h: h, inW: pw, inR: pr,
		inbox:  make(chan tui.Msg, 64),
		quit:   make(chan struct{}),
		done:   make(chan struct{}),
		screen: vtscreen.NewScreen(w, h),
	}
	all := append([]tui.ProgramOption{
		tui.WithAltScreen(true),
		tui.WithInput(pr),
		tui.WithOutput(screenWriter{s}),
		tui.WithErrOutput(io.Discard),
		// The emulator reads SGR, so keep colours whatever the environment
		// (NO_COLOR, a dumb TERM) says; a test can override with its own
		// tui.WithColorProfile.
		tui.WithColorProfile(ansi.TrueColor),
	}, opts...)
	s.prog = tui.NewProgram(&host{s: s, inner: model}, all...)
	go func() {
		defer close(s.done)
		_, _ = s.prog.Run()
		_ = pr.Close()
	}()
	s.inject(tui.ResizeMsg{Width: w, Height: h})
	return s
}

type screenWriter struct{ s *Session }

func (sw screenWriter) Write(p []byte) (int, error) {
	sw.s.mu.Lock()
	sw.s.screen.Write(p)
	sw.s.writes++
	sw.s.mu.Unlock()
	return len(p), nil
}

// host wraps the user's model to count updates, deliver injected Msgs and
// hide the pre-size frame.
type host struct {
	s     *Session
	inner tui.Model
	sized bool
}

type injected struct{ msg tui.Msg }

func (h *host) listen() tui.Cmd {
	return func() tui.Msg {
		select {
		case m := <-h.s.inbox:
			return injected{m}
		case <-h.s.quit:
			return nil
		}
	}
}

func (h *host) Init() tui.Cmd { return tui.Batch(h.inner.Init(), h.listen()) }

func (h *host) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	defer h.s.updates.Add(1)
	next := *h
	if in, ok := msg.(injected); ok {
		msg = in.msg
		if _, ok := msg.(tui.ResizeMsg); ok {
			next.sized = true
		}
		var c tui.Cmd
		next.inner, c = h.inner.Update(msg)
		return &next, tui.Batch(h.s.watchQuit(c), h.listen())
	}
	if rm, ok := msg.(tui.ResizeMsg); ok && !h.sized {
		msg = tui.ResizeMsg{Width: h.s.w, Height: h.s.h} // never expose the 80x24 default
		_ = rm
	}
	var c tui.Cmd
	next.inner, c = h.inner.Update(msg)
	return &next, h.s.watchQuit(c)
}

func (h *host) View() string {
	if !h.sized {
		return ""
	}
	return h.inner.View()
}

// Unwrap and Rewrap let the Program find the optional interfaces (CursorPlacer,
// Linearizer, Themeable, BindingsProvider) of the model under test through the
// host, so the host offers exactly the ones the inner model does without a
// wrapper type per combination.
func (h *host) Unwrap() tui.Model { return h.inner }

func (h *host) Rewrap(inner tui.Model) tui.Model {
	next := *h
	next.inner = inner
	return &next
}

// watchQuit releases the parked input reader as soon as a Cmd returns
// tui.Quit, so Run can return promptly instead of waiting on it.
func (s *Session) watchQuit(c tui.Cmd) tui.Cmd {
	if c == nil {
		return nil
	}
	return func() tui.Msg {
		m := c()
		if _, ok := m.(tui.QuitMsg); ok {
			_ = s.inR.Close()
		}
		return m
	}
}

func (s *Session) inject(msg tui.Msg) {
	start := s.updates.Load()
	select {
	case s.inbox <- msg:
	case <-s.done:
		return
	}
	s.settle(start + 1)
}

// settle waits until updates reaches want, then until output goes quiet.
func (s *Session) settle(want int64) {
	deadline := time.Now().Add(settleTimeout)
	for s.updates.Load() < want && time.Now().Before(deadline) {
		select {
		case <-s.done:
			return
		case <-time.After(time.Millisecond):
		}
	}
	lastU, lastW := int64(-1), -1
	quietSince := time.Now()
	for time.Now().Before(deadline) {
		s.mu.Lock()
		w := s.writes
		s.mu.Unlock()
		u := s.updates.Load()
		if u != lastU || w != lastW {
			lastU, lastW, quietSince = u, w, time.Now()
		} else if time.Since(quietSince) >= settleQuiet {
			return
		}
		select {
		case <-s.done:
			return
		case <-time.After(2 * time.Millisecond):
		}
	}
}

var keyBytes = map[string]string{
	"enter": "\r", "tab": "\t", "esc": "\x1b", "escape": "\x1b",
	"backspace": "\x7f", "space": " ",
	"up": "\x1b[A", "down": "\x1b[B", "right": "\x1b[C", "left": "\x1b[D",
	"home": "\x1b[H", "end": "\x1b[F", "pgup": "\x1b[5~", "pgdown": "\x1b[6~",
	"delete": "\x1b[3~", "shift+tab": "\x1b[Z",
}

func encodeKey(k string) string {
	if b, ok := keyBytes[k]; ok {
		return b
	}
	if rest, ok := strings.CutPrefix(k, "ctrl+"); ok && len(rest) == 1 && rest[0] >= 'a' && rest[0] <= 'z' {
		return string(rest[0] - 'a' + 1)
	}
	if rest, ok := strings.CutPrefix(k, "alt+"); ok {
		return "\x1b" + encodeKey(rest)
	}
	return k // literal text
}

// Keys sends key presses, one argument each. An argument is a named key
// ("enter", "tab", "esc", "backspace", "space", "up", "down", "left",
// "right", "home", "end", "pgup", "pgdown", "delete", "shift+tab",
// "ctrl+<letter>", "alt+<key>") or literal text, which is sent as one key
// press per character. Keys returns once the frame has settled.
func (s *Session) Keys(keys ...string) {
	var seqs []string
	for _, k := range keys {
		if enc := encodeKey(k); enc != k || len([]rune(k)) <= 1 {
			seqs = append(seqs, enc)
			continue
		}
		for _, r := range k {
			seqs = append(seqs, string(r))
		}
	}
	start := s.updates.Load()
	for _, q := range seqs {
		if _, err := s.inW.Write([]byte(q)); err != nil {
			return
		}
	}
	s.settle(start + int64(len(seqs)))
}

// Cell is one screen cell as the VT-emulated terminal shows it: the character
// and its styling. FG and BG are nil for the terminal's default colour;
// otherwise they are an ansi.BasicColor, ansi.Color256 or ansi.RGB, as the
// program's SGR sequence set them.
type Cell struct {
	// Grapheme is the character in the cell, " " when it is blank. The
	// emulator stores one rune per cell, so a wide or combining cluster is
	// not reassembled.
	Grapheme                                      string
	Bold, Dim, Italic, Underline, Reverse, Strike bool
	FG, BG                                        ansi.Color
}

// Cell returns the cell at column x, row y (0-based, origin top-left). A
// position off the screen is a blank default cell.
func (s *Session) Cell(x, y int) Cell {
	s.mu.Lock()
	c := s.screen.Cell(x, y)
	s.mu.Unlock()
	return Cell{
		Grapheme: string(c.Rune), Bold: c.Bold, Dim: c.Dim, Italic: c.Italic,
		Underline: c.Underline, Reverse: c.Reverse, Strike: c.Strike, FG: c.FG, BG: c.BG,
	}
}

// Cursor returns where the terminal cursor is (0-based column and row) and
// whether it is shown. A model implementing tui.CursorPlacer has it placed
// where CursorPos says; otherwise the Program keeps it hidden.
func (s *Session) Cursor() (x, y int, visible bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.screen.Cursor()
}

// Click delivers a left-button press and release at column x, row y (0-based)
// as tui.MouseEvents, straight to the model like Send, so it works whether or
// not the program enabled mouse reporting.
func (s *Session) Click(x, y int) {
	s.inject(tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	s.inject(tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease})
}

// Wheel delivers |dy| wheel notches at column x, row y: up for a negative dy,
// down for a positive one. Like Click it goes straight to the model.
func (s *Session) Wheel(x, y, dy int) {
	b := tui.MouseButtonWheelDown
	if dy < 0 {
		b, dy = tui.MouseButtonWheelUp, -dy
	}
	for ; dy > 0; dy-- {
		s.inject(tui.MouseEvent{X: x, Y: y, Button: b, Action: tui.MouseActionPress})
	}
}

// Paste delivers text as one bracketed paste.
func (s *Session) Paste(text string) {
	start := s.updates.Load()
	if _, err := s.inW.Write([]byte("\x1b[200~" + text + "\x1b[201~")); err != nil {
		return
	}
	s.settle(start + 1)
}

// Resize changes the terminal to w x h and delivers a tui.ResizeMsg.
func (s *Session) Resize(w, h int) {
	s.mu.Lock()
	s.screen.Resize(w, h)
	s.w, s.h = w, h
	s.mu.Unlock()
	s.inject(tui.ResizeMsg{Width: w, Height: h})
}

// Send delivers an arbitrary Msg to the model's Update.
func (s *Session) Send(msg tui.Msg) { s.inject(msg) }

// Screen returns the VT-emulated screen, one right-trimmed string per row.
func (s *Session) Screen() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.screen.Lines()
}

// Done reports whether the program has exited (for example after the model
// returned tui.Quit()).
func (s *Session) Done() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// Close stops the program and waits for it to exit. It is safe to call more
// than once.
func (s *Session) Close() {
	s.closeOnce.Do(func() {
		close(s.quit)
		_ = s.inW.Close() // EOF on input quits the program
		select {
		case <-s.done:
		case <-time.After(settleTimeout):
		}
	})
}

// Golden compares Screen against testdata/<name>.golden. Run the tests with
// -update (a bool flag the test package declares) or TUITEST_UPDATE=1 to
// write the file instead.
func (s *Session) Golden(t TB, name string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	got := strings.Join(s.Screen(), "\n") + "\n"
	if updating() {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { // #nosec G301 -- a golden directory committed to the repository, world-readable like any source file
			t.Fatalf("tuitest: %v", err)
			return
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil { // #nosec G306 -- a golden file committed to the repository, world-readable like any source file
			t.Fatalf("tuitest: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path) // #nosec G304 -- path is testdata/<name>.golden built from the test author's own golden name
	if err != nil {
		t.Fatalf("tuitest: reading golden (run with -update to create): %v", err)
		return
	}
	if string(want) != got {
		t.Errorf("tuitest: screen differs from %s (run with -update to accept)\n--- want\n%s--- got\n%s", path, want, got)
	}
}
