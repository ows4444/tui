// Package termpane runs a program on a pseudo-terminal and draws its screen in
// a pane: a shell, an editor or any full-screen tool inside the application.
// The Model starts the command in Init, reads its output into a terminal
// emulator, draws the emulator's screen from its LayoutNode, and sends the keys,
// pastes and (when the program asks for them) mouse events it is given to the
// program. Shift+PgUp and Shift+PgDn, and the mouse wheel, scroll back through
// the lines that left the top of the screen.
//
// Route every Msg to Update, and a size to SetSize whenever the pane's size
// changes (usually from the ResizeMsg handler, once the layout is known): that
// resizes the program's terminal and sends it SIGWINCH.
//
// The output is untrusted: it is a second terminal. Nothing it writes reaches
// the host terminal. Its escape sequences only change the emulator's screen; its
// title is kept as plain text (see Title); clipboard, hyperlink and image
// sequences are dropped; and the only things sent back to it are the answers to
// cursor-position and device-attribute queries.
//
// It works on darwin and linux; elsewhere the pane reports an ExitedMsg with an
// error at Init.
//
// Stability: experimental. Its API may change in any minor release.
package termpane

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/internal/vt"
	"github.com/ows4444/tui/keymap"
)

// Default size of the terminal before SetSize.
const (
	defaultCols = 80
	defaultRows = 24
	// wheelStep is the lines one wheel notch scrolls back.
	wheelStep = 3
)

// Model is a pane running a program on a pseudo-terminal. Build one with New.
type Model struct {
	// ID is carried by OutputMsg and ExitedMsg, to tell one pane's from
	// another's.
	ID string
	// Command and Args are the program to run. Command is looked up on PATH.
	Command string
	Args    []string
	// Dir is the working directory; "" is the application's.
	Dir string
	// Env is the environment of the program; nil is the application's own.
	// TERM is set to xterm-256color and COLORTERM to truecolor unless Env
	// already names them.
	Env []string
	// Cols and Rows are the size the terminal starts with, 80x24 when 0. Change
	// it later with SetSize.
	Cols, Rows int
	// Scrollback is how many lines are kept above the screen: 0 means 1000 and a
	// negative value keeps none.
	Scrollback int

	// KeyMap holds the keys that scroll back. They are not sent to the program.
	// New fills it with DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: events in Bounds
	// are sent to the program when it asked for them, and otherwise the wheel
	// scrolls back. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the pane.
	Bounds hittest.Rect

	s       *session
	focused bool
	offset  int // lines scrolled back; 0 is the live screen
}

// KeyMap is the keys Update reacts to itself.
type KeyMap struct {
	// ScrollUp and ScrollDown move a page back and forward through the
	// scrollback; ScrollEnd returns to the live screen.
	ScrollUp, ScrollDown, ScrollEnd keymap.Binding
}

// DefaultKeyMap returns shift+pgup, shift+pgdown and shift+end.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		ScrollUp:   keymap.NewBinding("scroll back", "shift+pgup"),
		ScrollDown: keymap.NewBinding("scroll forward", "shift+pgdown"),
		ScrollEnd:  keymap.NewBinding("scroll to the end", "shift+end"),
	}
}

func (m Model) keys() KeyMap {
	if len(m.KeyMap.ScrollUp.Keys)+len(m.KeyMap.ScrollDown.Keys)+len(m.KeyMap.ScrollEnd.Keys) == 0 {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the active bindings, for help text.
func (m Model) Bindings() []keymap.Binding {
	k := m.keys()
	return []keymap.Binding{k.ScrollUp, k.ScrollDown, k.ScrollEnd}
}

// New returns a pane that will run command with args.
func New(command string, args ...string) Model {
	return Model{Command: command, Args: args, KeyMap: DefaultKeyMap(), s: newSession()}
}

// OutputMsg is delivered when the program wrote something, or the size changed:
// the pane has a new screen to draw. Pass it to Update, which asks for the next.
type OutputMsg struct{ ID string }

// ExitedMsg is delivered once when the program has exited and its last output
// has been read. Code is its exit status, -1 when it was killed by a signal, and
// Err is set when it could not be started.
type ExitedMsg struct {
	ID   string
	Code int
	Err  error
}

// session is the part of a pane that outlives copies of the Model: the process,
// the pseudo-terminal and the emulator.
type session struct {
	mu      sync.Mutex
	term    *vt.Terminal
	master  *os.File
	cmd     *exec.Cmd
	started bool
	closed  bool

	notify chan struct{} // holds one token while there is output not yet reported
	done   chan struct{} // closed when the program has exited and its output is read
	code   int
	err    error
}

func newSession() *session {
	return &session{notify: make(chan struct{}, 1), done: make(chan struct{})}
}

func (s *session) wake() {
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

// size returns the terminal's size, falling back to the defaults.
func (m Model) size() (cols, rows int) {
	cols, rows = m.Cols, m.Rows
	if cols <= 0 {
		cols = defaultCols
	}
	if rows <= 0 {
		rows = defaultRows
	}
	return min(cols, 1<<15), min(rows, 1<<15)
}

// env is the environment the program runs in.
func (m Model) env() []string {
	env := m.Env
	if env == nil {
		env = os.Environ()
	}
	out := append([]string(nil), env...)
	has := func(name string) bool {
		for _, e := range out {
			if len(e) > len(name) && e[:len(name)+1] == name+"=" {
				return true
			}
		}
		return false
	}
	if !has("TERM") {
		out = append(out, "TERM=xterm-256color")
	}
	if !has("COLORTERM") {
		out = append(out, "COLORTERM=truecolor")
	}
	return out
}

// start launches the program once.
func (s *session) start(m Model) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	s.started = true
	cols, rows := m.size()
	s.term = vt.New(cols, rows)
	switch {
	case m.Scrollback < 0:
		s.term.SetScrollback(0)
	case m.Scrollback > 0:
		s.term.SetScrollback(m.Scrollback)
	}
	if m.Command == "" {
		return errors.New("termpane: no command")
	}
	cmd := exec.Command(m.Command, m.Args...) // #nosec G204 -- running the program the application names is the point
	cmd.Dir = m.Dir
	cmd.Env = m.env()
	master, err := startProcess(cmd, cols, rows)
	if err != nil {
		return err
	}
	s.master, s.cmd = master, cmd
	var replies [][]byte
	s.term.SetReply(func(b []byte) { replies = append(replies, append([]byte(nil), b...)) })
	go s.read(&replies)
	return nil
}

// read copies the program's output into the emulator until the terminal closes,
// then collects the exit status.
func (s *session) read(replies *[][]byte) {
	buf := make([]byte, 32<<10)
	for {
		n, err := s.master.Read(buf)
		if n > 0 {
			s.mu.Lock()
			s.term.Write(buf[:n])
			out := *replies
			*replies = nil
			s.mu.Unlock()
			for _, r := range out { // answers go out unlocked: the program may be slow to read them
				_, _ = s.master.Write(r)
			}
			s.wake()
		}
		if err != nil {
			break
		}
	}
	werr := s.cmd.Wait()
	s.mu.Lock()
	s.code = 0
	var ee *exec.ExitError
	switch {
	case errors.As(werr, &ee):
		s.code = ee.ExitCode()
	case werr != nil && !errors.Is(werr, io.EOF):
		s.err = werr
	}
	s.mu.Unlock()
	close(s.done)
}

// write sends bytes to the program.
func (s *session) write(b string) {
	s.mu.Lock()
	master, closed := s.master, s.closed
	s.mu.Unlock()
	if master != nil && !closed && b != "" {
		_, _ = master.Write([]byte(b))
	}
}

// resize changes the size of the terminal and tells the program.
func (s *session) resize(cols, rows int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.term == nil {
		return
	}
	if c, r := s.term.Size(); c == cols && r == rows {
		return
	}
	s.term.Resize(cols, rows)
	if s.master != nil && !s.closed {
		_ = resizeTerminal(s.master, cols, rows)
	}
	s.wake()
}

// Init starts the program and asks for its output.
func (m Model) Init() tui.Cmd {
	if m.s == nil {
		return nil
	}
	if err := m.s.start(m); err != nil {
		id := m.ID
		close(m.s.done)
		return func() tui.Msg { return ExitedMsg{ID: id, Code: -1, Err: err} }
	}
	return m.listen()
}

// listen waits for the next thing to tell the model.
func (m Model) listen() tui.Cmd {
	s, id := m.s, m.ID
	return tui.FromCtx(func(ctx context.Context) tui.Msg {
		select {
		case <-ctx.Done():
			return nil
		case <-s.notify:
			return OutputMsg{ID: id}
		case <-s.done:
		}
		select {
		case <-s.notify: // the last output before the exit
			return OutputMsg{ID: id}
		default:
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		return ExitedMsg{ID: id, Code: s.code, Err: s.err}
	})
}

// SetSize resizes the terminal to cols x rows and tells the program. It does
// nothing when the size is unchanged.
func (m *Model) SetSize(cols, rows int) {
	cols, rows = min(max(cols, 1), 1<<15), min(max(rows, 1), 1<<15)
	m.Cols, m.Rows = cols, rows
	if m.s != nil {
		m.s.resize(cols, rows)
	}
}

// Size returns the terminal's size in cells.
func (m Model) Size() (cols, rows int) {
	if m.s != nil {
		m.s.mu.Lock()
		defer m.s.mu.Unlock()
		if m.s.term != nil {
			return m.s.term.Size()
		}
	}
	return m.size()
}

// Title is the title the program set, as plain text, or "".
func (m Model) Title() string {
	if m.s == nil {
		return ""
	}
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	if m.s.term == nil {
		return ""
	}
	return m.s.term.Title()
}

// Running reports whether the program has started and not yet exited.
func (m Model) Running() bool {
	if m.s == nil {
		return false
	}
	m.s.mu.Lock()
	started, closed := m.s.started, m.s.closed
	m.s.mu.Unlock()
	if !started || closed {
		return false
	}
	select {
	case <-m.s.done:
		return false
	default:
		return true
	}
}

// Close ends the program (SIGHUP, then SIGKILL) and releases the terminal. It is
// safe to call more than once and after the program has exited.
func (m Model) Close() error {
	s := m.s
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	master, cmd := s.master, s.cmd
	s.mu.Unlock()
	if master == nil {
		return nil
	}
	err := master.Close() // the child gets SIGHUP when its terminal closes
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return err
}

// Focus makes Update send keys and pastes to the program, and draws the cursor.
func (m *Model) Focus() { m.focused = true }

// Blur stops Update sending keys and pastes, and hides the cursor.
func (m *Model) Blur() { m.focused = false }

// Focused reports whether the pane has focus.
func (m Model) Focused() bool { return m.focused }

// Scrolled returns how many lines the view is scrolled back; 0 is the live
// screen.
func (m Model) Scrolled() int { return m.offset }

// Input sends text to the program as if typed. It is not sent through the key
// encoding: control characters go as they are.
func (m Model) Input(text string) {
	if m.s != nil {
		m.s.write(text)
	}
}

// Update sends keys, pastes and mouse events to the program and asks for its
// next output. It returns a Cmd that delivers OutputMsg and, at the end,
// ExitedMsg.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if m.s == nil {
		return m, nil
	}
	switch ev := msg.(type) {
	case OutputMsg:
		if ev.ID != m.ID {
			return m, nil
		}
		m.clampOffset()
		return m, m.listen()
	case tui.Key:
		return m.updateKey(ev), nil
	case tui.PasteEvent:
		if m.focused {
			m.offset = 0
			m.s.write(encodePaste(ev.Text, m.flag(func(t *vt.Terminal) bool { return t.BracketedPaste() })))
		}
	case tui.MouseEvent:
		return m.updateMouse(ev), nil
	}
	return m, nil
}

// flag reads a mode of the emulator.
func (m Model) flag(f func(*vt.Terminal) bool) bool {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	return m.s.term != nil && f(m.s.term)
}

func (m *Model) clampOffset() {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	if m.s.term == nil || m.s.term.AltScreen() {
		m.offset = 0
		return
	}
	m.offset = min(m.offset, m.s.term.Scrollback())
}

func (m Model) updateKey(k tui.Key) Model {
	if !m.focused {
		return m
	}
	km := m.keys()
	_, rows := m.Size()
	switch {
	case keymap.Matches(k, km.ScrollUp):
		m.offset += max(rows-1, 1)
		m.clampOffset()
		return m
	case keymap.Matches(k, km.ScrollDown):
		m.offset = max(m.offset-max(rows-1, 1), 0)
		return m
	case keymap.Matches(k, km.ScrollEnd):
		m.offset = 0
		return m
	}
	if k.Action == tui.KeyRelease {
		return m
	}
	m.offset = 0
	m.s.write(encodeKey(k, m.flag(func(t *vt.Terminal) bool { return t.AppCursorKeys() })))
	return m
}

func (m Model) updateMouse(ev tui.MouseEvent) Model {
	if !m.Mouse || m.Bounds.Empty() || !m.Bounds.Contains(ev.X, ev.Y) {
		return m
	}
	x, y := m.Bounds.Local(ev.X, ev.Y)
	m.s.mu.Lock()
	var mode vt.MouseMode
	var sgr, alt bool
	if m.s.term != nil {
		mode, sgr, alt = m.s.term.Mouse(), m.s.term.MouseSGR(), m.s.term.AltScreen()
	}
	m.s.mu.Unlock()
	if mode != vt.MouseNone && sgr {
		if seq := encodeMouse(ev, x, y, int(mode)); seq != "" {
			m.s.write(seq)
		}
		return m
	}
	if alt || ev.Action != tui.MouseActionPress {
		return m
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.offset += wheelStep
		m.clampOffset()
	case tui.MouseButtonWheelDown:
		m.offset = max(m.offset-wheelStep, 0)
	}
	return m
}

// Compile-time proofs that Model satisfies the widget contracts.
var (
	_ tui.Component[Model] = Model{}
	_ tui.Linearizer       = Model{}
)
