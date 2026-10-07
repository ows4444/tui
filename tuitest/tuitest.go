// Package tuitest is a headless harness for testing tui models. A Session
// runs a model in a real tui.Program against an in-memory terminal: keys,
// pastes, resizes and arbitrary Msgs go in, and the VT-emulated screen (what
// a user would see after the renderer's escape sequences are applied) comes
// out as lines, optionally compared against golden files.
package tuitest

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/termio"
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
	term    *termio.Fake // the size the Program asks for; see New
	inW     *io.PipeWriter
	inR     *io.PipeReader
	inbox   chan tui.Msg
	quit    chan struct{}
	done    chan struct{}
	updates atomic.Int64 // every Update, whatever caused it: settle's quiet check

	// What the harness sent and the model has since received, counted by
	// kind. A method waits on the counter of what it sent, so an Update
	// caused by something else (a tick, a command's result) is not taken
	// for it.
	keys, pastes, resizes, injects atomic.Int64

	mu     sync.Mutex
	screen *vtscreen.Screen
	writes int

	closeOnce sync.Once
}

// New starts model in a w x h terminal and returns once its first frame is
// drawn. opts are extra tui.ProgramOptions, applied after the harness's own
// (which set the alternate screen, an in-memory input and output, and a
// terminal that reports the size w x h, so the Program draws frames of that
// size and not of the 80x24 a plain writer gets). A test that passes its own
// tui.WithTerminal replaces that terminal, and the Program then follows the
// size that one reports. Call Close when finished.
func New(model tui.Model, w, h int, opts ...tui.ProgramOption) *Session {
	pr, pw := io.Pipe()
	s := &Session{
		w: w, h: h, inW: pw, inR: pr,
		inbox:  make(chan tui.Msg, 64),
		quit:   make(chan struct{}),
		done:   make(chan struct{}),
		screen: vtscreen.NewScreen(w, h),
		term:   &termio.Fake{W: w, H: h, SizeKnown: true},
	}
	all := append([]tui.ProgramOption{
		tui.WithAltScreen(true),
		tui.WithInput(pr),
		tui.WithOutput(screenWriter{s}),
		tui.WithErrOutput(io.Discard),
		tui.WithTerminal(s.term),
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

// host wraps the user's model to count updates and what caused them, deliver
// injected Msgs and hide the pre-size frame.
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

// counter returns the counter of the kind of message msg is, or nil for a
// message the harness did not send: a tick, a command's result.
func (s *Session) counter(msg tui.Msg) *atomic.Int64 {
	switch msg.(type) {
	case injected:
		return &s.injects
	case tui.Key, tui.ChordMsg:
		return &s.keys
	case tui.PasteEvent:
		return &s.pastes
	case tui.ResizeMsg:
		return &s.resizes
	}
	return nil
}

func (h *host) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	// Counted once the model has handled it, so a waiter that sees the count
	// also sees the model's new state.
	defer func(c *atomic.Int64) {
		if c != nil {
			c.Add(1)
		}
		h.s.updates.Add(1)
	}(h.s.counter(msg))
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
	start := s.injects.Load()
	select {
	case s.inbox <- msg:
	case <-s.done:
		return
	}
	s.settle(&s.injects, start+1)
}

// settle waits until the model has received want messages of the kind n
// counts, then until output goes quiet.
func (s *Session) settle(n *atomic.Int64, want int64) {
	deadline := time.Now().Add(settleTimeout)
	for n.Load() < want && time.Now().Before(deadline) {
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

// csiLetterKeys and tildeKeys are the xterm sequences of the named keys that
// take a modifier parameter: ESC [ 1 ; mod LETTER and ESC [ code ; mod ~.
var (
	csiLetterKeys = map[string]byte{"up": 'A', "down": 'B', "right": 'C', "left": 'D', "home": 'H', "end": 'F'}
	tildeKeys     = map[string]int{
		"insert": 2, "delete": 3, "pgup": 5, "pgdown": 6,
		"f1": 11, "f2": 12, "f3": 13, "f4": 14, "f5": 15, "f6": 17, "f7": 18, "f8": 19, "f9": 20, "f10": 21,
		"f11": 23, "f12": 24, "f13": 25, "f14": 26, "f15": 28, "f16": 29, "f17": 31, "f18": 32, "f19": 33, "f20": 34,
	}
	// kittyKeys are the keys only the kitty protocol's ESC [ code ; mod u names.
	kittyKeys = map[string]int{
		"enter": 13, "tab": 9, "backspace": 127, "esc": 27, "space": 32,
		"f21": 57384, "f22": 57385, "f23": 57386, "f24": 57387,
		"media-play": 57428, "media-pause": 57429, "media-play-pause": 57430, "media-reverse": 57431,
		"media-stop": 57432, "media-fast-forward": 57433, "media-rewind": 57434, "media-next": 57435,
		"media-previous": 57436, "media-record": 57437, "volume-down": 57438, "volume-up": 57439, "mute": 57440,
	}
	modBits = map[string]int{"shift": 1, "alt": 2, "ctrl": 4, "super": 8}
)

// encodeKey returns the bytes a terminal sends for the key named k, in the
// spelling input.Key.String uses ("f5", "ctrl+left", "ctrl+shift+a"). A k
// that names no key is returned unchanged, as literal text.
func encodeKey(k string) string {
	if b, ok := keyBytes[k]; ok {
		return b
	}
	if rest, ok := strings.CutPrefix(k, "ctrl+"); ok && len(rest) == 1 && rest[0] >= 'a' && rest[0] <= 'z' {
		return string(rest[0] - 'a' + 1)
	}
	// Leading modifier names, then the key. "ctrl++" is ctrl and "+".
	mods, base := 0, k
	for {
		name, rest, ok := strings.Cut(base, "+")
		bit, isMod := modBits[name]
		if !ok || !isMod || rest == "" || mods&bit != 0 {
			break
		}
		mods, base = mods|bit, rest
	}
	if mods == 2 && base != "space" { // alt alone: the legacy ESC prefix (ESC SP decodes as "alt+ ")
		if enc := encodeKey(base); enc != base || len([]rune(base)) == 1 {
			return "\x1b" + enc
		}
		return k
	}
	// The xterm sequences have no super bit; only the kitty form carries it.
	if c, ok := csiLetterKeys[base]; ok && mods&8 == 0 {
		return fmt.Sprintf("\x1b[1;%d%c", mods+1, c)
	}
	if n, ok := tildeKeys[base]; ok && mods&8 == 0 {
		if mods == 0 {
			return fmt.Sprintf("\x1b[%d~", n)
		}
		return fmt.Sprintf("\x1b[%d;%d~", n, mods+1)
	}
	if n, ok := kittyKeys[base]; ok {
		return fmt.Sprintf("\x1b[%d;%du", n, mods+1)
	}
	if r := []rune(base); len(r) == 1 && mods != 0 {
		return fmt.Sprintf("\x1b[%d;%du", r[0], mods+1)
	}
	return k // literal text
}

// Keys sends key presses, one argument each. An argument is a key named the
// way tui.Key.String names it ("enter", "tab", "esc", "up", "pgdown", "f5",
// "insert", "ctrl+c", "alt+x", "shift+tab", "ctrl+left", "ctrl+shift+a",
// "volume-up") or literal text, which is sent as one key press per character.
// The one group of names it cannot send is super+ with a navigation or
// function key (up to f20), which the xterm sequences have no bit for; such an
// argument is typed as text.
//
// Keys returns once the model has received every key and the frame has
// settled. Only key messages count toward that: an Update caused by a tick or
// a command's result while the keys are in flight does not. A key the Program
// keeps for itself (the inspector's, or all but the last of a chord set with
// tui.WithChords) never reaches the model, so Keys then waits out its
// five-second limit before returning.
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
	start := s.keys.Load()
	for _, q := range seqs {
		if _, err := s.inW.Write([]byte(q)); err != nil {
			return
		}
	}
	s.settle(&s.keys, start+int64(len(seqs)))
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
	start := s.pastes.Load()
	if _, err := s.inW.Write([]byte("\x1b[200~" + text + "\x1b[201~")); err != nil {
		return
	}
	s.settle(&s.pastes, start+1)
}

// Resize changes the terminal to w x h: the emulated screen, the size the
// Program draws its frames at, and the tui.ResizeMsg the model receives.
func (s *Session) Resize(w, h int) {
	s.mu.Lock()
	s.screen.Resize(w, h)
	s.w, s.h = w, h
	s.mu.Unlock()
	// The Program reads its size from the terminal when a ResizeMsg passes
	// through its loop, so the message goes through the Program, not through
	// the inbox that reaches only the model.
	s.term.SetSize(w, h)
	start := s.resizes.Load()
	s.prog.Send(tui.ResizeMsg{Width: w, Height: h})
	s.settle(&s.resizes, start+1)
}

// Send delivers an arbitrary Msg to the model's Update.
func (s *Session) Send(msg tui.Msg) { s.inject(msg) }

// Screen returns the VT-emulated screen, one right-trimmed string per row.
func (s *Session) Screen() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.screen.Lines()
}

// WaitForText waits until some row of the screen contains text, and reports
// whether it did within timeout. Use it for output that arrives on its own
// time, after a command or a tick, which no call to Keys or Send waits for. A
// row is matched as Screen returns it, so text that wraps across two rows is
// not found. If the program exits first, the screen it left is checked once.
func (s *Session) WaitForText(text string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if s.hasText(text) {
			return true
		}
		if !time.Now().Before(deadline) {
			return false
		}
		select {
		case <-s.done:
			return s.hasText(text)
		case <-time.After(2 * time.Millisecond):
		}
	}
}

func (s *Session) hasText(text string) bool {
	for _, row := range s.Screen() {
		if strings.Contains(row, text) {
			return true
		}
	}
	return false
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
